package github

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
)

const rateLimitWarningThreshold = 100

// RateLimitInfo holds rate limit data parsed from HTTP response headers.
type RateLimitInfo struct {
	Limit     int
	Remaining int
	ResetAt   time.Time
}

// parseRateLimitHeaders extracts rate limit info from GitHub API response headers.
func parseRateLimitHeaders(h http.Header) *RateLimitInfo {
	if h == nil {
		return nil
	}

	remaining := h.Get("X-RateLimit-Remaining")
	if remaining == "" {
		return nil
	}

	info := &RateLimitInfo{}
	if v, err := strconv.Atoi(remaining); err == nil {
		info.Remaining = v
	}
	if v, err := strconv.Atoi(h.Get("X-RateLimit-Limit")); err == nil {
		info.Limit = v
	}
	if v, err := strconv.ParseInt(h.Get("X-RateLimit-Reset"), 10, 64); err == nil {
		info.ResetAt = time.Unix(v, 0)
	}

	return info
}

// parseResetFromHeaders extracts the rate limit reset time from headers.
func parseResetFromHeaders(h http.Header) time.Time {
	if h == nil {
		return time.Time{}
	}
	if v, err := strconv.ParseInt(h.Get("X-RateLimit-Reset"), 10, 64); err == nil {
		return time.Unix(v, 0)
	}
	return time.Time{}
}

// warnIfLowRateLimit logs a warning to stderr if remaining API calls are below the threshold.
func warnIfLowRateLimit(info *RateLimitInfo) {
	if info == nil {
		return
	}
	if info.Remaining < rateLimitWarningThreshold {
		slog.Debug("rate limit low", "remaining", info.Remaining, "limit", info.Limit, "reset", info.ResetAt)
		fmt.Fprintf(os.Stderr, "Warning: GitHub API rate limit low (%d/%d remaining, resets at %s)\n",
			info.Remaining, info.Limit, info.ResetAt.Format(time.Kitchen))
	}
}

// checkRateLimit returns a RateLimitError if the rate limit is exhausted.
func checkRateLimit(info *RateLimitInfo) error {
	if info == nil {
		return nil
	}
	if info.Remaining == 0 {
		return &RateLimitError{
			ResetAt: info.ResetAt,
		}
	}
	warnIfLowRateLimit(info)
	return nil
}
