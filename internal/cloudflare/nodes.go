package cloudflare

import (
	"fmt"

	"github.com/ClyentSoftwares/cloudflare-cloud-controller-manager/internal/cfapi"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

func nodeExternalIP(node *v1.Node) (string, bool) {
	for _, address := range node.Status.Addresses {
		if address.Type == v1.NodeExternalIP {
			return address.Address, true
		}
	}
	return "", false
}

// desiredOrigins builds the pool origins from the nodes' external IPs,
// skipping nodes without one.
func desiredOrigins(nodes []*v1.Node) ([]cfapi.Origin, error) {
	origins := make([]cfapi.Origin, 0, len(nodes))
	for _, node := range nodes {
		ip, ok := nodeExternalIP(node)
		if !ok {
			klog.InfoS("skipping node without external IP", "node", node.Name)
			continue
		}
		origins = append(origins, cfapi.Origin{
			Name:    originName(ip),
			Address: ip,
			Enabled: true,
			Weight:  1,
		})
	}
	if len(origins) == 0 {
		return nil, fmt.Errorf("no nodes with an external IP address available as load balancer origins")
	}
	return origins, nil
}
