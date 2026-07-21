// Package cloudflare implements a kubernetes cloud provider for cloudflare
// load balancers.
package cloudflare

import (
	"context"
	"io"
	"time"

	"github.com/ClyentSoftwares/cloudflare-cloud-controller-manager/internal/cfapi"
	"github.com/ClyentSoftwares/cloudflare-cloud-controller-manager/internal/config"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

const providerName = "cloudflare"

const credentialValidationTimeout = 30 * time.Second

// providerVersion is set by the build process using -ldflags -X.
var providerVersion = "unknown"

type cloud struct {
	loadBalancers *loadBalancers
}

func newCloud(_ io.Reader) (cloudprovider.Interface, error) {
	cfg, err := config.Read()
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	client := cfapi.NewClient(
		cfg.CloudflareClient.Token,
		cfg.CloudflareClient.ZoneID,
		cfg.CloudflareClient.AccountID,
		klog.Background().WithName("cfapi"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), credentialValidationTimeout)
	defer cancel()
	if err := client.ValidateCredentials(ctx); err != nil {
		return nil, err
	}

	klog.InfoS("cloudflare cloud controller manager started", "version", providerVersion)

	return &cloud{loadBalancers: newLoadBalancers(client)}, nil
}

func (c *cloud) Initialize(_ cloudprovider.ControllerClientBuilder, _ <-chan struct{}) {}

func (c *cloud) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return c.loadBalancers, true
}

func (c *cloud) Instances() (cloudprovider.Instances, bool) {
	return nil, false
}

// InstancesV2 is not implemented as cloudflare does not host the cluster nodes
func (c *cloud) InstancesV2() (cloudprovider.InstancesV2, bool) {
	return nil, false
}

func (c *cloud) Zones() (cloudprovider.Zones, bool) {
	return nil, false
}

func (c *cloud) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

func (c *cloud) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}

func (c *cloud) ProviderName() string {
	return providerName
}

func (c *cloud) HasClusterID() bool {
	return false
}

func init() {
	cloudprovider.RegisterCloudProvider(providerName, newCloud)
}
