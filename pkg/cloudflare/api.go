package cloudflare

import (
	"regexp"

	"github.com/cloudflare/cloudflare-go"
	"github.com/go-logr/logr"
)

// CloudflareAPI config object holding all relevant fields to use the API
type CloudflareAPI struct {
	Log              logr.Logger
	AccountId        string
	APIToken         string
	ZoneId           string
	CloudflareClient *cloudflare.API
}

func NewCloudflareAPI(token string, zoneId string, accountId string) (*CloudflareAPI, error) {

	client, err := cloudflare.NewWithAPIToken(token)

	if err != nil {
		return nil, err
	}

	return &CloudflareAPI{
		CloudflareClient: client,
		APIToken:         token,
		ZoneId:           zoneId,
		AccountId:        accountId,
	}, err

}

// ValidateAll validates the contents of the CloudflareAPI struct
func (c *CloudflareAPI) ValidateAll() error {
	c.Log.Info("In validation")
	if _, err := c.GetAccountId(); err != nil {
		return err
	}

	//TODO load balancer check

	c.Log.Info("Validation successful")
	return nil
}

func (c *CloudflareAPI) FormatResourceName(name string) string {
	re := regexp.MustCompile(`[^A-Za-z0-9_.-]`)
	return re.ReplaceAllString(name, "_")
}
