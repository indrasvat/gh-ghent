package github

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

// AuthError indicates an authentication problem with the GitHub API.
type AuthError struct {
	Err error
}

func (e *AuthError) Error() string {
	return "not authenticated"
}

func (e *AuthError) Unwrap() error {
	return e.Err
}

// RateLimitError indicates the GitHub API rate limit has been exceeded.
type RateLimitError struct {
	ResetAt time.Time
	Err     error
}

func (e *RateLimitError) Error() string {
	if !e.ResetAt.IsZero() {
		return fmt.Sprintf("rate limit exceeded, resets at %s", e.ResetAt.Format(time.Kitchen))
	}
	return "rate limit exceeded"
}

func (e *RateLimitError) Unwrap() error {
	return e.Err
}

// NotFoundError indicates a GitHub resource was not found.
type NotFoundError struct {
	Resource string // e.g., "repository", "pull request"
	Detail   string // e.g., "PR #42 in owner/repo"
	Err      error
}

func (e *NotFoundError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s not found", e.Detail)
	}
	if e.Resource != "" {
		return fmt.Sprintf("%s not found", e.Resource)
	}
	return "resource not found"
}

func (e *NotFoundError) Unwrap() error {
	return e.Err
}

// classifyError inspects an API error and wraps it with a specific error type
// when the cause can be determined (auth, rate limit, not found).
func classifyError(err error) error {
	if err == nil {
		return nil
	}

	// Check go-gh HTTPError (REST and GraphQL HTTP-level errors).
	var httpErr *api.HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.StatusCode {
		case 401:
			return &AuthError{Err: err}
		case 403:
			if isRateLimitMessage(httpErr.Message) {
				return &RateLimitError{
					ResetAt: parseResetFromHeaders(httpErr.Headers),
					Err:     err,
				}
			}
			return &AuthError{Err: err}
		case 404:
			return &NotFoundError{Err: err}
		case 429:
			return &RateLimitError{
				ResetAt: parseResetFromHeaders(httpErr.Headers),
				Err:     err,
			}
		}
		return err
	}

	// Check go-gh GraphQLError.
	var gqlErr *api.GraphQLError
	if errors.As(err, &gqlErr) {
		return classifyGraphQLError(gqlErr, err)
	}

	// Fallback: check error message patterns.
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "authentication") || strings.Contains(msg, "auth token") ||
		strings.Contains(msg, "token not found") {
		return &AuthError{Err: err}
	}

	return err
}

// classifyGraphQLError inspects a GraphQL error for known patterns.
func classifyGraphQLError(gqlErr *api.GraphQLError, original error) error {
	for _, item := range gqlErr.Errors {
		lower := strings.ToLower(item.Message)
		if strings.Contains(lower, "could not resolve to a repository") {
			return &NotFoundError{Resource: "repository", Err: original}
		}
		if strings.Contains(lower, "could not resolve to a pullrequest") {
			return &NotFoundError{Resource: "pull request", Err: original}
		}
		if item.Type == "NOT_FOUND" {
			return &NotFoundError{Err: original}
		}
	}
	return original
}

// classifyWithContext wraps classifyError and adds resource context to NotFoundError.
func classifyWithContext(err error, resource, detail string) error {
	classified := classifyError(err)
	var notFound *NotFoundError
	if errors.As(classified, &notFound) {
		if notFound.Resource == "" {
			notFound.Resource = resource
		}
		if notFound.Detail == "" {
			notFound.Detail = detail
		}
	}
	return classified
}

// isRetryable returns true if the error is a server-side error (5xx) that
// may succeed on retry.
func isRetryable(err error) bool {
	var httpErr *api.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode >= 500
	}
	return false
}

func isRateLimitMessage(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "rate limit")
}
