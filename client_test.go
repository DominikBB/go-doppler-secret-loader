package doppler

import (
	"context"
	"errors"
	"io"
	"maps"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestListSecrets(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       map[string]string
		fail       bool
	}{
		{"computed", `{"secrets":{"DB_PASS":{"raw":"raw","computed":"resolved"},"DB_USER":{"computed":"admin"},"EXTRA":{"computed":"ignored"}}}`, map[string]string{"DB_PASS": "resolved", "DB_USER": "admin"}, false},
		{"missing default", `{"secrets":{"DB_PASS":{"computed":"resolved"}}}`, map[string]string{"DB_PASS": "resolved", "DB_USER": "postgres"}, false},
		{"empty default", `{"secrets":{"DB_PASS":{"computed":"resolved"},"DB_USER":{"computed":""}}}`, map[string]string{"DB_PASS": "resolved", "DB_USER": "postgres"}, false},
		{"missing required", `{"secrets":{}}`, nil, true},
		{"empty required", `{"secrets":{"DB_PASS":{"computed":""}}}`, nil, true},
		{"invalid JSON", `{`, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defaults := map[string]string{"DB_PASS": "", "DB_USER": "postgres"}
			original := maps.Clone(defaults)
			client, err := New("my project", WithAPIKey("custom"), WithHTTPClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" || r.URL.Path != "/v3/configs/config/secrets" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				q := r.URL.Query()
				if q.Get("project") != "my project" || q.Get("config") != "dev config" || q.Get("secrets") != "DB_PASS,DB_USER" {
					t.Errorf("unexpected query: %v", q)
				}
				if r.Header.Get("Authorization") != "Bearer custom" || r.Header.Get("Accept") != "application/json" {
					t.Error("incorrect headers")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			})}))
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.ListSecrets(context.Background(), "dev config", defaults)
			if (err != nil) != tc.fail {
				t.Fatalf("error = %v, want failure %v", err, tc.fail)
			}
			if !maps.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
			if !maps.Equal(defaults, original) {
				t.Fatal("input mutated")
			}
			if got != nil {
				got["DB_USER"] = "changed"
				if defaults["DB_USER"] != "postgres" {
					t.Fatal("result aliases input")
				}
			}
		})
	}
}

func TestNew(t *testing.T) {
	t.Setenv("DOPPLER_SERVICE_TOKEN", "environment")
	c, err := New("project")
	if err != nil || c.apiKey != "environment" {
		t.Fatalf("environment token: %v", err)
	}
	c, err = New("project", WithAPIKey("override"))
	if err != nil || c.apiKey != "override" {
		t.Fatalf("override: %v", err)
	}
	for _, options := range [][]Option{{WithAPIKey("")}, {WithHTTPClient(nil)}, {nil}} {
		if _, err := New("project", options...); err == nil {
			t.Error("expected invalid option error")
		}
	}
	if _, err := New(""); err == nil {
		t.Error("expected project error")
	}
	t.Setenv("DOPPLER_SERVICE_TOKEN", "")
	if _, err := New("project"); err == nil {
		t.Error("expected missing token error")
	}
}

func TestErrorsAndEmptyInput(t *testing.T) {
	sentinel := errors.New("transport failure")
	for _, status := range []int{401, 429, 500, 200} {
		c, _ := New("project", WithAPIKey("token"), WithHTTPClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if status == 200 {
				return nil, sentinel
			}
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("sensitive response")), Header: make(http.Header)}, nil
		})}))
		_, err := c.ListSecrets(context.Background(), "dev", map[string]string{"KEY": "fallback"})
		if err == nil || strings.Contains(err.Error(), "sensitive response") {
			t.Fatalf("unexpected error: %v", err)
		}
		if status == 200 && !errors.Is(err, sentinel) {
			t.Fatal("transport error not wrapped")
		}
	}
	c, _ := New("project", WithAPIKey("token"), WithHTTPClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) { t.Fatal("unexpected request"); return nil, nil })}))
	got, err := c.ListSecrets(context.Background(), "dev", nil)
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty input: %v %v", got, err)
	}
	for _, tc := range []struct{ config, key string }{{"", "KEY"}, {"dev", ""}, {"dev", "A,B"}} {
		if _, err := c.ListSecrets(context.Background(), tc.config, map[string]string{tc.key: "value"}); err == nil {
			t.Error("expected input error")
		}
	}
}

func TestCancellation(t *testing.T) {
	c, _ := New("project", WithAPIKey("token"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.ListSecrets(ctx, "dev", map[string]string{"KEY": ""})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}
