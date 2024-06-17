package cloudflare

import (
	"context"
	"fmt"
)

func (c CloudflareAPI) validateAccountId() bool {
	if c.AccountId == "" {
		c.Log.Info("Account ID not provided")
		return false
	}

	ctx := context.Background()
	account, _, err := c.CloudflareClient.Account(ctx, c.AccountId)

	if err != nil {
		c.Log.Error(err, "error retrieving account details", "accountId", c.AccountId)
		return false
	}

	return account.ID == c.AccountId
}

// gets AccountId from Account Name
func (c *CloudflareAPI) GetAccountId() (string, error) {
	if !c.validateAccountId() {
		return "", fmt.Errorf("error fetching Account by account ID")
	}
	return c.AccountId, nil
}
