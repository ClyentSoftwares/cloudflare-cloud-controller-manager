package cloudflare

import (
	"regexp"
	"strings"
)

var invalidNameChars = regexp.MustCompile(`[^a-z0-9_-]+`)

// sanitizeResourceName replaces characters not accepted by the cloudflare API
// for pool and origin names with hyphens.
func sanitizeResourceName(name string) string {
	sanitized := invalidNameChars.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(sanitized, "-")
}

// loadBalancerName stays a hostname as it doubles as the DNS record.
func loadBalancerName(hostname string) string {
	return strings.ToLower(hostname)
}

func poolName(hostname string) string {
	return sanitizeResourceName(hostname) + "-pool"
}

func monitorName(hostname string) string {
	return sanitizeResourceName(hostname) + "-monitor"
}

func originName(ip string) string {
	return sanitizeResourceName(ip)
}
