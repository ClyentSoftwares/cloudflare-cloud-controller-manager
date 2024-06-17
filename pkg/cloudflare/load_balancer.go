package cloudflare

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// retrieves a load balancer by name for a given zone ID.
func (c *CloudflareAPI) GetLoadBalancer(ctx context.Context, name string) (cloudflare.LoadBalancer, error) {

	lbs, err := c.CloudflareClient.ListLoadBalancers(ctx, cloudflare.ZoneIdentifier(c.ZoneId), cloudflare.ListLoadBalancerParams{})
	if err != nil {
		c.Log.Error(err, "error listing load balancers", "zoneID", c.ZoneId)
		return cloudflare.LoadBalancer{}, fmt.Errorf("error listing load balancers: %w", err)
	}

	for _, lb := range lbs {
		if lb.Name == name {
			return lb, nil
		}
	}

	return cloudflare.LoadBalancer{}, fmt.Errorf("failed to get load balancer by name: %v", name)
}

// creates a new load balancer for a given zone ID.
func (c *CloudflareAPI) CreateLoadBalancer(ctx context.Context, loadBalancer cloudflare.LoadBalancer) (cloudflare.LoadBalancer, error) {

	params := cloudflare.CreateLoadBalancerParams{
		LoadBalancer: loadBalancer,
	}

	response, err := c.CloudflareClient.CreateLoadBalancer(ctx, cloudflare.ZoneIdentifier(c.ZoneId), params)

	if err != nil {
		c.Log.Error(err, "error creating load balancer", "zoneID", c.ZoneId, "name", loadBalancer.Name)
		return cloudflare.LoadBalancer{}, fmt.Errorf("error creating load balancer: %w", err)
	}

	c.Log.Info("load balancer created successfully", "zoneID", c.ZoneId, "name", loadBalancer.Name, "response", response)
	return response, nil
}

// delete a load balancer by name.
func (c *CloudflareAPI) DeleteLoadBalancer(ctx context.Context, name string) error {

	lb, err := c.GetLoadBalancer(ctx, name)
	if err != nil {
		return err
	}

	err = c.CloudflareClient.DeleteLoadBalancer(ctx, cloudflare.ZoneIdentifier(c.ZoneId), lb.ID)

	return err
}

// gets a pool by name.
func (c *CloudflareAPI) GetLoadBalancerPool(ctx context.Context, poolName string) (cloudflare.LoadBalancerPool, error) {

	pools, err := c.CloudflareClient.ListLoadBalancerPools(ctx, cloudflare.AccountIdentifier(c.AccountId), cloudflare.ListLoadBalancerPoolParams{})
	if err != nil {
		c.Log.Error(err, "error listing load balancer pools")
		return cloudflare.LoadBalancerPool{}, err
	}

	for _, pool := range pools {
		if pool.Name == poolName {
			return pool, nil
		}
	}

	return cloudflare.LoadBalancerPool{}, fmt.Errorf("failed to get load balancer pool by name: %v", poolName)
}

// creates a new pool.
func (c *CloudflareAPI) CreateLoadBalancerPool(ctx context.Context, loadBalancerPool cloudflare.LoadBalancerPool) (cloudflare.LoadBalancerPool, error) {
	params := cloudflare.CreateLoadBalancerPoolParams{
		LoadBalancerPool: loadBalancerPool,
	}

	pool, err := c.CloudflareClient.CreateLoadBalancerPool(ctx, cloudflare.AccountIdentifier(c.AccountId), params)
	if err != nil {
		c.Log.Error(err, "error creating load balancer pool")
		return cloudflare.LoadBalancerPool{}, err
	}

	return pool, nil
}

// update an existing pool.
func (c *CloudflareAPI) UpdateLoadBalancerPool(ctx context.Context, loadBalancerPool cloudflare.LoadBalancerPool) (cloudflare.LoadBalancerPool, error) {
	params := cloudflare.UpdateLoadBalancerPoolParams{
		LoadBalancer: loadBalancerPool,
	}

	pool, err := c.CloudflareClient.UpdateLoadBalancerPool(ctx, cloudflare.AccountIdentifier(c.AccountId), params)
	if err != nil {
		c.Log.Error(err, "error updating load balancer pool")
		return cloudflare.LoadBalancerPool{}, err
	}

	return pool, nil
}

// delete a pool by name.
func (c *CloudflareAPI) DeleteLoadBalancerPool(ctx context.Context, poolName string) error {
	pool, err := c.GetLoadBalancerPool(ctx, poolName)
	if err != nil {
		return err
	}

	err = c.CloudflareClient.DeleteLoadBalancerPool(ctx, cloudflare.AccountIdentifier(c.AccountId), pool.ID)

	return err
}

// gets the configuration of an existing pool.
func (c *CloudflareAPI) GetPoolConfiguration(ctx context.Context, poolId string) (cloudflare.LoadBalancerPool, error) {

	pool, err := c.CloudflareClient.GetLoadBalancerPool(ctx, cloudflare.AccountIdentifier(c.AccountId), poolId)
	if err != nil {
		c.Log.Error(err, "error fetching load balancer pool", "poolId", poolId)
		return cloudflare.LoadBalancerPool{}, err
	}

	return pool, nil
}

// updates the configuration of an existing pool.
func (c *CloudflareAPI) UpdatePoolConfiguration(ctx context.Context, loadBalancerPool cloudflare.LoadBalancerPool) error {

	params := cloudflare.UpdateLoadBalancerPoolParams{
		LoadBalancer: loadBalancerPool,
	}

	_, err := c.CloudflareClient.UpdateLoadBalancerPool(ctx, cloudflare.AccountIdentifier(c.AccountId), params)
	if err != nil {
		c.Log.Error(err, "error updating load balancer pool", "poolId", loadBalancerPool.ID)
		return err
	}

	return nil
}

// retrieves a health monitor by name.
func (c *CloudflareAPI) GetLoadBalancerMonitor(ctx context.Context, monitorName string) (cloudflare.LoadBalancerMonitor, error) {

	monitors, err := c.CloudflareClient.ListLoadBalancerMonitors(ctx, cloudflare.AccountIdentifier(c.AccountId), cloudflare.ListLoadBalancerMonitorParams{})
	if err != nil {
		c.Log.Error(err, "error listing load balancer monitors")
		return cloudflare.LoadBalancerMonitor{}, err
	}

	for _, monitor := range monitors {
		if monitor.Description == monitorName {
			return monitor, nil
		}
	}

	return cloudflare.LoadBalancerMonitor{}, fmt.Errorf("failed to get load balancer monitor by name: %v", monitorName)
}

// creates a new health monitor.
func (c *CloudflareAPI) CreateLoadBalancerMonitor(ctx context.Context, monitor cloudflare.LoadBalancerMonitor) (cloudflare.LoadBalancerMonitor, error) {
	params := cloudflare.CreateLoadBalancerMonitorParams{
		LoadBalancerMonitor: monitor,
	}

	monitor, err := c.CloudflareClient.CreateLoadBalancerMonitor(ctx, cloudflare.AccountIdentifier(c.AccountId), params)
	if err != nil {
		c.Log.Error(err, "error creating load balancer monitor")
		return cloudflare.LoadBalancerMonitor{}, err
	}

	return monitor, nil
}

// updates an existing health monitor.
func (c *CloudflareAPI) UpdateLoadBalancerMonitor(ctx context.Context, monitor cloudflare.LoadBalancerMonitor) (cloudflare.LoadBalancerMonitor, error) {
	params := cloudflare.UpdateLoadBalancerMonitorParams{
		LoadBalancerMonitor: monitor,
	}

	monitor, err := c.CloudflareClient.UpdateLoadBalancerMonitor(ctx, cloudflare.AccountIdentifier(c.AccountId), params)
	if err != nil {
		c.Log.Error(err, "error creating load balancer monitor", "monitorId", monitor.ID)
		return cloudflare.LoadBalancerMonitor{}, err
	}

	return monitor, nil
}

// delete a load balancer monitor by name.
func (c *CloudflareAPI) DeleteLoadBalancerMonitor(ctx context.Context, name string) error {

	monitor, err := c.GetLoadBalancerMonitor(ctx, name)
	if err != nil {
		return err
	}

	err = c.CloudflareClient.DeleteLoadBalancerMonitor(ctx, cloudflare.AccountIdentifier(c.AccountId), monitor.ID)

	return err
}
