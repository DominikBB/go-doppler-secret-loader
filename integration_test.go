package doppler

import (
	"context"
	"testing"
)

func TestIntegrationListSecrets(t *testing.T) {
	t.Skip("manual integration test: fill in the constants and remove this skip")

	const (
		apiKey  = "paste-your-service-token-here"
		project = "your-project"
		config  = "dev"
	)
	defaults := map[string]string{
		"DB_PASS": "",
		"DB_USER": "postgres",
	}

	client, err := New(project, WithAPIKey(apiKey))
	if err != nil {
		t.Fatal(err)
	}
	secrets, err := client.ListSecrets(context.Background(), config, defaults)
	if err != nil {
		t.Fatal(err)
	}
	for key := range defaults {
		if secrets[key] == "" {
			t.Errorf("secret %q was not resolved", key)
		}
	}
	// Avoid logging secrets. Inspect them locally with a debugger if needed.
}
