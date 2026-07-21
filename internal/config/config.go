// Package config reads the controller configuration from the environment.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	cloudflareAPIToken  = "CLOUDFLARE_API_TOKEN"
	cloudflareZoneID    = "CLOUDFLARE_ZONE_ID"
	cloudflareAccountID = "CLOUDFLARE_ACCOUNT_ID"
)

// CloudflareClientConfiguration holds the credentials for the Cloudflare API.
type CloudflareClientConfiguration struct {
	Token     string
	ZoneID    string
	AccountID string
}

// CloudflareCCMConfiguration is the controller's full configuration.
type CloudflareCCMConfiguration struct {
	CloudflareClient CloudflareClientConfiguration
}

// read values from environment variables or from file set via _FILE env var
// values set directly via env var take precedence over values set via file.
func readFromEnvOrFile(envVar string) (string, error) {
	// check if the value is set directly via env
	value, ok := os.LookupEnv(envVar)
	if ok {
		return value, nil
	}

	// check if the value is set via a file
	value, ok = os.LookupEnv(envVar + "_FILE")
	if !ok {
		// return no error here, the values could be optional
		// and the function "Validate()" below checks that all required variables are set
		return "", nil
	}

	// read file content
	valueBytes, err := os.ReadFile(value)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", envVar+"_FILE", err)
	}

	return strings.TrimSpace(string(valueBytes)), nil
}

// Read evaluates all environment variables and returns a [CloudflareCCMConfiguration]. It only validates as far as
// it needs to parse the values. For business logic validation, check out [CloudflareCCMConfiguration.Validate].
func Read() (CloudflareCCMConfiguration, error) {
	var err error
	// Collect all errors and return them as one.
	// This helps users because they will see all the errors at once
	// instead of having to fix them one by one.
	var errs []error
	var cfg CloudflareCCMConfiguration

	cfg.CloudflareClient.Token, err = readFromEnvOrFile(cloudflareAPIToken)
	if err != nil {
		errs = append(errs, err)
	}

	cfg.CloudflareClient.ZoneID, err = readFromEnvOrFile(cloudflareZoneID)
	if err != nil {
		errs = append(errs, err)
	}

	cfg.CloudflareClient.AccountID, err = readFromEnvOrFile(cloudflareAccountID)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return CloudflareCCMConfiguration{}, errors.Join(errs...)
	}

	return cfg, nil
}

// Validate checks that all required configuration values are set.
func (c CloudflareCCMConfiguration) Validate() error {
	// Collect all errors and return them as one.
	// This helps users because they will see all the errors at once
	// instead of having to fix them one by one.
	var errs []error

	if c.CloudflareClient.Token == "" {
		errs = append(errs, fmt.Errorf("environment variable %q is required", cloudflareAPIToken))
	}

	if c.CloudflareClient.ZoneID == "" {
		errs = append(errs, fmt.Errorf("environment variable %q is required", cloudflareZoneID))
	}

	if c.CloudflareClient.AccountID == "" {
		errs = append(errs, fmt.Errorf("environment variable %q is required", cloudflareAccountID))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
