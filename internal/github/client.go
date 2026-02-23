// Package github provides the GitHub API adapter using go-gh.
package github

import (
	"context"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/indrasvat/ghent/internal/domain"
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

func (c *Client) FetchThreads(ctx context.Context, owner, repo string, pr int) (*domain.CommentsResult, error) {
	return nil, fmt.Errorf("FetchThreads: not implemented")
}

func (c *Client) FetchChecks(ctx context.Context, owner, repo string, pr int) (*domain.ChecksResult, error) {
	return nil, fmt.Errorf("FetchChecks: not implemented")
}

func (c *Client) ResolveThread(ctx context.Context, threadID string) error {
	return fmt.Errorf("ResolveThread: not implemented")
}

func (c *Client) UnresolveThread(ctx context.Context, threadID string) error {
	return fmt.Errorf("UnresolveThread: not implemented")
}

func (c *Client) ReplyToThread(ctx context.Context, owner, repo string, pr int, threadID, body string) (*domain.ReplyResult, error) {
	return nil, fmt.Errorf("ReplyToThread: not implemented")
}

func (c *Client) FetchReviews(ctx context.Context, owner, repo string, pr int) ([]domain.Review, error) {
	return nil, fmt.Errorf("FetchReviews: not implemented")
}
