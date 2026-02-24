// Package github provides the GitHub API adapter using go-gh.
package github

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"

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

// New creates a GitHub client with defaults from go-gh.
// Use options to inject mock clients for testing.
func New(opts ...Option) (*Client, error) {
	c := &Client{}
	for _, opt := range opts {
		opt(c)
	}
	if c.gql == nil {
		clientOpts := api.ClientOptions{}
		if debug.Enabled() {
			clientOpts.Log = os.Stderr
			clientOpts.LogVerboseHTTP = true
			clientOpts.LogColorize = true
		}
		gql, err := api.NewGraphQLClient(clientOpts)
		if err != nil {
			return nil, classifyError(err)
		}
		c.gql = gql
	}
	if c.rest == nil {
		clientOpts := api.ClientOptions{}
		if debug.Enabled() {
			clientOpts.Log = os.Stderr
			clientOpts.LogVerboseHTTP = true
			clientOpts.LogColorize = true
		}
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
	_ domain.ThreadFetcher  = (*Client)(nil)
	_ domain.CheckFetcher   = (*Client)(nil)
	_ domain.ThreadResolver = (*Client)(nil)
	_ domain.ThreadReplier  = (*Client)(nil)
	_ domain.ReviewFetcher  = (*Client)(nil)
)

// ResolveThread and UnresolveThread are implemented in resolve.go.

// ReplyToThread is implemented in reply.go.

// FetchReviews is implemented in reviews.go.
