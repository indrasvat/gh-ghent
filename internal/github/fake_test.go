package github

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

// fakeResponse is one canned REST reply.
type fakeResponse struct {
	status int
	body   string
}

// fakeREST answers requests by path+query ("repos/o/r/pulls?head=..."), and
// records every path it was asked for. Unknown paths are a 404, so a test
// fails loudly when code calls an endpoint the test did not expect.
type fakeREST struct {
	mu     sync.Mutex
	routes map[string]fakeResponse
	calls  []string
}

func (f *fakeREST) RoundTrip(req *http.Request) (*http.Response, error) {
	key := strings.TrimPrefix(req.URL.Path, "/")
	if req.URL.RawQuery != "" {
		key += "?" + req.URL.RawQuery
	}
	f.mu.Lock()
	f.calls = append(f.calls, key)
	r, ok := f.routes[key]
	f.mu.Unlock()
	if !ok {
		r = fakeResponse{status: 404, body: `{"message":"Not Found"}`}
	}
	return &http.Response{
		StatusCode: r.status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(r.body)),
		Request:    req,
	}, nil
}

func newFakeClient(t *testing.T, routes map[string]fakeResponse) (*Client, *fakeREST) {
	t.Helper()
	f := &fakeREST{routes: routes}
	opts := api.ClientOptions{Host: "github.com", AuthToken: "x", Transport: f, LogIgnoreEnv: true}
	rest, err := api.NewRESTClient(opts)
	if err != nil {
		t.Fatalf("NewRESTClient: %v", err)
	}
	gql, err := api.NewGraphQLClient(opts)
	if err != nil {
		t.Fatalf("NewGraphQLClient: %v", err)
	}
	c, err := New(WithRESTClient(rest), WithGraphQLClient(gql))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c, f
}
