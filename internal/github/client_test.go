package github

import (
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

func TestNewWithOptions(t *testing.T) {
	gql, err := api.NewGraphQLClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "fake-token",
	})
	if err != nil {
		t.Fatalf("creating test GQL client: %v", err)
	}

	rest, err := api.NewRESTClient(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "fake-token",
	})
	if err != nil {
		t.Fatalf("creating test REST client: %v", err)
	}

	c, err := New(WithGraphQLClient(gql), WithRESTClient(rest))
	if err != nil {
		t.Fatalf("New() with options: %v", err)
	}

	if c.gql != gql {
		t.Error("GraphQL client not set by option")
	}
	if c.rest != rest {
		t.Error("REST client not set by option")
	}
}

// FetchReviews is no longer a stub — it's fully implemented in reviews.go.
// See reviews_test.go for its tests.
