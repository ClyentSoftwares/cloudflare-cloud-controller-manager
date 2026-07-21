package cloudflare

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeResourceName(t *testing.T) {
	cases := map[string]string{
		"hello.test.com":  "hello-test-com",
		"Hello.Test.COM":  "hello-test-com",
		"203.0.113.1":     "203-0-113-1",
		"2001:db8::1":     "2001-db8-1",
		"with_underscore": "with_underscore",
		"..leading.dots.": "leading-dots",
	}
	for input, want := range cases {
		assert.Equal(t, want, sanitizeResourceName(input), "input %q", input)
	}
}

func TestNameDerivation(t *testing.T) {
	assert.Equal(t, "hello.test.com", loadBalancerName("Hello.Test.Com"))
	assert.Equal(t, "hello-test-com-pool", poolName("hello.test.com"))
	assert.Equal(t, "hello-test-com-monitor", monitorName("hello.test.com"))
	assert.Equal(t, "203-0-113-1", originName("203.0.113.1"))
}
