// Package github provides the GitHub API adapter using go-gh.
package github

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/auth"

	"github.com/indrasvat/gh-ghent/internal/debug"
	"github.com/indrasvat/gh-ghent/internal/domain"
)

const defaultTimeout = 30 * time.Second

// retryBackoff is the delay between retry attempts. Package-level for testing.
var retryBackoff = 1 * time.Second

// doWithRetry executes fn, retrying once with backoff on server errors (5xx).
func doWithRetry(fn func() error) error {
	err := fn()
	if err == nil {
		return nil
	}
	if !isRetryable(err) {
		return err
	}
	slog.Debug("retrying after server error", "error", err)
	time.Sleep(retryBackoff)
	return fn()
}

// withTimeout returns a context with the default API timeout applied.
func withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, defaultTimeout)
}

// Client wraps go-gh GraphQL and REST clients to implement domain port interfaces.
type Client struct {
	gql  *api.GraphQLClient
	rest *api.RESTClient
}

// Option configures the Client.
type Option func(*Client)

// WithGraphQLClient sets a custom GraphQL client (for testing).
func WithGraphQLClient(c *api.GraphQLClient) Option {
	return func(client *Client) {
		client.gql = c
	}
}

// WithRESTClient sets a custom REST client (for testing).
func WithRESTClient(c *api.RESTClient) Option {
	return func(client *Client) {
		client.rest = c
	}
}

// authSchemeEnv selects the Authorization scheme: "token" (default, what
// gh sends) or "bearer", for credential vaults that only swap the real token
// into Bearer headers.
const authSchemeEnv = "GH_GHENT_AUTH_SCHEME"

// clientOptions resolves the host and token the way gh does, and applies the
// auth scheme from GH_GHENT_AUTH_SCHEME. A missing token is an *AuthError
// with NoToken set, so the CLI can say that a token is needed at all.
func clientOptions() (api.ClientOptions, error) {
	host, _ := auth.DefaultHost()
	token, _ := auth.TokenForHost(host)
	if token == "" {
		return api.ClientOptions{}, &AuthError{NoToken: true, Err: fmt.Errorf("no token for host %s", host)}
	}
	opts := api.ClientOptions{Host: host, AuthToken: token}
	switch scheme := os.Getenv(authSchemeEnv); strings.ToLower(scheme) {
	case "", "token":
		// go-gh sends "token <token>" itself.
	case "bearer":
		// go-gh never overrides a preset Authorization header, and still only
		// sends it to the resolved host.
		opts.Headers = map[string]string{"Authorization": "Bearer " + token}
	default:
		return api.ClientOptions{}, fmt.Errorf("%s=%q is not supported: use \"token\" or \"bearer\"", authSchemeEnv, scheme)
	}
	if debug.Enabled() {
		opts.Log = os.Stderr
		opts.LogVerboseHTTP = true
		opts.LogColorize = true
	}
	return opts, nil
}

// New creates a GitHub client with defaults from go-gh.
// Use options to inject mock clients for testing.
func New(opts ...Option) (*Client, error) {
	c := &Client{}
	for _, opt := range opts {
		opt(c)
	}
	if c.gql != nil && c.rest != nil {
		return c, nil
	}
	clientOpts, err := clientOptions()
	if err != nil {
		return nil, err
	}
	if c.gql == nil {
		gql, err := api.NewGraphQLClient(clientOpts)
		if err != nil {
			return nil, classifyError(err)
		}
		c.gql = gql
	}
	if c.rest == nil {
		rest, err := api.NewRESTClient(clientOpts)
		if err != nil {
			return nil, classifyError(err)
		}
		c.rest = rest
	}
	return c, nil
}

// Compile-time interface satisfaction checks.
var (
	_ domain.ThreadFetcher   = (*Client)(nil)
	_ domain.CheckFetcher    = (*Client)(nil)
	_ domain.ThreadResolver  = (*Client)(nil)
	_ domain.ThreadReplier   = (*Client)(nil)
	_ domain.ReviewFetcher   = (*Client)(nil)
	_ domain.ReviewDismisser = (*Client)(nil)
	_ domain.ActivityProber  = (*Client)(nil)
)

// ResolveThread and UnresolveThread are implemented in resolve.go.

// ReplyToThread is implemented in reply.go.

// FetchReviews is implemented in reviews.go.
