package cloudflare

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const testHostname = "hello.test.com"

func testService(annotations map[string]string, ports ...int32) *v1.Service {
	servicePorts := make([]v1.ServicePort, 0, len(ports))
	for _, port := range ports {
		servicePorts = append(servicePorts, v1.ServicePort{Port: port})
	}
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "hello-world",
			Namespace:   "default",
			Annotations: annotations,
		},
		Spec: v1.ServiceSpec{
			Type:  v1.ServiceTypeLoadBalancer,
			Ports: servicePorts,
		},
	}
}

func managedService(extraAnnotations map[string]string, ports ...int32) *v1.Service {
	annotations := map[string]string{annotationHostname: testHostname}
	for key, value := range extraAnnotations {
		annotations[key] = value
	}
	if len(ports) == 0 {
		ports = []int32{80}
	}
	return testService(annotations, ports...)
}

func testNode(name, externalIP string) *v1.Node {
	node := &v1.Node{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if externalIP != "" {
		node.Status.Addresses = []v1.NodeAddress{
			{Type: v1.NodeInternalIP, Address: "10.0.0.1"},
			{Type: v1.NodeExternalIP, Address: externalIP},
		}
	} else {
		node.Status.Addresses = []v1.NodeAddress{
			{Type: v1.NodeInternalIP, Address: "10.0.0.1"},
		}
	}
	return node
}

func TestEnsureLoadBalancerCreatesResources(t *testing.T) {
	fake := newFakeLBClient()
	lbs := newLoadBalancers(fake)
	nodes := []*v1.Node{testNode("node-1", "203.0.113.1"), testNode("node-2", "203.0.113.2")}

	status, err := lbs.EnsureLoadBalancer(context.Background(), "cluster", managedService(nil), nodes)
	require.NoError(t, err)
	require.NotNil(t, status)
	require.Len(t, status.Ingress, 1)
	assert.Equal(t, testHostname, status.Ingress[0].Hostname)

	monitor, ok := fake.monitors["hello-test-com-monitor"]
	require.True(t, ok, "monitor should be created under the sanitized description")
	assert.Equal(t, "http", monitor.Type)
	assert.Equal(t, "GET", monitor.Method)
	assert.Equal(t, "/", monitor.Path)
	assert.Equal(t, int64(80), monitor.Port)
	assert.Equal(t, "2xx", monitor.ExpectedCodes)
	assert.Equal(t, int64(60), monitor.Interval)

	pool, ok := fake.pools["hello-test-com-pool"]
	require.True(t, ok, "pool should be created under the sanitized name")
	assert.Equal(t, monitor.ID, pool.MonitorID)
	require.Len(t, pool.Origins, 2)
	assert.Equal(t, "203-0-113-1", pool.Origins[0].Name)
	assert.Equal(t, "203.0.113.1", pool.Origins[0].Address)

	lb, ok := fake.loadBalancers[testHostname]
	require.True(t, ok, "load balancer should be created under the hostname")
	assert.Equal(t, pool.ID, lb.FallbackPoolID)
	assert.Equal(t, []string{pool.ID}, lb.DefaultPoolIDs)
	assert.True(t, lb.Proxied)
}

func TestEnsureLoadBalancerIsIdempotent(t *testing.T) {
	fake := newFakeLBClient()
	lbs := newLoadBalancers(fake)
	nodes := []*v1.Node{testNode("node-1", "203.0.113.1")}

	_, err := lbs.EnsureLoadBalancer(context.Background(), "cluster", managedService(nil), nodes)
	require.NoError(t, err)
	fake.calls = nil

	_, err = lbs.EnsureLoadBalancer(context.Background(), "cluster", managedService(nil), nodes)
	require.NoError(t, err)

	for _, method := range []string{
		"CreateMonitor", "UpdateMonitor",
		"CreatePool", "UpdatePool",
		"CreateLoadBalancer", "UpdateLoadBalancer",
	} {
		assert.Zerof(t, fake.callCount(method), "second ensure must not call %s", method)
	}
}

func TestEnsureLoadBalancerTransientErrorDoesNotCreate(t *testing.T) {
	fake := newFakeLBClient()
	fake.errs["GetMonitor"] = errors.New("cloudflare: 500 internal server error")
	lbs := newLoadBalancers(fake)
	nodes := []*v1.Node{testNode("node-1", "203.0.113.1")}

	_, err := lbs.EnsureLoadBalancer(context.Background(), "cluster", managedService(nil), nodes)
	require.Error(t, err)
	assert.Zero(t, fake.callCount("CreateMonitor"), "a transient error must not trigger creation")
	assert.Zero(t, fake.callCount("CreatePool"))
	assert.Zero(t, fake.callCount("CreateLoadBalancer"))
}

func TestEnsureLoadBalancerUpdatesDriftedMonitor(t *testing.T) {
	fake := newFakeLBClient()
	lbs := newLoadBalancers(fake)
	nodes := []*v1.Node{testNode("node-1", "203.0.113.1")}

	_, err := lbs.EnsureLoadBalancer(context.Background(), "cluster", managedService(nil), nodes)
	require.NoError(t, err)
	originalID := fake.monitors["hello-test-com-monitor"].ID
	fake.calls = nil

	changed := managedService(map[string]string{annotationMonitorPath: "/health"})
	_, err = lbs.EnsureLoadBalancer(context.Background(), "cluster", changed, nodes)
	require.NoError(t, err)

	assert.Equal(t, 1, fake.callCount("UpdateMonitor"))
	assert.Zero(t, fake.callCount("CreateMonitor"))
	monitor := fake.monitors["hello-test-com-monitor"]
	assert.Equal(t, "/health", monitor.Path)
	assert.Equal(t, originalID, monitor.ID, "update must reuse the existing monitor ID")
}

func TestEnsureLoadBalancerUpdatesPoolOnNodeChange(t *testing.T) {
	fake := newFakeLBClient()
	lbs := newLoadBalancers(fake)

	_, err := lbs.EnsureLoadBalancer(context.Background(), "cluster", managedService(nil),
		[]*v1.Node{testNode("node-1", "203.0.113.1")})
	require.NoError(t, err)
	fake.calls = nil

	_, err = lbs.EnsureLoadBalancer(context.Background(), "cluster", managedService(nil),
		[]*v1.Node{testNode("node-1", "203.0.113.1"), testNode("node-2", "203.0.113.2")})
	require.NoError(t, err)

	assert.Equal(t, 1, fake.callCount("UpdatePool"))
	assert.Len(t, fake.pools["hello-test-com-pool"].Origins, 2)
}

func TestEnsureLoadBalancerSkipsNodesWithoutExternalIP(t *testing.T) {
	fake := newFakeLBClient()
	lbs := newLoadBalancers(fake)

	_, err := lbs.EnsureLoadBalancer(context.Background(), "cluster", managedService(nil),
		[]*v1.Node{testNode("node-1", "203.0.113.1"), testNode("node-2", "")})
	require.NoError(t, err)
	require.Len(t, fake.pools["hello-test-com-pool"].Origins, 1)
	assert.Equal(t, "203.0.113.1", fake.pools["hello-test-com-pool"].Origins[0].Address)
}

func TestEnsureLoadBalancerFailsWithoutAnyExternalIP(t *testing.T) {
	fake := newFakeLBClient()
	lbs := newLoadBalancers(fake)

	_, err := lbs.EnsureLoadBalancer(context.Background(), "cluster", managedService(nil),
		[]*v1.Node{testNode("node-1", ""), testNode("node-2", "")})
	require.Error(t, err)
	assert.Zero(t, fake.callCount("CreatePool"))
}

func TestEnsureLoadBalancerFailsWithoutPorts(t *testing.T) {
	fake := newFakeLBClient()
	lbs := newLoadBalancers(fake)
	service := testService(map[string]string{annotationHostname: testHostname}) // no ports

	_, err := lbs.EnsureLoadBalancer(context.Background(), "cluster", service,
		[]*v1.Node{testNode("node-1", "203.0.113.1")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no ports")
}

func TestUnmanagedServiceIsIgnoredEverywhere(t *testing.T) {
	fake := newFakeLBClient()
	lbs := newLoadBalancers(fake)
	service := testService(nil, 80)
	nodes := []*v1.Node{testNode("node-1", "203.0.113.1")}
	ctx := context.Background()

	status, exists, err := lbs.GetLoadBalancer(ctx, "cluster", service)
	require.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, status)

	assert.Empty(t, lbs.GetLoadBalancerName(ctx, "cluster", service))

	ensured, err := lbs.EnsureLoadBalancer(ctx, "cluster", service, nodes)
	require.NoError(t, err)
	assert.Empty(t, ensured.Ingress)

	require.NoError(t, lbs.UpdateLoadBalancer(ctx, "cluster", service, nodes))
	require.NoError(t, lbs.EnsureLoadBalancerDeleted(ctx, "cluster", service))
	assert.Empty(t, fake.calls, "unmanaged services must not reach the cloudflare API")
}

func TestEnsureLoadBalancerDeletedRemovesEverything(t *testing.T) {
	fake := newFakeLBClient()
	lbs := newLoadBalancers(fake)
	ctx := context.Background()

	_, err := lbs.EnsureLoadBalancer(ctx, "cluster", managedService(nil),
		[]*v1.Node{testNode("node-1", "203.0.113.1")})
	require.NoError(t, err)

	require.NoError(t, lbs.EnsureLoadBalancerDeleted(ctx, "cluster", managedService(nil)))
	assert.Empty(t, fake.loadBalancers)
	assert.Empty(t, fake.pools)
	assert.Empty(t, fake.monitors)
}

func TestEnsureLoadBalancerDeletedIsIdempotent(t *testing.T) {
	fake := newFakeLBClient()
	lbs := newLoadBalancers(fake)
	ctx := context.Background()

	_, err := lbs.EnsureLoadBalancer(ctx, "cluster", managedService(nil),
		[]*v1.Node{testNode("node-1", "203.0.113.1")})
	require.NoError(t, err)

	// simulate the load balancer having been deleted out of band
	delete(fake.loadBalancers, testHostname)

	require.NoError(t, lbs.EnsureLoadBalancerDeleted(ctx, "cluster", managedService(nil)))
	assert.Empty(t, fake.pools, "pool must still be deleted when the load balancer is already gone")
	assert.Empty(t, fake.monitors, "monitor must still be deleted when the load balancer is already gone")

	require.NoError(t, lbs.EnsureLoadBalancerDeleted(ctx, "cluster", managedService(nil)))
}

func TestEnsureLoadBalancerDeletedPropagatesRealErrors(t *testing.T) {
	fake := newFakeLBClient()
	fake.errs["DeletePool"] = errors.New("cloudflare: 429 rate limited")
	lbs := newLoadBalancers(fake)
	ctx := context.Background()

	_, err := lbs.EnsureLoadBalancer(ctx, "cluster", managedService(nil),
		[]*v1.Node{testNode("node-1", "203.0.113.1")})
	require.NoError(t, err)

	require.Error(t, lbs.EnsureLoadBalancerDeleted(ctx, "cluster", managedService(nil)))
}

func TestGetLoadBalancer(t *testing.T) {
	fake := newFakeLBClient()
	lbs := newLoadBalancers(fake)
	ctx := context.Background()

	status, exists, err := lbs.GetLoadBalancer(ctx, "cluster", managedService(nil))
	require.NoError(t, err)
	assert.False(t, exists)
	assert.Nil(t, status)

	_, err = lbs.EnsureLoadBalancer(ctx, "cluster", managedService(nil),
		[]*v1.Node{testNode("node-1", "203.0.113.1")})
	require.NoError(t, err)

	status, exists, err = lbs.GetLoadBalancer(ctx, "cluster", managedService(nil))
	require.NoError(t, err)
	assert.True(t, exists)
	require.Len(t, status.Ingress, 1)
	assert.Equal(t, testHostname, status.Ingress[0].Hostname)

	fake.errs["GetLoadBalancer"] = errors.New("cloudflare: 503 unavailable")
	_, _, err = lbs.GetLoadBalancer(ctx, "cluster", managedService(nil))
	require.Error(t, err)
}

func TestEnsureLoadBalancerAppliesMonitorAnnotations(t *testing.T) {
	fake := newFakeLBClient()
	lbs := newLoadBalancers(fake)

	service := managedService(map[string]string{
		annotationMonitorType:          "https",
		annotationMonitorPath:          "/healthz",
		annotationMonitorMethod:        "HEAD",
		annotationMonitorExpectedCodes: "200",
		annotationMonitorInterval:      "30",
		annotationMonitorTimeout:       "10",
		annotationMonitorRetries:       "1",
		annotationMonitorAllowInsecure: "true",
		annotationMonitorProbeZone:     "test.com",
		annotationMonitorHeader:        "ingress.test.com",
	}, 443)

	_, err := lbs.EnsureLoadBalancer(context.Background(), "cluster", service,
		[]*v1.Node{testNode("node-1", "203.0.113.1")})
	require.NoError(t, err)

	monitor := fake.monitors["hello-test-com-monitor"]
	assert.Equal(t, "https", monitor.Type)
	assert.Equal(t, "/healthz", monitor.Path)
	assert.Equal(t, "HEAD", monitor.Method)
	assert.Equal(t, "200", monitor.ExpectedCodes)
	assert.Equal(t, int64(30), monitor.Interval)
	assert.Equal(t, int64(10), monitor.Timeout)
	assert.Equal(t, int64(1), monitor.Retries)
	assert.Equal(t, int64(443), monitor.Port)
	assert.True(t, monitor.AllowInsecure)
	assert.Equal(t, "test.com", monitor.ProbeZone)
	assert.Equal(t, map[string][]string{"Host": {"ingress.test.com"}}, monitor.Header)
}

func TestEnsureLoadBalancerRejectsInvalidAnnotations(t *testing.T) {
	fake := newFakeLBClient()
	lbs := newLoadBalancers(fake)

	service := managedService(map[string]string{
		annotationMonitorAllowInsecure: "not-a-bool",
		annotationMonitorInterval:      "soon",
	})

	_, err := lbs.EnsureLoadBalancer(context.Background(), "cluster", service,
		[]*v1.Node{testNode("node-1", "203.0.113.1")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), annotationMonitorAllowInsecure)
	assert.Contains(t, err.Error(), annotationMonitorInterval)
	assert.Zero(t, fake.callCount("CreateMonitor"))
}
