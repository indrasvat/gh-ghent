package github

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestParseRateLimitHeaders(t *testing.T) {
	t.Run("valid headers", func(t *testing.T) {
		h := http.Header{}
		h.Set("X-RateLimit-Limit", "5000")
		h.Set("X-RateLimit-Remaining", "4500")
		h.Set("X-RateLimit-Reset", "1708617600")

		info := parseRateLimitHeaders(h)
		if info == nil {
			t.Fatal("expected non-nil RateLimitInfo")
		}
		if info.Limit != 5000 {
			t.Errorf("Limit = %d, want 5000", info.Limit)
		}
		if info.Remaining != 4500 {
			t.Errorf("Remaining = %d, want 4500", info.Remaining)
		}
		if info.ResetAt.IsZero() {
			t.Error("expected non-zero ResetAt")
		}
	})

	t.Run("nil headers", func(t *testing.T) {
		info := parseRateLimitHeaders(nil)
		if info != nil {
			t.Errorf("expected nil, got %+v", info)
		}
	})

	t.Run("missing remaining", func(t *testing.T) {
		h := http.Header{}
		h.Set("X-RateLimit-Limit", "5000")
		info := parseRateLimitHeaders(h)
		if info != nil {
			t.Errorf("expected nil when Remaining header missing, got %+v", info)
		}
	})
}

func TestCheckRateLimit(t *testing.T) {
	t.Run("nil info", func(t *testing.T) {
		if err := checkRateLimit(nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("remaining zero", func(t *testing.T) {
		info := &RateLimitInfo{
			Remaining: 0,
			ResetAt:   time.Now().Add(5 * time.Minute),
		}
		err := checkRateLimit(info)
		if err == nil {
			t.Fatal("expected RateLimitError")
		}
		var rlErr *RateLimitError
		if !errors.As(err, &rlErr) {
			t.Errorf("expected RateLimitError, got %T", err)
		}
	})

	t.Run("remaining above threshold", func(t *testing.T) {
		info := &RateLimitInfo{
			Remaining: 4000,
			Limit:     5000,
		}
		if err := checkRateLimit(info); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestParseResetFromHeaders(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		h := http.Header{}
		h.Set("X-RateLimit-Reset", "1708617600")
		got := parseResetFromHeaders(h)
		want := time.Unix(1708617600, 0)
		if !got.Equal(want) {
			t.Errorf("parseResetFromHeaders = %v, want %v", got, want)
		}
	})

	t.Run("nil headers", func(t *testing.T) {
		got := parseResetFromHeaders(nil)
		if !got.IsZero() {
			t.Errorf("expected zero time, got %v", got)
		}
	})

	t.Run("missing header", func(t *testing.T) {
		h := http.Header{}
		got := parseResetFromHeaders(h)
		if !got.IsZero() {
			t.Errorf("expected zero time, got %v", got)
		}
	})
}
