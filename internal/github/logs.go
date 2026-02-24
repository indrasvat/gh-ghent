package github

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

// maxExcerptLines is the maximum number of lines in an extracted log excerpt.
const maxExcerptLines = 50

// ansiRegexp matches ANSI escape sequences.
var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// timestampRegexp matches GitHub Actions timestamp prefixes like:
// 2025-01-01T00:00:00.0000000Z
var timestampRegexp = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z\s?`)

// errorPatterns are substrings (lowercased) that indicate error-relevant lines.
var errorPatterns = []string{
	"error",
	"fail",
	"fatal",
	"panic",
}

// errorPrefixes are exact prefixes (case-insensitive) that indicate error lines.
var errorPrefixes = []string{
	"error:",
	"fail:",
	"##[error]",
}

// fileLineRegexp matches file:line patterns like foo.go:42: or src/main.rs:10:5:
var fileLineRegexp = regexp.MustCompile(`\S+\.\w+:\d+`)

// FetchJobLog fetches the plain-text log for a GitHub Actions job via REST.
// The endpoint returns a 302 redirect to the log content; go-gh follows redirects
// automatically. We use RequestWithContext to get the raw response since the
// endpoint returns plain text, not JSON.
func (c *Client) FetchJobLog(ctx context.Context, owner, repo string, jobID int64) (string, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	start := time.Now()
	slog.Debug("fetching job log", "owner", owner, "repo", repo, "jobID", jobID)

	path := fmt.Sprintf("repos/%s/%s/actions/jobs/%d/logs", owner, repo, jobID)
	resp, err := c.rest.RequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return "", classifyError(err)
	}
	defer resp.Body.Close()

	// Check rate limit from response headers.
	if rlInfo := parseRateLimitHeaders(resp.Header); rlInfo != nil {
		if rlErr := checkRateLimit(rlInfo); rlErr != nil {
			return "", rlErr
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read job log for %d: %w", jobID, err)
	}

	slog.Debug("fetched job log", "jobID", jobID, "bytes", len(body), "duration", time.Since(start))

	return string(body), nil
}

// ExtractErrorLines extracts error-relevant lines from a job log.
// It looks for lines containing error keywords, file:line patterns,
// and common error prefixes. Context lines (1 before, 1 after) are included
// around each match. The result is truncated to maxExcerptLines.
func ExtractErrorLines(log string) string {
	lines := strings.Split(log, "\n")

	// Clean all lines first: strip ANSI codes and timestamps
	cleaned := make([]string, len(lines))
	for i, line := range lines {
		cleaned[i] = cleanLine(line)
	}

	// Find indices of error-relevant lines
	matchSet := make(map[int]bool)
	for i, line := range cleaned {
		if isErrorLine(line) {
			// Include the match and 1 line of context on each side
			for j := max(0, i-1); j <= min(len(cleaned)-1, i+1); j++ {
				matchSet[j] = true
			}
		}
	}

	if len(matchSet) == 0 {
		return ""
	}

	// Collect matched lines in order, inserting "..." for gaps
	var result []string
	prevIdx := -2

	for i := range len(cleaned) {
		if !matchSet[i] {
			continue
		}

		// Insert gap marker if there's a discontinuity
		if prevIdx >= 0 && i > prevIdx+1 {
			result = append(result, "...")
		}

		result = append(result, cleaned[i])
		prevIdx = i

		if len(result) >= maxExcerptLines {
			break
		}
	}

	return strings.Join(result, "\n")
}

// cleanLine strips ANSI escape codes and GitHub Actions timestamp prefixes.
func cleanLine(line string) string {
	line = ansiRegexp.ReplaceAllString(line, "")
	line = timestampRegexp.ReplaceAllString(line, "")
	return line
}

// isErrorLine checks whether a line is error-relevant.
func isErrorLine(line string) bool {
	lower := strings.ToLower(line)

	// Check error prefixes (case-insensitive, trimmed)
	trimmed := strings.TrimSpace(lower)
	for _, prefix := range errorPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}

	// Check error pattern substrings
	for _, pattern := range errorPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	// Check file:line patterns
	if fileLineRegexp.MatchString(line) {
		return true
	}

	return false
}
