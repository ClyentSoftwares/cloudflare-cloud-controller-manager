package cfapi

import (
	"context"
	"errors"
	"fmt"

	cloudflare "github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/load_balancers"
)

func monitorFromSDK(m load_balancers.Monitor) Monitor {
	return Monitor{
		ID:              m.ID,
		Description:     m.Description,
		Type:            string(m.Type),
		Method:          m.Method,
		Path:            m.Path,
		Port:            m.Port,
		ExpectedCodes:   m.ExpectedCodes,
		Interval:        m.Interval,
		Timeout:         m.Timeout,
		Retries:         m.Retries,
		FollowRedirects: m.FollowRedirects,
		AllowInsecure:   m.AllowInsecure,
		ProbeZone:       m.ProbeZone,
		Header:          m.Header,
	}
}

// GetMonitor retrieves a health monitor by its description.
func (c *Client) GetMonitor(ctx context.Context, description string) (Monitor, error) {
	iter := c.api.LoadBalancers.Monitors.ListAutoPaging(ctx, load_balancers.MonitorListParams{
		AccountID: cloudflare.F(c.accountID),
	})
	for iter.Next() {
		if m := iter.Current(); m.Description == description {
			return monitorFromSDK(m), nil
		}
	}
	if err := iter.Err(); err != nil {
		return Monitor{}, wrapAPIError(err, "listing load balancer monitors")
	}
	return Monitor{}, fmt.Errorf("monitor %q: %w", description, ErrNotFound)
}

// CreateMonitor creates a new health monitor.
func (c *Client) CreateMonitor(ctx context.Context, monitor Monitor) (Monitor, error) {
	params := load_balancers.MonitorNewParams{
		AccountID:       cloudflare.F(c.accountID),
		Description:     cloudflare.F(monitor.Description),
		Type:            cloudflare.F(load_balancers.MonitorNewParamsType(monitor.Type)),
		Method:          cloudflare.F(monitor.Method),
		Path:            cloudflare.F(monitor.Path),
		ExpectedCodes:   cloudflare.F(monitor.ExpectedCodes),
		Interval:        cloudflare.F(monitor.Interval),
		Timeout:         cloudflare.F(monitor.Timeout),
		Retries:         cloudflare.F(monitor.Retries),
		FollowRedirects: cloudflare.F(monitor.FollowRedirects),
		AllowInsecure:   cloudflare.F(monitor.AllowInsecure),
	}
	// leave optional fields unset so the API applies its defaults
	if monitor.Port != 0 {
		params.Port = cloudflare.F(monitor.Port)
	}
	if monitor.ProbeZone != "" {
		params.ProbeZone = cloudflare.F(monitor.ProbeZone)
	}
	if len(monitor.Header) > 0 {
		params.Header = cloudflare.F(monitor.Header)
	}

	created, err := c.api.LoadBalancers.Monitors.New(ctx, params)
	if err != nil {
		return Monitor{}, wrapAPIError(err, fmt.Sprintf("creating monitor %q", monitor.Description))
	}
	c.log.Info("created load balancer monitor", "description", monitor.Description, "monitorID", created.ID)
	return monitorFromSDK(*created), nil
}

// UpdateMonitor updates an existing health monitor.
func (c *Client) UpdateMonitor(ctx context.Context, monitor Monitor) (Monitor, error) {
	params := load_balancers.MonitorUpdateParams{
		AccountID:       cloudflare.F(c.accountID),
		Description:     cloudflare.F(monitor.Description),
		Type:            cloudflare.F(load_balancers.MonitorUpdateParamsType(monitor.Type)),
		Method:          cloudflare.F(monitor.Method),
		Path:            cloudflare.F(monitor.Path),
		ExpectedCodes:   cloudflare.F(monitor.ExpectedCodes),
		Interval:        cloudflare.F(monitor.Interval),
		Timeout:         cloudflare.F(monitor.Timeout),
		Retries:         cloudflare.F(monitor.Retries),
		FollowRedirects: cloudflare.F(monitor.FollowRedirects),
		AllowInsecure:   cloudflare.F(monitor.AllowInsecure),
	}
	// leave optional fields unset so the API applies its defaults
	if monitor.Port != 0 {
		params.Port = cloudflare.F(monitor.Port)
	}
	if monitor.ProbeZone != "" {
		params.ProbeZone = cloudflare.F(monitor.ProbeZone)
	}
	if len(monitor.Header) > 0 {
		params.Header = cloudflare.F(monitor.Header)
	}

	updated, err := c.api.LoadBalancers.Monitors.Update(ctx, monitor.ID, params)
	if err != nil {
		return Monitor{}, wrapAPIError(err, fmt.Sprintf("updating monitor %q", monitor.Description))
	}
	c.log.Info("updated load balancer monitor", "description", monitor.Description, "monitorID", monitor.ID)
	return monitorFromSDK(*updated), nil
}

// DeleteMonitor deletes a health monitor by description, ignoring monitors
// that are already gone.
func (c *Client) DeleteMonitor(ctx context.Context, description string) error {
	monitor, err := c.GetMonitor(ctx, description)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	_, err = c.api.LoadBalancers.Monitors.Delete(ctx, monitor.ID, load_balancers.MonitorDeleteParams{
		AccountID: cloudflare.F(c.accountID),
	})
	if err != nil {
		if isNotFoundAPIError(err) {
			return nil
		}
		return wrapAPIError(err, fmt.Sprintf("deleting monitor %q", description))
	}
	c.log.Info("deleted load balancer monitor", "description", description, "monitorID", monitor.ID)
	return nil
}
