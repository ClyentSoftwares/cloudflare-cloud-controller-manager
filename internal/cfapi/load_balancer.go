package cfapi

import (
	"context"
	"errors"
	"fmt"

	cloudflare "github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/load_balancers"
)

func loadBalancerFromSDK(lb load_balancers.LoadBalancer) LoadBalancer {
	return LoadBalancer{
		ID:             lb.ID,
		Name:           lb.Name,
		FallbackPoolID: lb.FallbackPool,
		DefaultPoolIDs: lb.DefaultPools,
		Proxied:        lb.Proxied,
	}
}

// GetLoadBalancer retrieves a load balancer by name for the configured zone.
func (c *Client) GetLoadBalancer(ctx context.Context, name string) (LoadBalancer, error) {
	iter := c.api.LoadBalancers.ListAutoPaging(ctx, load_balancers.LoadBalancerListParams{
		ZoneID: cloudflare.F(c.zoneID),
	})
	for iter.Next() {
		if lb := iter.Current(); lb.Name == name {
			return loadBalancerFromSDK(lb), nil
		}
	}
	if err := iter.Err(); err != nil {
		return LoadBalancer{}, wrapAPIError(err, "listing load balancers")
	}
	return LoadBalancer{}, fmt.Errorf("load balancer %q: %w", name, ErrNotFound)
}

// CreateLoadBalancer creates a new load balancer.
func (c *Client) CreateLoadBalancer(ctx context.Context, lb LoadBalancer) (LoadBalancer, error) {
	created, err := c.api.LoadBalancers.New(ctx, load_balancers.LoadBalancerNewParams{
		ZoneID:       cloudflare.F(c.zoneID),
		Name:         cloudflare.F(lb.Name),
		DefaultPools: cloudflare.F(lb.DefaultPoolIDs),
		FallbackPool: cloudflare.F(lb.FallbackPoolID),
		Proxied:      cloudflare.F(lb.Proxied),
	})
	if err != nil {
		return LoadBalancer{}, wrapAPIError(err, fmt.Sprintf("creating load balancer %q", lb.Name))
	}
	c.log.Info("created load balancer", "name", lb.Name, "loadBalancerID", created.ID)
	return loadBalancerFromSDK(*created), nil
}

// UpdateLoadBalancer updates an existing load balancer.
func (c *Client) UpdateLoadBalancer(ctx context.Context, lb LoadBalancer) (LoadBalancer, error) {
	updated, err := c.api.LoadBalancers.Update(ctx, lb.ID, load_balancers.LoadBalancerUpdateParams{
		ZoneID:       cloudflare.F(c.zoneID),
		Name:         cloudflare.F(lb.Name),
		DefaultPools: cloudflare.F(lb.DefaultPoolIDs),
		FallbackPool: cloudflare.F(lb.FallbackPoolID),
		Proxied:      cloudflare.F(lb.Proxied),
	})
	if err != nil {
		return LoadBalancer{}, wrapAPIError(err, fmt.Sprintf("updating load balancer %q", lb.Name))
	}
	c.log.Info("updated load balancer", "name", lb.Name, "loadBalancerID", lb.ID)
	return loadBalancerFromSDK(*updated), nil
}

// DeleteLoadBalancer deletes a load balancer by name, ignoring load balancers
// that are already gone.
func (c *Client) DeleteLoadBalancer(ctx context.Context, name string) error {
	lb, err := c.GetLoadBalancer(ctx, name)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	_, err = c.api.LoadBalancers.Delete(ctx, lb.ID, load_balancers.LoadBalancerDeleteParams{
		ZoneID: cloudflare.F(c.zoneID),
	})
	if err != nil {
		if isNotFoundAPIError(err) {
			return nil
		}
		return wrapAPIError(err, fmt.Sprintf("deleting load balancer %q", name))
	}
	c.log.Info("deleted load balancer", "name", name, "loadBalancerID", lb.ID)
	return nil
}
