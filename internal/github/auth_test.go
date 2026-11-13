package github

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

// isolateAuthEnv points go-gh at an empty config and no gh binary, so only
// the variables a test sets can supply a token.
func isolateAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GH_CONFIG_DIR", t.TempDir())
	t.Setenv("GH_PATH", "/nonexistent/gh")
	t.Setenv("GH_HOST", "github.com")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_ENTERPRISE_TOKEN", "")
	t.Setenv("GITHUB_ENTERPRISE_TOKEN", "")
	t.Setenv(AuthSchemeEnv, "")
}

type captureTransport struct{ got http.Header }

func (c *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.got = req.Header.Clone()
	return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("{}")), Request: req}, nil
}

// TestAuthSchemeOnTheWire drives the options through go-gh's real REST
// client and checks the Authorization header that actually leaves the process.
// A vault that only swaps Bearer headers depends on this exact value.
func TestAuthSchemeOnTheWire(t *testing.T) {
	tests := []struct {
		scheme string
		want   string
	}{
		{scheme: "", want: "token s3cret"},
		{scheme: "token", want: "token s3cret"},
		{scheme: "bearer", want: "Bearer s3cret"},
		{scheme: "Bearer", want: "Bearer s3cret"},
	}
	for _, tt := range tests {
		t.Run("scheme="+tt.scheme, func(t *testing.T) {
			isolateAuthEnv(t)
			t.Setenv("GH_TOKEN", "s3cret")
			t.Setenv(AuthSchemeEnv, tt.scheme)

			opts, err := clientOptions()
			if err != nil {
				t.Fatalf("clientOptions: %v", err)
			}
			ct := &captureTransport{}
			opts.Transport = ct
			rest, err := api.NewRESTClient(opts)
			if err != nil {
				t.Fatalf("NewRESTClient: %v", err)
			}
			if err := rest.Get("rate_limit", nil); err != nil {
				t.Fatalf("GET: %v", err)
			}
			if got := ct.got.Get("Authorization"); got != tt.want {
				t.Errorf("Authorization = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAuthSchemeInvalid(t *testing.T) {
	isolateAuthEnv(t)
	t.Setenv("GH_TOKEN", "s3cret")
	t.Setenv(AuthSchemeEnv, "basic")
	if _, err := clientOptions(); err == nil {
		t.Fatal("want error for unsupported scheme, got nil")
	}
}

func TestNoTokenIsAuthError(t *testing.T) {
	isolateAuthEnv(t)
	_, err := New()
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("New() error = %v (%T), want *AuthError", err, err)
	}
	if !authErr.NoToken {
		t.Errorf("AuthError.NoToken = false, want true for a missing token")
	}
}
