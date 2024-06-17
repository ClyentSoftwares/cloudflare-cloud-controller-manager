package cloudflare

import (
	"encoding/base64"
	"io"

	"github.com/ClyentSoftwares/cloudflare-cloud-controller-manager/internal/config"
	"github.com/ClyentSoftwares/cloudflare-cloud-controller-manager/pkg/cloudflare"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

const (
	providerName = "cloudflare"
)

// providerVersion is set by the build process using -ldflags -X.
var providerVersion = "vUnknown"

type cloud struct {
	cfg    config.CloudflareCCMConfiguration
	Client *cloudflare.CloudflareAPI
}

func newCloud(_ io.Reader) (cloudprovider.Interface, error) {
	cfg, err := config.Read()
	if err != nil {
		return nil, err
	}
	err = cfg.Validate()
	if err != nil {
		return nil, err
	}

	apiToken, err := base64.StdEncoding.DecodeString(cfg.CloudflareClient.Token)
	if err != nil {
		return nil, err
	}

	zoneId, err := base64.StdEncoding.DecodeString(cfg.CloudflareClient.ZoneId)
	if err != nil {
		return nil, err
	}

	accountId, err := base64.StdEncoding.DecodeString(cfg.CloudflareClient.AccountId)
	if err != nil {
		return nil, err
	}

	cloudflareClient, err := cloudflare.NewCloudflareAPI(string(apiToken), string(zoneId), string(accountId))

	if err != nil {
		return nil, err
	}

	klog.Info("Cloudflare Client init")

	err = cloudflareClient.ValidateAll()

	if err != nil {
		klog.Info("Error Validating account ", err)
		return nil, err
	}

	klog.Info("Validated account sucessfully")

	klog.Infof("Cloudflare k8s cloud controller %s started\n", providerVersion)

	return &cloud{
		cfg:    cfg,
		Client: cloudflareClient,
	}, nil
}

func (c *cloud) Initialize(_ cloudprovider.ControllerClientBuilder, _ <-chan struct{}) {
}

func (c *cloud) Instances() (cloudprovider.Instances, bool) {
	// Replaced by InstancesV2
	return nil, false
}

func (c *cloud) InstancesV2() (cloudprovider.InstancesV2, bool) {
	return nil, false
}

func (c *cloud) Zones() (cloudprovider.Zones, bool) {
	// Replaced by InstancesV2
	return nil, false
}

func (c *cloud) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	lbOps := &LoadBalancerOps{}

	return newLoadbalancers(c.Client, lbOps), true
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
