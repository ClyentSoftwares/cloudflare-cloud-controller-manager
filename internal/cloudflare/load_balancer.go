package cloudflare

import (
	"context"
	"fmt"

	cloudflareClient "github.com/ClyentSoftwares/cloudflare-cloud-controller-manager/pkg/cloudflare"
	"github.com/cloudflare/cloudflare-go"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

type loadBalancers struct {
	client *cloudflareClient.CloudflareAPI
	lbOps  *LoadBalancerOps
}

type LoadBalancerOps struct {
}

func newLoadbalancers(client *cloudflareClient.CloudflareAPI, lbOps *LoadBalancerOps) *loadBalancers {
	return &loadBalancers{
		client: client,
		lbOps:  lbOps,
	}

}

// GetLoadBalancer returns whether the specified load balancer exists, and
// if so, what its status is.
// Implementations must treat the *v1.Service parameter as read-only and not modify it.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (l *loadBalancers) GetLoadBalancer(ctx context.Context, clusterName string, service *v1.Service) (*v1.LoadBalancerStatus, bool, error) {

	hostName, err := GetLoadBalancerHostName(service)

	if err != nil {
		// If it doesn't have a hostname annotation then it isn't a load balancer
		// service that we want to manage
		return nil, false, nil
	}

	_, err = l.client.GetLoadBalancer(ctx, hostName)

	if err != nil {
		return nil, false, fmt.Errorf("failed to get load balancer by host name: %v", err)
	}

	status := &v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{{Hostname: hostName}},
	}

	return status, true, nil
}

// GetLoadBalancerName returns the name of the load balancer. Implementations must treat the
// *v1.Service parameter as read-only and not modify it.
func (l *loadBalancers) GetLoadBalancerName(ctx context.Context, clusterName string, service *v1.Service) string {
	hostName, err := GetLoadBalancerHostName(service)

	if err != nil {
		return ""
	}

	return hostName
}

// EnsureLoadBalancer creates a new load balancer 'name', or updates the existing one. Returns the status of the balancer
// Implementations must treat the *v1.Service and *v1.Node
// parameters as read-only and not modify them.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (l *loadBalancers) EnsureLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {

	// Get Annotations
	_, err := GetLoadBalancerHostName(service)
	if err != nil {
		// If it doesn't have a hostname annotation then it isn't a load balancer
		// service that we want to manage
		return &v1.LoadBalancerStatus{}, nil
	}

	// Verify LB monitor exists if not create
	monitor, err := l.createLoadBalancerMonitorIfNotExist(ctx, service)
	if err != nil {
		return nil, err
	}

	klog.Info("Verified monitor exists on cloudflare")

	// Verify LB pool exists if not create
	pool, err := l.createLoadBalancerPoolIfNotExist(ctx, monitor, service, nodes)
	if err != nil {
		return nil, err
	}

	klog.Info("Verified pool exists on cloudflare")

	// Verify LB pool exists if not create
	loadBalancer, err := l.createLoadBalancerIfNotExist(ctx, pool, service)
	if err != nil {
		return nil, err
	}

	klog.Info("Verified loadBalancer exists on cloudflare with hostname: ", loadBalancer.Name)

	status := &v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{{Hostname: loadBalancer.Name}},
	}
	return status, nil
}

// UpdateLoadBalancer updates hosts under the specified load balancer.
// Implementations must treat the *v1.Service and *v1.Node
// parameters as read-only and not modify them.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (l *loadBalancers) UpdateLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) error {

	_, err := GetLoadBalancerHostName(service)
	if err != nil {
		// If it doesn't have a hostname annotation then it isn't a load balancer
		// service that we want to manage
		return nil
	}

	// Verify LB monitor exists if not create
	monitor, err := l.createLoadBalancerMonitorIfNotExist(ctx, service)
	if err != nil {
		return err
	}

	_, err = l.updateLoadBalancerPool(ctx, monitor, service, nodes)

	return err
}

// EnsureLoadBalancerDeleted deletes the specified load balancer if it
// exists, returning nil if the load balancer specified either didn't exist or
// was successfully deleted.
// Implementations must treat the *v1.Service parameter as read-only and not modify it.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (l *loadBalancers) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	if service.Spec.LoadBalancerClass != nil {
		return nil
	}

	_, err := GetLoadBalancerHostName(service)

	if err != nil {
		// If it doesn't have a hostname annotation then it isn't a load balancer
		// service that we want to manage
		return nil
	}

	return l.deleteLoadBalancer(ctx, service)
}

func (l *loadBalancers) getLoadBalancerPoolName(service *v1.Service) (string, error) {
	hostName, err := GetLoadBalancerHostName(service)

	if err != nil {
		return "", fmt.Errorf("failed to get load balancer host name: %v", err)
	}

	return l.client.FormatResourceName(hostName + "-pool"), nil
}

func (l *loadBalancers) getLoadBalancerMonitorName(service *v1.Service) (string, error) {
	hostName, err := GetLoadBalancerHostName(service)

	if err != nil {
		return "", fmt.Errorf("failed to get load balancer host name: %v", err)
	}

	return l.client.FormatResourceName(hostName + "-monitor"), nil
}

// createLoadBalancerMonitorIfNotExist will check with the cloudflare API that the monitor exists
// if not it will create a new one using the service config
func (l *loadBalancers) createLoadBalancerMonitorIfNotExist(ctx context.Context, service *v1.Service) (cloudflare.LoadBalancerMonitor, error) {

	monitorName, _ := l.getLoadBalancerMonitorName(service) // Ignore err as it has already been checked
	monitor, err := l.client.GetLoadBalancerMonitor(ctx, monitorName)

	if err != nil {

		klog.Info("Creating LB monitor")

		monitorPath, _ := GetLoadBalancerMonitorPath(service)
		monitorAllowInsecure, _ := GetLoadBalancerMonitorAllowInsecure(service)
		monitorType, _ := GetLoadBalancerMonitorType(service)

		// Try creating a new load balancer monitor
		monitor, err = l.client.CreateLoadBalancerMonitor(ctx, cloudflare.LoadBalancerMonitor{
			Description:     monitorName,
			Type:            monitorType,
			Method:          "GET",
			Path:            monitorPath,
			Port:            uint16(service.Spec.Ports[0].Port), //TODO get additional ports
			ExpectedCodes:   "2xx",
			Interval:        60,
			Timeout:         5,
			Retries:         2,
			FollowRedirects: true,
			AllowInsecure:   monitorAllowInsecure,
		})

		return monitor, err
	}

	return monitor, nil
}

// createLoadBalancerPoolIfNotExist will check with the cloudflare API that the pool exists
// if not it will create a new one using the service config
func (l *loadBalancers) createLoadBalancerPoolIfNotExist(ctx context.Context, monitor cloudflare.LoadBalancerMonitor, service *v1.Service, nodes []*v1.Node) (cloudflare.LoadBalancerPool, error) {

	poolName, _ := l.getLoadBalancerPoolName(service) // Ignore err as it has already been checked
	_, err := l.client.GetLoadBalancerPool(ctx, poolName)

	if err != nil {

		klog.Info("LB Pool does not exist - creating a new pool")

		config := cloudflare.LoadBalancerPool{
			Name:    poolName,
			Monitor: monitor.ID,
			Origins: []cloudflare.LoadBalancerOrigin{},
			Enabled: true,
		}

		for _, node := range nodes {

			ip, err := GetNodeExternalIP(node)

			if err != nil {
				return cloudflare.LoadBalancerPool{}, err
			}

			origin := cloudflare.LoadBalancerOrigin{
				Name:    l.client.FormatResourceName(ip),
				Address: ip,
				Enabled: true,
				Weight:  1,
			}

			config.Origins = append(config.Origins, origin)
		}

		// Try creating a new load balancer pool
		pool, err := l.client.CreateLoadBalancerPool(ctx, config)

		return pool, err
	}

	return l.updateLoadBalancerPool(ctx, monitor, service, nodes)
}

// createLoadBalancerPoolIfNotExist will check with the cloudflare API that the pool exists
// if not it will create a new one using the service config
func (l *loadBalancers) updateLoadBalancerPool(ctx context.Context, monitor cloudflare.LoadBalancerMonitor, service *v1.Service, nodes []*v1.Node) (cloudflare.LoadBalancerPool, error) {

	klog.Info("Trying to update load balancer pool for ", len(nodes), " nodes")

	poolName, _ := l.getLoadBalancerPoolName(service) // Ignore err as it has already been checked
	pool, err := l.client.GetLoadBalancerPool(ctx, poolName)

	if err != nil {
		return cloudflare.LoadBalancerPool{}, err
	}

	config := cloudflare.LoadBalancerPool{
		ID:      pool.ID,
		Name:    pool.Name,
		Monitor: monitor.ID,
		Origins: []cloudflare.LoadBalancerOrigin{},
		Enabled: true,
	}

	for _, node := range nodes {

		ip, err := GetNodeExternalIP(node)

		if err != nil {
			klog.Warning(err) // Likely the node doesn't have an external ip so skip but log
			continue
		}

		origin := cloudflare.LoadBalancerOrigin{
			Name:    l.client.FormatResourceName(ip),
			Address: ip,
			Enabled: true,
			Weight:  1,
		}

		config.Origins = append(config.Origins, origin)
	}

	klog.Info("Updating LB pool with config: ", config)

	return l.client.UpdateLoadBalancerPool(ctx, config)

}

// createLoadBalancerIfNotExist will check with the cloudflare API that the load balancer exists
// if not it will create a new one using the service config
func (l *loadBalancers) createLoadBalancerIfNotExist(ctx context.Context, pool cloudflare.LoadBalancerPool, service *v1.Service) (cloudflare.LoadBalancer, error) {

	hostName, _ := GetLoadBalancerHostName(service) // Ignore err as it has already been checked
	loadBalancer, err := l.client.GetLoadBalancer(ctx, hostName)

	if err != nil {
		klog.Info("Creating LB")

		// Try creating a new load balancer
		loadBalancer, err := l.client.CreateLoadBalancer(ctx, cloudflare.LoadBalancer{
			Name:         l.client.FormatResourceName(hostName),
			FallbackPool: pool.ID,
			DefaultPools: []string{pool.ID},
			TTL:          30,
			Proxied:      true,
		})

		return loadBalancer, err
	}

	return loadBalancer, nil
}

// deleteLoadBalancer will delete a load balancer and its related origin pools and monitors
func (l *loadBalancers) deleteLoadBalancer(ctx context.Context, service *v1.Service) error {

	// Delete Load Balancer First
	hostName, _ := GetLoadBalancerHostName(service) // Ignore err as it has already been checked
	err := l.client.DeleteLoadBalancer(ctx, hostName)
	if err != nil {
		return err
	}

	klog.Info("Deleted Load Balancer: ", hostName)

	// Delete Load Balancer pool
	poolName, err := l.getLoadBalancerPoolName(service)
	if err != nil {
		return err
	}

	err = l.client.DeleteLoadBalancerPool(ctx, poolName)
	if err != nil {
		return err
	}

	klog.Info("Deleted Load Balancer Pool: ", poolName)

	// Delete Load Balancer monitor last
	monitorName, err := l.getLoadBalancerMonitorName(service)
	if err != nil {
		return err
	}

	err = l.client.DeleteLoadBalancerMonitor(ctx, monitorName)
	if err != nil {
		return err
	}

	klog.Info("Deleted Load Balancer Monitor: ", hostName)

	return nil
}
