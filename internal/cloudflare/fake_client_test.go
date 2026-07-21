package cloudflare

import (
	"context"
	"fmt"

	"github.com/ClyentSoftwares/cloudflare-cloud-controller-manager/internal/cfapi"
)

// fakeLBClient is an in-memory implementation of LBClient with per-method
// error injection and call recording, mirroring the real client's contract
type fakeLBClient struct {
	monitors      map[string]cfapi.Monitor      // keyed by description
	pools         map[string]cfapi.Pool         // keyed by name
	loadBalancers map[string]cfapi.LoadBalancer // keyed by name

	errs   map[string]error // method name -> error returned by that method
	calls  []string
	nextID int
}

func newFakeLBClient() *fakeLBClient {
	return &fakeLBClient{
		monitors:      map[string]cfapi.Monitor{},
		pools:         map[string]cfapi.Pool{},
		loadBalancers: map[string]cfapi.LoadBalancer{},
		errs:          map[string]error{},
	}
}

func (f *fakeLBClient) record(method string) error {
	f.calls = append(f.calls, method)
	return f.errs[method]
}

func (f *fakeLBClient) callCount(method string) int {
	count := 0
	for _, call := range f.calls {
		if call == method {
			count++
		}
	}
	return count
}

func (f *fakeLBClient) newID(kind string) string {
	f.nextID++
	return fmt.Sprintf("%s-id-%d", kind, f.nextID)
}

var _ LBClient = (*fakeLBClient)(nil)

func (f *fakeLBClient) GetMonitor(_ context.Context, description string) (cfapi.Monitor, error) {
	if err := f.record("GetMonitor"); err != nil {
		return cfapi.Monitor{}, err
	}
	monitor, ok := f.monitors[description]
	if !ok {
		return cfapi.Monitor{}, fmt.Errorf("monitor %q: %w", description, cfapi.ErrNotFound)
	}
	return monitor, nil
}

func (f *fakeLBClient) CreateMonitor(_ context.Context, monitor cfapi.Monitor) (cfapi.Monitor, error) {
	if err := f.record("CreateMonitor"); err != nil {
		return cfapi.Monitor{}, err
	}
	monitor.ID = f.newID("monitor")
	f.monitors[monitor.Description] = monitor
	return monitor, nil
}

func (f *fakeLBClient) UpdateMonitor(_ context.Context, monitor cfapi.Monitor) (cfapi.Monitor, error) {
	if err := f.record("UpdateMonitor"); err != nil {
		return cfapi.Monitor{}, err
	}
	f.monitors[monitor.Description] = monitor
	return monitor, nil
}

func (f *fakeLBClient) DeleteMonitor(_ context.Context, description string) error {
	if err := f.record("DeleteMonitor"); err != nil {
		return err
	}
	delete(f.monitors, description)
	return nil
}

func (f *fakeLBClient) GetPool(_ context.Context, name string) (cfapi.Pool, error) {
	if err := f.record("GetPool"); err != nil {
		return cfapi.Pool{}, err
	}
	pool, ok := f.pools[name]
	if !ok {
		return cfapi.Pool{}, fmt.Errorf("pool %q: %w", name, cfapi.ErrNotFound)
	}
	return pool, nil
}

func (f *fakeLBClient) CreatePool(_ context.Context, pool cfapi.Pool) (cfapi.Pool, error) {
	if err := f.record("CreatePool"); err != nil {
		return cfapi.Pool{}, err
	}
	pool.ID = f.newID("pool")
	f.pools[pool.Name] = pool
	return pool, nil
}

func (f *fakeLBClient) UpdatePool(_ context.Context, pool cfapi.Pool) (cfapi.Pool, error) {
	if err := f.record("UpdatePool"); err != nil {
		return cfapi.Pool{}, err
	}
	f.pools[pool.Name] = pool
	return pool, nil
}

func (f *fakeLBClient) DeletePool(_ context.Context, name string) error {
	if err := f.record("DeletePool"); err != nil {
		return err
	}
	delete(f.pools, name)
	return nil
}

func (f *fakeLBClient) GetLoadBalancer(_ context.Context, name string) (cfapi.LoadBalancer, error) {
	if err := f.record("GetLoadBalancer"); err != nil {
		return cfapi.LoadBalancer{}, err
	}
	lb, ok := f.loadBalancers[name]
	if !ok {
		return cfapi.LoadBalancer{}, fmt.Errorf("load balancer %q: %w", name, cfapi.ErrNotFound)
	}
	return lb, nil
}

func (f *fakeLBClient) CreateLoadBalancer(_ context.Context, lb cfapi.LoadBalancer) (cfapi.LoadBalancer, error) {
	if err := f.record("CreateLoadBalancer"); err != nil {
		return cfapi.LoadBalancer{}, err
	}
	lb.ID = f.newID("lb")
	f.loadBalancers[lb.Name] = lb
	return lb, nil
}

func (f *fakeLBClient) UpdateLoadBalancer(_ context.Context, lb cfapi.LoadBalancer) (cfapi.LoadBalancer, error) {
	if err := f.record("UpdateLoadBalancer"); err != nil {
		return cfapi.LoadBalancer{}, err
	}
	f.loadBalancers[lb.Name] = lb
	return lb, nil
}

func (f *fakeLBClient) DeleteLoadBalancer(_ context.Context, name string) error {
	if err := f.record("DeleteLoadBalancer"); err != nil {
		return err
	}
	delete(f.loadBalancers, name)
	return nil
}
