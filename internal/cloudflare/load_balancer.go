package cloudflare

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/ClyentSoftwares/cloudflare-cloud-controller-manager/internal/cfapi"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

// LBClient is the cloudflare API surface used by the load balancer controller
type LBClient interface {
	GetLoadBalancer(ctx context.Context, name string) (cfapi.LoadBalancer, error)
	CreateLoadBalancer(ctx context.Context, lb cfapi.LoadBalancer) (cfapi.LoadBalancer, error)
	UpdateLoadBalancer(ctx context.Context, lb cfapi.LoadBalancer) (cfapi.LoadBalancer, error)
	DeleteLoadBalancer(ctx context.Context, name string) error

	GetPool(ctx context.Context, name string) (cfapi.Pool, error)
	CreatePool(ctx context.Context, pool cfapi.Pool) (cfapi.Pool, error)
	UpdatePool(ctx context.Context, pool cfapi.Pool) (cfapi.Pool, error)
	DeletePool(ctx context.Context, name string) error

	GetMonitor(ctx context.Context, description string) (cfapi.Monitor, error)
	CreateMonitor(ctx context.Context, monitor cfapi.Monitor) (cfapi.Monitor, error)
	UpdateMonitor(ctx context.Context, monitor cfapi.Monitor) (cfapi.Monitor, error)
	DeleteMonitor(ctx context.Context, description string) error
}

var _ LBClient = (*cfapi.Client)(nil)

type loadBalancers struct {
	client LBClient
}

func newLoadBalancers(client LBClient) *loadBalancers {
	return &loadBalancers{client: client}
}

// GetLoadBalancer returns whether the specified load balancer exists, and
// if so, what its status is.
// Implementations must treat the *v1.Service parameter as read-only and not modify it.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (l *loadBalancers) GetLoadBalancer(ctx context.Context, _ string, service *v1.Service) (*v1.LoadBalancerStatus, bool, error) {
	hostname, ok := hostnameFromService(service)
	if !ok {
		return nil, false, nil
	}

	lb, err := l.client.GetLoadBalancer(ctx, loadBalancerName(hostname))
	if errors.Is(err, cfapi.ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("getting load balancer for service %s/%s: %w", service.Namespace, service.Name, err)
	}

	return loadBalancerStatus(lb), true, nil
}

// GetLoadBalancerName returns the name of the load balancer. Implementations must treat the
// *v1.Service parameter as read-only and not modify it.
func (l *loadBalancers) GetLoadBalancerName(_ context.Context, _ string, service *v1.Service) string {
	hostname, ok := hostnameFromService(service)
	if !ok {
		return ""
	}
	return loadBalancerName(hostname)
}

// EnsureLoadBalancer creates a new load balancer 'name', or updates the existing one. Returns the status of the balancer
// Implementations must treat the *v1.Service and *v1.Node
// parameters as read-only and not modify them.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (l *loadBalancers) EnsureLoadBalancer(ctx context.Context, _ string, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
	hostname, ok := hostnameFromService(service)
	if !ok {
		return &v1.LoadBalancerStatus{}, nil
	}

	lb, err := l.reconcile(ctx, hostname, service, nodes)
	if err != nil {
		return nil, err
	}
	return loadBalancerStatus(lb), nil
}

// UpdateLoadBalancer updates hosts under the specified load balancer.
// Implementations must treat the *v1.Service and *v1.Node
// parameters as read-only and not modify them.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (l *loadBalancers) UpdateLoadBalancer(ctx context.Context, _ string, service *v1.Service, nodes []*v1.Node) error {
	hostname, ok := hostnameFromService(service)
	if !ok {
		return nil
	}

	_, err := l.reconcile(ctx, hostname, service, nodes)
	return err
}

// EnsureLoadBalancerDeleted deletes the specified load balancer if it
// exists, returning nil if the load balancer specified either didn't exist or
// was successfully deleted.
// Implementations must treat the *v1.Service parameter as read-only and not modify it.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (l *loadBalancers) EnsureLoadBalancerDeleted(ctx context.Context, _ string, service *v1.Service) error {
	hostname, ok := hostnameFromService(service)
	if !ok {
		return nil
	}

	// Delete Load Balancer first as it references the pool
	if err := l.client.DeleteLoadBalancer(ctx, loadBalancerName(hostname)); err != nil {
		return fmt.Errorf("deleting load balancer for service %s/%s: %w", service.Namespace, service.Name, err)
	}
	if err := l.client.DeletePool(ctx, poolName(hostname)); err != nil {
		return fmt.Errorf("deleting pool for service %s/%s: %w", service.Namespace, service.Name, err)
	}
	if err := l.client.DeleteMonitor(ctx, monitorName(hostname)); err != nil {
		return fmt.Errorf("deleting monitor for service %s/%s: %w", service.Namespace, service.Name, err)
	}

	klog.InfoS("deleted cloudflare load balancer resources", "service", klog.KObj(service), "hostname", hostname)
	return nil
}

// reconcile will drive the monitor, pool and load balancer to the desired
// state in dependency order
func (l *loadBalancers) reconcile(ctx context.Context, hostname string, service *v1.Service, nodes []*v1.Node) (cfapi.LoadBalancer, error) {
	monitor, err := l.reconcileMonitor(ctx, hostname, service)
	if err != nil {
		return cfapi.LoadBalancer{}, err
	}

	pool, err := l.reconcilePool(ctx, hostname, monitor.ID, nodes)
	if err != nil {
		return cfapi.LoadBalancer{}, err
	}

	return l.reconcileLoadBalancer(ctx, hostname, pool.ID, service)
}

// reconcileMonitor will check with the cloudflare API that the monitor exists
// and matches the service config, if not it will create or update it
func (l *loadBalancers) reconcileMonitor(ctx context.Context, hostname string, service *v1.Service) (cfapi.Monitor, error) {
	desired, err := desiredMonitor(service, hostname)
	if err != nil {
		return cfapi.Monitor{}, err
	}

	existing, err := l.client.GetMonitor(ctx, desired.Description)
	if errors.Is(err, cfapi.ErrNotFound) {
		klog.InfoS("creating cloudflare monitor", "service", klog.KObj(service), "monitor", desired.Description)
		return l.client.CreateMonitor(ctx, desired)
	}
	if err != nil {
		return cfapi.Monitor{}, err
	}

	if monitorNeedsUpdate(existing, desired) {
		desired.ID = existing.ID
		klog.InfoS("updating cloudflare monitor", "service", klog.KObj(service), "monitor", desired.Description)
		return l.client.UpdateMonitor(ctx, desired)
	}
	return existing, nil
}

// reconcilePool will check with the cloudflare API that the pool exists and
// matches the current nodes, if not it will create or update it
func (l *loadBalancers) reconcilePool(ctx context.Context, hostname, monitorID string, nodes []*v1.Node) (cfapi.Pool, error) {
	origins, err := desiredOrigins(nodes)
	if err != nil {
		return cfapi.Pool{}, err
	}

	desired := cfapi.Pool{
		Name:      poolName(hostname),
		MonitorID: monitorID,
		Enabled:   true,
		Origins:   origins,
	}

	existing, err := l.client.GetPool(ctx, desired.Name)
	if errors.Is(err, cfapi.ErrNotFound) {
		klog.InfoS("creating cloudflare pool", "pool", desired.Name, "origins", len(origins))
		return l.client.CreatePool(ctx, desired)
	}
	if err != nil {
		return cfapi.Pool{}, err
	}

	if poolNeedsUpdate(existing, desired) {
		desired.ID = existing.ID
		klog.InfoS("updating cloudflare pool", "pool", desired.Name, "origins", len(origins))
		return l.client.UpdatePool(ctx, desired)
	}
	return existing, nil
}

// reconcileLoadBalancer will check with the cloudflare API that the load
// balancer exists and references the pool, if not it will create or update it
func (l *loadBalancers) reconcileLoadBalancer(ctx context.Context, hostname, poolID string, service *v1.Service) (cfapi.LoadBalancer, error) {
	desired := cfapi.LoadBalancer{
		Name:           loadBalancerName(hostname),
		FallbackPoolID: poolID,
		DefaultPoolIDs: []string{poolID},
		Proxied:        true,
	}

	existing, err := l.client.GetLoadBalancer(ctx, desired.Name)
	if errors.Is(err, cfapi.ErrNotFound) {
		klog.InfoS("creating cloudflare load balancer", "service", klog.KObj(service), "loadBalancer", desired.Name)
		return l.client.CreateLoadBalancer(ctx, desired)
	}
	if err != nil {
		return cfapi.LoadBalancer{}, err
	}

	if loadBalancerNeedsUpdate(existing, desired) {
		desired.ID = existing.ID
		klog.InfoS("updating cloudflare load balancer", "service", klog.KObj(service), "loadBalancer", desired.Name)
		return l.client.UpdateLoadBalancer(ctx, desired)
	}
	return existing, nil
}

// desiredMonitor builds the monitor config from the service annotations
func desiredMonitor(service *v1.Service, hostname string) (cfapi.Monitor, error) {
	if len(service.Spec.Ports) == 0 {
		return cfapi.Monitor{}, fmt.Errorf("service %s/%s has no ports to monitor", service.Namespace, service.Name)
	}
	if len(service.Spec.Ports) > 1 {
		klog.InfoS("service has multiple ports; only the first is used for the cloudflare monitor",
			"service", klog.KObj(service), "port", service.Spec.Ports[0].Port)
	}

	var errs []error
	allowInsecure, err := boolAnnotation(service, annotationMonitorAllowInsecure, false)
	if err != nil {
		errs = append(errs, err)
	}
	interval, err := intAnnotation(service, annotationMonitorInterval, defaultMonitorInterval)
	if err != nil {
		errs = append(errs, err)
	}
	timeout, err := intAnnotation(service, annotationMonitorTimeout, defaultMonitorTimeout)
	if err != nil {
		errs = append(errs, err)
	}
	retries, err := intAnnotation(service, annotationMonitorRetries, defaultMonitorRetries)
	if err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return cfapi.Monitor{}, errors.Join(errs...)
	}

	var header map[string][]string
	if hostHeader := stringAnnotation(service, annotationMonitorHeader, ""); hostHeader != "" {
		header = map[string][]string{"Host": {hostHeader}}
	}

	return cfapi.Monitor{
		Description:     monitorName(hostname),
		Type:            stringAnnotation(service, annotationMonitorType, defaultMonitorType),
		Method:          stringAnnotation(service, annotationMonitorMethod, defaultMonitorMethod),
		Path:            stringAnnotation(service, annotationMonitorPath, defaultMonitorPath),
		Port:            int64(service.Spec.Ports[0].Port),
		ExpectedCodes:   stringAnnotation(service, annotationMonitorExpectedCodes, defaultMonitorExpectedCodes),
		Interval:        interval,
		Timeout:         timeout,
		Retries:         retries,
		FollowRedirects: true,
		AllowInsecure:   allowInsecure,
		ProbeZone:       stringAnnotation(service, annotationMonitorProbeZone, ""),
		Header:          header,
	}, nil
}

func monitorNeedsUpdate(existing, desired cfapi.Monitor) bool {
	return existing.Type != desired.Type ||
		existing.Method != desired.Method ||
		existing.Path != desired.Path ||
		existing.Port != desired.Port ||
		existing.ExpectedCodes != desired.ExpectedCodes ||
		existing.Interval != desired.Interval ||
		existing.Timeout != desired.Timeout ||
		existing.Retries != desired.Retries ||
		existing.FollowRedirects != desired.FollowRedirects ||
		existing.AllowInsecure != desired.AllowInsecure ||
		existing.ProbeZone != desired.ProbeZone ||
		!slices.Equal(existing.Header["Host"], desired.Header["Host"])
}

func poolNeedsUpdate(existing, desired cfapi.Pool) bool {
	return existing.MonitorID != desired.MonitorID ||
		existing.Enabled != desired.Enabled ||
		!originsEqual(existing.Origins, desired.Origins)
}

func originsEqual(a, b []cfapi.Origin) bool {
	if len(a) != len(b) {
		return false
	}
	byName := func(x, y cfapi.Origin) int {
		return strings.Compare(x.Name, y.Name)
	}
	as := slices.Clone(a)
	bs := slices.Clone(b)
	slices.SortFunc(as, byName)
	slices.SortFunc(bs, byName)
	return slices.Equal(as, bs)
}

func loadBalancerNeedsUpdate(existing, desired cfapi.LoadBalancer) bool {
	return existing.FallbackPoolID != desired.FallbackPoolID ||
		existing.Proxied != desired.Proxied ||
		!slices.Equal(existing.DefaultPoolIDs, desired.DefaultPoolIDs)
}

func loadBalancerStatus(lb cfapi.LoadBalancer) *v1.LoadBalancerStatus {
	return &v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{{Hostname: lb.Name}},
	}
}
