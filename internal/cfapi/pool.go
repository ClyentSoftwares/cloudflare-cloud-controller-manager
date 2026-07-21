package cfapi

import (
	"context"
	"errors"
	"fmt"

	cloudflare "github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/load_balancers"
)

func poolFromSDK(p load_balancers.Pool) Pool {
	origins := make([]Origin, 0, len(p.Origins))
	for _, o := range p.Origins {
		origins = append(origins, Origin{
			Name:    o.Name,
			Address: o.Address,
			Enabled: o.Enabled,
			Weight:  o.Weight,
		})
	}
	return Pool{
		ID:        p.ID,
		Name:      p.Name,
		MonitorID: p.Monitor,
		Enabled:   p.Enabled,
		Origins:   origins,
	}
}

func originParams(origins []Origin) []load_balancers.OriginParam {
	params := make([]load_balancers.OriginParam, 0, len(origins))
	for _, o := range origins {
		params = append(params, load_balancers.OriginParam{
			Name:    cloudflare.F(o.Name),
			Address: cloudflare.F(o.Address),
			Enabled: cloudflare.F(o.Enabled),
			Weight:  cloudflare.F(o.Weight),
		})
	}
	return params
}

// GetPool gets a pool by name.
func (c *Client) GetPool(ctx context.Context, name string) (Pool, error) {
	iter := c.api.LoadBalancers.Pools.ListAutoPaging(ctx, load_balancers.PoolListParams{
		AccountID: cloudflare.F(c.accountID),
	})
	for iter.Next() {
		if p := iter.Current(); p.Name == name {
			return poolFromSDK(p), nil
		}
	}
	if err := iter.Err(); err != nil {
		return Pool{}, wrapAPIError(err, "listing load balancer pools")
	}
	return Pool{}, fmt.Errorf("pool %q: %w", name, ErrNotFound)
}

// CreatePool creates a new pool.
func (c *Client) CreatePool(ctx context.Context, pool Pool) (Pool, error) {
	created, err := c.api.LoadBalancers.Pools.New(ctx, load_balancers.PoolNewParams{
		AccountID: cloudflare.F(c.accountID),
		Name:      cloudflare.F(pool.Name),
		Origins:   cloudflare.F(originParams(pool.Origins)),
		Monitor:   cloudflare.F(pool.MonitorID),
		Enabled:   cloudflare.F(pool.Enabled),
	})
	if err != nil {
		return Pool{}, wrapAPIError(err, fmt.Sprintf("creating pool %q", pool.Name))
	}
	c.log.Info("created load balancer pool", "name", pool.Name, "poolID", created.ID, "origins", len(pool.Origins))
	return poolFromSDK(*created), nil
}

// UpdatePool updates an existing pool.
func (c *Client) UpdatePool(ctx context.Context, pool Pool) (Pool, error) {
	updated, err := c.api.LoadBalancers.Pools.Update(ctx, pool.ID, load_balancers.PoolUpdateParams{
		AccountID: cloudflare.F(c.accountID),
		Name:      cloudflare.F(pool.Name),
		Origins:   cloudflare.F(originParams(pool.Origins)),
		Monitor:   cloudflare.F(pool.MonitorID),
		Enabled:   cloudflare.F(pool.Enabled),
	})
	if err != nil {
		return Pool{}, wrapAPIError(err, fmt.Sprintf("updating pool %q", pool.Name))
	}
	c.log.Info("updated load balancer pool", "name", pool.Name, "poolID", pool.ID, "origins", len(pool.Origins))
	return poolFromSDK(*updated), nil
}

// DeletePool deletes a pool by name, ignoring pools that are already gone.
func (c *Client) DeletePool(ctx context.Context, name string) error {
	pool, err := c.GetPool(ctx, name)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	_, err = c.api.LoadBalancers.Pools.Delete(ctx, pool.ID, load_balancers.PoolDeleteParams{
		AccountID: cloudflare.F(c.accountID),
	})
	if err != nil {
		if isNotFoundAPIError(err) {
			return nil
		}
		return wrapAPIError(err, fmt.Sprintf("deleting pool %q", name))
	}
	c.log.Info("deleted load balancer pool", "name", name, "poolID", pool.ID)
	return nil
}
