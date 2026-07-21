package cloudflare

import (
	"fmt"
	"strconv"

	v1 "k8s.io/api/core/v1"
)

const (
	annotationPrefix = "cloudflare-load-balancer.clyent.dev/"

	// annotationHostname is the hostname of the loadbalancer
	annotationHostname = annotationPrefix + "hostname"

	// annotationMonitorPath is the path the monitor will perform the health check
	annotationMonitorPath = annotationPrefix + "monitor-path"

	// annotationMonitorAllowInsecure allows for insecure health checks
	annotationMonitorAllowInsecure = annotationPrefix + "monitor-allow-insecure"

	// annotationMonitorType defines the type of monitor e.g tcp, http, https
	annotationMonitorType = annotationPrefix + "monitor-type"

	// annotationMonitorProbeZone assign this monitor to emulate the specified zone while probing. This parameter is only valid for HTTP and HTTPS monitors.
	annotationMonitorProbeZone = annotationPrefix + "monitor-probe-zone"

	// annotationMonitorHeader defines the request header used to pass additional information within HTTP request. Currently supported header is 'Host'.
	annotationMonitorHeader = annotationPrefix + "monitor-header"

	// annotationMonitorInterval is the interval in seconds between health checks
	annotationMonitorInterval = annotationPrefix + "monitor-interval"

	// annotationMonitorTimeout is the timeout in seconds of the health check
	annotationMonitorTimeout = annotationPrefix + "monitor-timeout"

	// annotationMonitorRetries is the number of retries before an origin is marked as unhealthy
	annotationMonitorRetries = annotationPrefix + "monitor-retries"

	// annotationMonitorExpectedCodes is the expected HTTP response code or code range of the health check e.g 2xx
	annotationMonitorExpectedCodes = annotationPrefix + "monitor-expected-codes"

	// annotationMonitorMethod is the HTTP method the monitor uses for the health check
	annotationMonitorMethod = annotationPrefix + "monitor-method"
)

const (
	defaultMonitorType          = "http"
	defaultMonitorPath          = "/"
	defaultMonitorMethod        = "GET"
	defaultMonitorExpectedCodes = "2xx"
	defaultMonitorInterval      = 60
	defaultMonitorTimeout       = 5
	defaultMonitorRetries       = 2
)

// hostnameFromService returns the hostname annotation of the service if it is set.
// Services without it are not managed by this controller.
func hostnameFromService(service *v1.Service) (string, bool) {
	hostname, ok := service.Annotations[annotationHostname]
	if !ok || hostname == "" {
		return "", false
	}
	return hostname, true
}

func stringAnnotation(service *v1.Service, key, defaultValue string) string {
	if value, ok := service.Annotations[key]; ok {
		return value
	}
	return defaultValue
}

func boolAnnotation(service *v1.Service, key string, defaultValue bool) (bool, error) {
	value, ok := service.Annotations[key]
	if !ok {
		return defaultValue, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("annotation %s: invalid boolean %q: %w", key, value, err)
	}
	return parsed, nil
}

func intAnnotation(service *v1.Service, key string, defaultValue int64) (int64, error) {
	value, ok := service.Annotations[key]
	if !ok {
		return defaultValue, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("annotation %s: invalid integer %q: %w", key, value, err)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("annotation %s: must be a positive integer, got %d", key, parsed)
	}
	return parsed, nil
}
