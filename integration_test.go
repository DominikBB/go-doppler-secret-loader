package doppler

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"testing"
	"time"
)

// TestIntegrationListSecrets calls the real Doppler API only when explicitly enabled.
// Supply credentials and inputs through environment variables, not source code.
func TestIntegrationListSecrets(t *testing.T) {
	if os.Getenv("DOPPLER_INTEGRATION") != "1" {
		t.Skip("set DOPPLER_INTEGRATION=1 to call the live Doppler API")
	}
	if testing.Short() {
		t.Skip("live integration test disabled in short mode")
	}
	required := func(name string) string {
		t.Helper()
		value := os.Getenv(name)
		if value == "" {
			t.Fatalf("%s is required", name)
		}
		return value
	}
	project := required("DOPPLER_TEST_PROJECT")
	config := required("DOPPLER_TEST_CONFIG")
	token := required("DOPPLER_SERVICE_TOKEN")
	var defaults map[string]string
	if err := json.Unmarshal([]byte(required("DOPPLER_TEST_DEFAULTS")), &defaults); err != nil {
		t.Fatal("DOPPLER_TEST_DEFAULTS must be a JSON object of string values")
	}
	if len(defaults) == 0 {
		t.Fatal("DOPPLER_TEST_DEFAULTS must contain at least one secret key")
	}
	original := maps.Clone(defaults)
	client, err := New(project, WithAPIKey(token))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	secrets, err := client.ListSecrets(ctx, config, defaults)
	if err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(defaults, original) {
		t.Fatal("input defaults were modified")
	}
	if len(secrets) != len(defaults) {
		t.Fatal("result keys do not match input keys")
	}
	for key := range defaults {
		if secrets[key] == "" {
			t.Errorf("secret %q was not resolved", key)
		}
	}
	// Optionally check exact values without ever printing them on failure.
	if raw := os.Getenv("DOPPLER_TEST_EXPECTED"); raw != "" {
		var expected map[string]string
		if err := json.Unmarshal([]byte(raw), &expected); err != nil {
			t.Fatal("DOPPLER_TEST_EXPECTED must be a JSON object of string values")
		}
		for key, want := range expected {
			got, exists := secrets[key]
			if !exists || got != want {
				t.Errorf("secret %q does not match the expected value", key)
			}
		}
	}
}
