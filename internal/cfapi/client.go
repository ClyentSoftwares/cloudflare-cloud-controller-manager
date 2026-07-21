// Package cfapi wraps the cloudflare-go SDK behind small domain types.
package cfapi

import (
	"context"
	"fmt"

	cloudflare "github.com/cloudflare/cloudflare-go/v7"
	"github.com/cloudflare/cloudflare-go/v7/accounts"
	"github.com/cloudflare/cloudflare-go/v7/option"
	"github.com/go-logr/logr"
)

// Client config object holding all relevant fields to use the cloudflare API
type Client struct {
	log       logr.Logger
	api       *cloudflare.Client
	zoneID    string
	accountID string
}

// NewClient creates a new cloudflare API client for the given zone and account.
func NewClient(token, zoneID, accountID string, log logr.Logger) *Client {
	return &Client{
		log:       log,
		api:       cloudflare.NewClient(option.WithAPIToken(token)),
		zoneID:    zoneID,
		accountID: accountID,
	}
}

// ValidateCredentials validates the API token against the configured account.
func (c *Client) ValidateCredentials(ctx context.Context) error {
	account, err := c.api.Accounts.Get(ctx, accounts.AccountGetParams{
		AccountID: cloudflare.F(c.accountID),
	})
	if err != nil {
		return wrapAPIError(err, fmt.Sprintf("validating cloudflare credentials for account %q", c.accountID))
	}
	c.log.Info("validated cloudflare credentials", "accountID", c.accountID, "accountName", account.Name)
	return nil
}
