package cfapi

import (
	"errors"
	"fmt"
	"net/http"

	cloudflare "github.com/cloudflare/cloudflare-go/v7"
)

// ErrNotFound is returned when a resource does not exist on cloudflare.
// Callers must check for it with errors.Is before creating a resource, any
// other error may be transient and must not be treated as "not found".
var ErrNotFound = errors.New("cloudflare resource not found")

// checks whether the error is a cloudflare API 404 response.
func isNotFoundAPIError(err error) bool {
	var apiErr *cloudflare.Error
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// wraps cloudflare API errors and maps 404 responses onto ErrNotFound.
func wrapAPIError(err error, action string) error {
	if isNotFoundAPIError(err) {
		return fmt.Errorf("%s: %w", action, ErrNotFound)
	}
	return fmt.Errorf("%s: %w", action, err)
}
