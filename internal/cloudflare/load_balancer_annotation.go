package cloudflare

import (
	"errors"
	"strconv"

	v1 "k8s.io/api/core/v1"
)

const (
	// serviceAnnotationLoadBalancerID is the name of the loadbalancer
	serviceAnnotationLoadBalancerHostName = "cloudflare-load-balancer.clyent.dev/hostname"

	// serviceAnnotationLoadBalancerMonitorPath is the path the monitor will perform the health check
	serviceAnnotationLoadBalancerMonitorPath = "cloudflare-load-balancer.clyent.dev/monitor-path"

	// serviceAnnotationLoadBalancerMonitorAllowInsecure allows for insecure health checks
	serviceAnnotationLoadBalancerMonitorAllowInsecure = "cloudflare-load-balancer.clyent.dev/monitor-allow-insecure"

	// serviceAnnotationLoadBalancerMonitorType defines the type of monitor e.g tcp, http, https
	serviceAnnotationLoadBalancerMonitorType = "cloudflare-load-balancer.clyent.dev/monitor-type"
)

var (
	errLoadBalancerInvalidAnnotation = errors.New("load balancer invalid loadbalancer annotation")
)

func GetLoadBalancerHostName(service *v1.Service) (string, error) {
	loadBalancerHostName, ok := service.Annotations[serviceAnnotationLoadBalancerHostName]
	if !ok {
		return "", errLoadBalancerInvalidAnnotation
	}

	return loadBalancerHostName, nil
}

func GetLoadBalancerMonitorPath(service *v1.Service) (string, error) {
	loadBalancerMonitorPath, ok := service.Annotations[serviceAnnotationLoadBalancerMonitorPath]
	if !ok {
		return "/", nil
	}

	return loadBalancerMonitorPath, nil
}

func GetLoadBalancerMonitorAllowInsecure(service *v1.Service) (bool, error) {
	loadBalancerMonitorAllowInsecure, ok := service.Annotations[serviceAnnotationLoadBalancerMonitorAllowInsecure]
	if !ok {
		return false, nil
	}

	value, err := strconv.ParseBool(loadBalancerMonitorAllowInsecure)
	if err != nil {
		return false, err
	}

	return value, nil
}

func GetLoadBalancerMonitorType(service *v1.Service) (string, error) {
	loadBalancerMonitorType, ok := service.Annotations[serviceAnnotationLoadBalancerMonitorType]
	if !ok {
		return "http", nil
	}

	return loadBalancerMonitorType, nil
}
