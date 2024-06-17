package cloudflare

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

func GetNodeExternalIP(node *v1.Node) (string, error) {
	for _, address := range node.Status.Addresses {
		if address.Type == v1.NodeExternalIP {
			return address.Address, nil
		}
	}
	return "", fmt.Errorf("no external IP found for node %v. %v", node.Name, node)
}
