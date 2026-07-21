package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFromEnv(t *testing.T) {
	t.Setenv("CLOUDFLARE_API_TOKEN", "token-123")
	t.Setenv("CLOUDFLARE_ZONE_ID", "zone-123")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "account-123")

	cfg, err := Read()
	require.NoError(t, err)
	assert.Equal(t, "token-123", cfg.CloudflareClient.Token)
	assert.Equal(t, "zone-123", cfg.CloudflareClient.ZoneID)
	assert.Equal(t, "account-123", cfg.CloudflareClient.AccountID)
}

func TestReadFromFile(t *testing.T) {
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "token")
	require.NoError(t, os.WriteFile(tokenFile, []byte("  file-token\n"), 0o600))

	t.Setenv("CLOUDFLARE_API_TOKEN_FILE", tokenFile)

	cfg, err := Read()
	require.NoError(t, err)
	assert.Equal(t, "file-token", cfg.CloudflareClient.Token, "file content must be whitespace-trimmed")
}

func TestReadEnvTakesPrecedenceOverFile(t *testing.T) {
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "token")
	require.NoError(t, os.WriteFile(tokenFile, []byte("file-token"), 0o600))

	t.Setenv("CLOUDFLARE_API_TOKEN", "env-token")
	t.Setenv("CLOUDFLARE_API_TOKEN_FILE", tokenFile)

	cfg, err := Read()
	require.NoError(t, err)
	assert.Equal(t, "env-token", cfg.CloudflareClient.Token)
}

func TestReadMissingFileFails(t *testing.T) {
	t.Setenv("CLOUDFLARE_API_TOKEN_FILE", filepath.Join(t.TempDir(), "does-not-exist"))

	_, err := Read()
	require.Error(t, err)
}

func TestValidateRequiresAllCredentials(t *testing.T) {
	err := CloudflareCCMConfiguration{}.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CLOUDFLARE_API_TOKEN")
	assert.Contains(t, err.Error(), "CLOUDFLARE_ZONE_ID")
	assert.Contains(t, err.Error(), "CLOUDFLARE_ACCOUNT_ID")

	valid := CloudflareCCMConfiguration{
		CloudflareClient: CloudflareClientConfiguration{
			Token:     "t",
			ZoneID:    "z",
			AccountID: "a",
		},
	}
	assert.NoError(t, valid.Validate())
}
