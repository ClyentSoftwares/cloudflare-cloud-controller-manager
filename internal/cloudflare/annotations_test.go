package cloudflare

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostnameFromService(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		hostname, ok := hostnameFromService(managedService(nil))
		assert.True(t, ok)
		assert.Equal(t, testHostname, hostname)
	})

	t.Run("absent", func(t *testing.T) {
		_, ok := hostnameFromService(testService(nil, 80))
		assert.False(t, ok)
	})

	t.Run("empty value treated as absent", func(t *testing.T) {
		_, ok := hostnameFromService(testService(map[string]string{annotationHostname: ""}, 80))
		assert.False(t, ok)
	})
}

func TestDesiredMonitorDefaults(t *testing.T) {
	monitor, err := desiredMonitor(managedService(nil), testHostname)
	require.NoError(t, err)

	assert.Equal(t, "hello-test-com-monitor", monitor.Description)
	assert.Equal(t, "http", monitor.Type)
	assert.Equal(t, "GET", monitor.Method)
	assert.Equal(t, "/", monitor.Path)
	assert.Equal(t, "2xx", monitor.ExpectedCodes)
	assert.Equal(t, int64(60), monitor.Interval)
	assert.Equal(t, int64(5), monitor.Timeout)
	assert.Equal(t, int64(2), monitor.Retries)
	assert.True(t, monitor.FollowRedirects)
	assert.False(t, monitor.AllowInsecure)
	assert.Empty(t, monitor.ProbeZone)
	assert.Nil(t, monitor.Header)
}

func TestBoolAnnotation(t *testing.T) {
	service := managedService(map[string]string{annotationMonitorAllowInsecure: "true"})
	value, err := boolAnnotation(service, annotationMonitorAllowInsecure, false)
	require.NoError(t, err)
	assert.True(t, value)

	_, err = boolAnnotation(managedService(map[string]string{annotationMonitorAllowInsecure: "yep"}), annotationMonitorAllowInsecure, false)
	require.Error(t, err)
}

func TestIntAnnotation(t *testing.T) {
	service := managedService(map[string]string{annotationMonitorInterval: "90"})
	value, err := intAnnotation(service, annotationMonitorInterval, defaultMonitorInterval)
	require.NoError(t, err)
	assert.Equal(t, int64(90), value)

	for name, raw := range map[string]string{
		"not a number": "ninety",
		"zero":         "0",
		"negative":     "-5",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := intAnnotation(managedService(map[string]string{annotationMonitorInterval: raw}), annotationMonitorInterval, defaultMonitorInterval)
			require.Error(t, err)
		})
	}
}
