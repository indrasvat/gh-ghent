// Package github provides the GitHub API adapter using go-gh.
package github

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

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
		gql, err := api.DefaultGraphQLClient()
		if err != nil {
			return nil, fmt.Errorf("graphql client: %w", err)
		}
		c.gql = gql
	}
	if c.rest == nil {
		rest, err := api.DefaultRESTClient()
		if err != nil {
			return nil, fmt.Errorf("rest client: %w", err)
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
