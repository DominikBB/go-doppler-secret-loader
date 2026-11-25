// Package doppler loads computed Doppler secrets over caller-provided defaults.
package doppler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

// Client accesses secrets for a single Doppler project. Create one with New.
type Client struct {
	project    string
	apiKey     string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithAPIKey overrides DOPPLER_SERVICE_TOKEN.
func WithAPIKey(key string) Option { return func(c *Client) { c.apiKey = key } }

// WithHTTPClient supplies the HTTP client used for requests.
// A nil client is rejected by New.
func WithHTTPClient(client *http.Client) Option { return func(c *Client) { c.httpClient = client } }

// New creates a client for project, reading DOPPLER_SERVICE_TOKEN unless overridden.
func New(project string, options ...Option) (*Client, error) {
	c := &Client{project: project, apiKey: os.Getenv("DOPPLER_SERVICE_TOKEN"), httpClient: &http.Client{Timeout: 30 * time.Second}}
	for _, option := range options {
		if option == nil {
			return nil, fmt.Errorf("doppler: nil option")
		}
		option(c)
	}
	if strings.TrimSpace(c.project) == "" {
		return nil, fmt.Errorf("doppler: project is required")
	}
	if strings.TrimSpace(c.apiKey) == "" {
		return nil, fmt.Errorf("doppler: API key is required (set DOPPLER_SERVICE_TOKEN or use WithAPIKey)")
	}
	if c.httpClient == nil {
		return nil, fmt.Errorf("doppler: HTTP client is required")
	}
	return c, nil
}

// ListSecrets returns a copy of defaults with non-empty computed Doppler values
// applied. Only keys present in defaults are requested and returned. Missing or
// empty Doppler values preserve defaults. Any empty value in the resulting map
// causes an error. The input map is never modified, even on error.
// An empty defaults map returns an empty map without making an HTTP request.
func (c *Client) ListSecrets(ctx context.Context, config string, defaults map[string]string) (map[string]string, error) {
	if strings.TrimSpace(config) == "" {
		return nil, fmt.Errorf("doppler: config is required")
	}
	result := make(map[string]string, len(defaults))
	keys := make([]string, 0, len(defaults))
	for key, value := range defaults {
		if key == "" || strings.Contains(key, ",") {
			return nil, fmt.Errorf("doppler: invalid secret key %q", key)
		}
		keys = append(keys, key)
		result[key] = value
	}
	if len(keys) == 0 {
		return result, nil
	}
	sort.Strings(keys)
	query := url.Values{"project": {c.project}, "config": {config}, "secrets": {strings.Join(keys, ",")}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.doppler.com/v3/configs/config/secrets?"+query.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("doppler: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("doppler: list secrets: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Do not expose response bodies, which could contain sensitive data.
		return nil, fmt.Errorf("doppler: list secrets: HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Secrets map[string]struct {
			Computed string `json:"computed"`
		} `json:"secrets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("doppler: decode secrets response: %w", err)
	}
	missing := make([]string, 0)
	for _, key := range keys {
		if value := payload.Secrets[key].Computed; value != "" {
			result[key] = value
		}
		if result[key] == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("doppler: empty required secrets: %s", strings.Join(missing, ", "))
	}
	return result, nil
}
