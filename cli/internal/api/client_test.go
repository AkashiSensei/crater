package api

import (
	"strings"
	"testing"
)

func TestSandboxPassthroughRejectsNonLoopbackHost(t *testing.T) {
	t.Setenv("CRATER_TEST_SANDBOX", "1")
	t.Setenv("CRATER_TEST_SANDBOX_HTTP", "passthrough")

	client := NewClient("https://example.com")
	_, err := client.httpClient.R().Get("/api/v1/vcjobs")
	if err == nil || !strings.Contains(err.Error(), "rejected non-loopback host") {
		t.Fatalf("unexpected passthrough result: %v", err)
	}
}
