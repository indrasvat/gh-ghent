package github

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

func TestAuthErrorMessage(t *testing.T) {
	err := &AuthError{Err: fmt.Errorf("original")}
	if err.Error() != "not authenticated" {
		t.Errorf("AuthError.Error() = %q, want %q", err.Error(), "not authenticated")
	}
	if err.Unwrap().Error() != "original" {
		t.Errorf("AuthError.Unwrap() = %q, want %q", err.Unwrap().Error(), "original")
	}
}

func TestRateLimitErrorMessage(t *testing.T) {
	tests := []struct {
		name    string
		resetAt time.Time
		want    string
	}{
		{
			name:    "with reset time",
			resetAt: time.Date(2026, 2, 22, 15, 30, 0, 0, time.UTC),
			want:    "rate limit exceeded, resets at 3:30PM",
		},
		{
			name: "without reset time",
			want: "rate limit exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &RateLimitError{ResetAt: tt.resetAt}
			if err.Error() != tt.want {
				t.Errorf("RateLimitError.Error() = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestNotFoundErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		detail   string
		want     string
	}{
		{name: "with detail", resource: "pull request", detail: "PR #42 in owner/repo", want: "PR #42 in owner/repo not found"},
		{name: "with resource only", resource: "repository", want: "repository not found"},
		{name: "bare", want: "resource not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &NotFoundError{Resource: tt.resource, Detail: tt.detail}
			if err.Error() != tt.want {
				t.Errorf("NotFoundError.Error() = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func testReqURL() *url.URL {
	u, _ := url.Parse("https://api.github.com/repos/owner/repo")
	return u
}

func TestClassifyError_HTTPErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
		wantType   string
	}{
		{name: "401 auth", statusCode: 401, message: "Bad credentials", wantType: "auth"},
		{name: "403 rate limit", statusCode: 403, message: "API rate limit exceeded", wantType: "ratelimit"},
		{name: "403 permissions", statusCode: 403, message: "Resource not accessible", wantType: "auth"},
		{name: "404 not found", statusCode: 404, message: "Not Found", wantType: "notfound"},
		{name: "429 throttle", statusCode: 429, message: "Too Many Requests", wantType: "ratelimit"},
		{name: "500 passthrough", statusCode: 500, message: "Internal Server Error", wantType: "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpErr := &api.HTTPError{
				StatusCode: tt.statusCode,
				Message:    tt.message,
				RequestURL: testReqURL(),
			}
			classified := classifyError(httpErr)

			switch tt.wantType {
			case "auth":
				var target *AuthError
				if !errors.As(classified, &target) {
					t.Errorf("expected AuthError, got %T: %v", classified, classified)
				}
			case "ratelimit":
				var target *RateLimitError
				if !errors.As(classified, &target) {
					t.Errorf("expected RateLimitError, got %T: %v", classified, classified)
				}
			case "notfound":
				var target *NotFoundError
				if !errors.As(classified, &target) {
					t.Errorf("expected NotFoundError, got %T: %v", classified, classified)
				}
			case "other":
				var a *AuthError
				var r *RateLimitError
				var n *NotFoundError
				if errors.As(classified, &a) || errors.As(classified, &r) || errors.As(classified, &n) {
					t.Errorf("expected passthrough, got %T: %v", classified, classified)
				}
			}
		})
	}
}

func TestClassifyError_GraphQLErrors(t *testing.T) {
	tests := []struct {
		name    string
		message string
		errType string
	}{
		{
			name:    "repo not found",
			message: "Could not resolve to a Repository with the name 'nonexistent/repo'.",
			errType: "NOT_FOUND",
		},
		{
			name:    "NOT_FOUND type",
			message: "Something not found.",
			errType: "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gqlErr := &api.GraphQLError{
				Errors: []api.GraphQLErrorItem{
					{Message: tt.message, Type: tt.errType},
				},
			}
			classified := classifyError(gqlErr)

			var nfErr *NotFoundError
			if !errors.As(classified, &nfErr) {
				t.Errorf("expected NotFoundError, got %T: %v", classified, classified)
			}
		})
	}
}

func TestClassifyWithContext(t *testing.T) {
	httpErr := &api.HTTPError{StatusCode: 404, Message: "Not Found", RequestURL: testReqURL()}
	classified := classifyWithContext(httpErr, "pull request", "PR #42 in owner/repo")

	var nfErr *NotFoundError
	if !errors.As(classified, &nfErr) {
		t.Fatalf("expected NotFoundError, got %T: %v", classified, classified)
	}
	if nfErr.Resource != "pull request" {
		t.Errorf("Resource = %q, want %q", nfErr.Resource, "pull request")
	}
	if nfErr.Detail != "PR #42 in owner/repo" {
		t.Errorf("Detail = %q, want %q", nfErr.Detail, "PR #42 in owner/repo")
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{name: "500", statusCode: 500, want: true},
		{name: "502", statusCode: 502, want: true},
		{name: "503", statusCode: 503, want: true},
		{name: "404", statusCode: 404, want: false},
		{name: "401", statusCode: 401, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpErr := &api.HTTPError{StatusCode: tt.statusCode, RequestURL: testReqURL()}
			if got := isRetryable(httpErr); got != tt.want {
				t.Errorf("isRetryable(%d) = %v, want %v", tt.statusCode, got, tt.want)
			}
		})
	}

	t.Run("non-HTTP error", func(t *testing.T) {
		if isRetryable(fmt.Errorf("random error")) {
			t.Error("non-HTTP error should not be retryable")
		}
	})
}

func TestDoWithRetry(t *testing.T) {
	// Speed up retry for testing.
	orig := retryBackoff
	retryBackoff = time.Millisecond
	defer func() { retryBackoff = orig }()

	t.Run("success first try", func(t *testing.T) {
		calls := 0
		err := doWithRetry(func() error {
			calls++
			return nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if calls != 1 {
			t.Errorf("calls = %d, want 1", calls)
		}
	})

	t.Run("retry on 5xx", func(t *testing.T) {
		calls := 0
		err := doWithRetry(func() error {
			calls++
			if calls == 1 {
				return &api.HTTPError{StatusCode: 500, Message: "Server Error", RequestURL: testReqURL()}
			}
			return nil
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if calls != 2 {
			t.Errorf("calls = %d, want 2", calls)
		}
	})

	t.Run("no retry on 4xx", func(t *testing.T) {
		calls := 0
		err := doWithRetry(func() error {
			calls++
			return &api.HTTPError{StatusCode: 404, Message: "Not Found", RequestURL: testReqURL()}
		})
		if err == nil {
			t.Error("expected error")
		}
		if calls != 1 {
			t.Errorf("calls = %d, want 1 (no retry on 4xx)", calls)
		}
	})

	t.Run("both retries fail", func(t *testing.T) {
		calls := 0
		err := doWithRetry(func() error {
			calls++
			return &api.HTTPError{StatusCode: 503, Message: "Unavailable", RequestURL: testReqURL()}
		})
		if err == nil {
			t.Error("expected error")
		}
		if calls != 2 {
			t.Errorf("calls = %d, want 2 (initial + 1 retry)", calls)
		}
	})
}

func TestClassifyError_Nil(t *testing.T) {
	if got := classifyError(nil); got != nil {
		t.Errorf("classifyError(nil) = %v, want nil", got)
	}
}

func TestErrorUnwrap(t *testing.T) {
	original := fmt.Errorf("original error")

	t.Run("AuthError", func(t *testing.T) {
		err := &AuthError{Err: original}
		if !errors.Is(err, original) {
			t.Error("AuthError should unwrap to original")
		}
	})

	t.Run("RateLimitError", func(t *testing.T) {
		err := &RateLimitError{Err: original}
		if !errors.Is(err, original) {
			t.Error("RateLimitError should unwrap to original")
		}
	})

	t.Run("NotFoundError", func(t *testing.T) {
		err := &NotFoundError{Err: original}
		if !errors.Is(err, original) {
			t.Error("NotFoundError should unwrap to original")
		}
	})
}

func TestClassifyError_RateLimitWithResetHeader(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-RateLimit-Reset", "1708617600")

	httpErr := &api.HTTPError{
		StatusCode: 429,
		Message:    "Too Many Requests",
		RequestURL: testReqURL(),
		Headers:    headers,
	}

	classified := classifyError(httpErr)

	var rlErr *RateLimitError
	if !errors.As(classified, &rlErr) {
		t.Fatalf("expected RateLimitError, got %T: %v", classified, classified)
	}
	if rlErr.ResetAt.IsZero() {
		t.Error("expected non-zero ResetAt from header")
	}
}
