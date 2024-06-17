package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	cloudflareAPIToken  = "CLOUDFLARE_API_TOKEN"
	cloudflareZoneId    = "CLOUDFLARE_ZONE_ID"
	cloudflareAccountId = "CLOUDFLARE_ACCOUNT_ID"

	debug = "DEBUG"
)

type CloudflareClientConfiguration struct {
	Token     string
	ZoneId    string
	AccountId string
	Debug     bool
}

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

	cfg.CloudflareClient.ZoneId, err = readFromEnvOrFile(cloudflareZoneId)
	if err != nil {
		errs = append(errs, err)
	}

	cfg.CloudflareClient.AccountId, err = readFromEnvOrFile(cloudflareAccountId)
	if err != nil {
		errs = append(errs, err)
	}

	cfg.CloudflareClient.Debug, err = getEnvBool(debug, false)
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return CloudflareCCMConfiguration{}, errors.Join(errs...)
	}

	return cfg, nil
}

func (c CloudflareCCMConfiguration) Validate() (err error) {
	// Collect all errors and return them as one.
	// This helps users because they will see all the errors at once
	// instead of having to fix them one by one.
	var errs []error

	if c.CloudflareClient.Token == "" {
		errs = append(errs, fmt.Errorf("environment variable %q is required", cloudflareAPIToken))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// getEnvBool returns the boolean parsed from the environment variable with the given key and a potential error
// parsing the var. Returns the default value if the env var is unset.
func getEnvBool(key string, defaultValue bool) (bool, error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue, nil
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("failed to parse %s: %v", key, err)
	}

	return b, nil
}
