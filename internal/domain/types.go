package domain

import "time"

// ReviewThread represents a PR review thread from GitHub.
type ReviewThread struct {
	ID                 string    `json:"id"`
	Path               string    `json:"path"`
	Line               int       `json:"line"`
	StartLine          int       `json:"start_line,omitempty"`
	DiffSide           string    `json:"diff_side,omitempty"`
	IsResolved         bool      `json:"is_resolved"`
	IsOutdated         bool      `json:"is_outdated"`
	ViewerCanResolve   bool      `json:"viewer_can_resolve"`
	ViewerCanUnresolve bool      `json:"viewer_can_unresolve"`
	ViewerCanReply     bool      `json:"viewer_can_reply"`
	Comments           []Comment `json:"comments"`
}

// Comment represents a single comment within a review thread.
type Comment struct {
	ID         string    `json:"id"`
	DatabaseID int64     `json:"database_id"` // needed by REST reply endpoint
	Author     string    `json:"author"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
	URL        string    `json:"url"`
	DiffHunk   string    `json:"diff_hunk,omitempty"`
	Path       string    `json:"path,omitempty"`
}

// CommentsResult wraps the result of fetching review threads.
type CommentsResult struct {
	PRNumber        int            `json:"pr_number"`
	Threads         []ReviewThread `json:"threads"`
	TotalCount      int            `json:"total_count"`
	ResolvedCount   int            `json:"resolved_count"`
	UnresolvedCount int            `json:"unresolved_count"`
	Since           string         `json:"since,omitempty"`
}

// CommentGroup represents a group of threads under a common key.
type CommentGroup struct {
	Key     string         `json:"key"`
	Threads []ReviewThread `json:"threads"`
}

// GroupedCommentsResult wraps grouped comment output.
type GroupedCommentsResult struct {
	PRNumber        int            `json:"pr_number"`
	GroupBy         string         `json:"group_by"`
	Groups          []CommentGroup `json:"groups"`
	TotalCount      int            `json:"total_count"`
	ResolvedCount   int            `json:"resolved_count"`
	UnresolvedCount int            `json:"unresolved_count"`
}

// OverallStatus represents the aggregate CI status.
type OverallStatus string

const (
	StatusPass    OverallStatus = "pass"
	StatusFail    OverallStatus = "failure"
	StatusPending OverallStatus = "pending"
)

// IsFailConclusion returns true if a check run conclusion is classified as a failure.
// This includes: failure, timed_out, action_required, startup_failure, stale, cancelled.
// Matches the classification in github.classifyCheckStatus.
func IsFailConclusion(conclusion string) bool {
	switch conclusion {
	case "failure", "timed_out", "action_required", "startup_failure", "stale", "cancelled":
		return true
	default:
		return false
	}
}

// AggregateStatus returns the highest-priority status: fail > pending > pass.
func AggregateStatus(statuses []OverallStatus) OverallStatus {
	result := StatusPass
	for _, s := range statuses {
		switch s {
		case StatusFail:
			return StatusFail
		case StatusPending:
			result = StatusPending
		}
	}
	return result
}

// CheckRun represents a CI check run.
type CheckRun struct {
	ID          int64        `json:"id"`
	Name        string       `json:"name"`
	Status      string       `json:"status"`     // queued, in_progress, completed
	Conclusion  string       `json:"conclusion"` // success, failure, neutral, cancelled, skipped, timed_out, action_required
	StartedAt   time.Time    `json:"started_at"`
	CompletedAt time.Time    `json:"completed_at,omitzero"`
	HTMLURL     string       `json:"html_url"`
	Annotations []Annotation `json:"annotations,omitempty"`
	LogExcerpt  string       `json:"log_excerpt,omitempty"`
}

// Annotation represents a check run annotation (lint error, test failure, etc.).
type Annotation struct {
	Path            string `json:"path"`
	StartLine       int    `json:"start_line"`
	EndLine         int    `json:"end_line"`
	AnnotationLevel string `json:"annotation_level"` // notice, warning, failure
	Title           string `json:"title"`
	Message         string `json:"message"`
}

// ChecksResult wraps the result of fetching check runs.
type ChecksResult struct {
	PRNumber      int           `json:"pr_number"`
	HeadSHA       string        `json:"head_sha"`
	OverallStatus OverallStatus `json:"overall_status"`
	Checks        []CheckRun    `json:"checks"`
	PassCount     int           `json:"pass_count"`
	FailCount     int           `json:"fail_count"`
	PendingCount  int           `json:"pending_count"`
	Since         string        `json:"since,omitempty"`
}

// ReviewState represents the state of a PR review.
type ReviewState string

const (
	ReviewApproved         ReviewState = "APPROVED"
	ReviewChangesRequested ReviewState = "CHANGES_REQUESTED"
	ReviewCommented        ReviewState = "COMMENTED"
	ReviewPending          ReviewState = "PENDING"
	ReviewDismissed        ReviewState = "DISMISSED"
)

// Review represents a pull request review.
type Review struct {
	ID          string      `json:"id"`
	Author      string      `json:"author"`
	State       ReviewState `json:"state"`
	Body        string      `json:"body,omitempty"`
	SubmittedAt time.Time   `json:"submitted_at"`
}

// ReplyResult represents the result of posting a reply to a thread.
type ReplyResult struct {
	ThreadID  string    `json:"thread_id"`
	CommentID int64     `json:"comment_id"`
	URL       string    `json:"url"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// ResolveResult represents the result of resolving/unresolving a single thread.
type ResolveResult struct {
	ThreadID   string `json:"thread_id"`
	Path       string `json:"path"`
	Line       int    `json:"line"`
	IsResolved bool   `json:"is_resolved"`
	Action     string `json:"action"` // "resolved" or "unresolved"
}

// ResolveError records a per-thread resolution failure.
type ResolveError struct {
	ThreadID string `json:"thread_id"`
	Message  string `json:"message"`
}

// ResolveResults wraps results from one or more resolve/unresolve operations.
type ResolveResults struct {
	Results      []ResolveResult `json:"results"`
	SuccessCount int             `json:"success_count"`
	FailureCount int             `json:"failure_count"`
	SkippedCount int             `json:"skipped_count,omitempty"`
	Errors       []ResolveError  `json:"errors,omitempty"`
	DryRun       bool            `json:"dry_run,omitempty"`
}

// WatchEvent represents a single check status change during watch mode.
type WatchEvent struct {
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	Conclusion string    `json:"conclusion"`
	Timestamp  time.Time `json:"timestamp"`
}

// WatchStatus represents the aggregate status emitted on each poll cycle.
type WatchStatus struct {
	Timestamp     time.Time     `json:"timestamp"`
	OverallStatus OverallStatus `json:"overall_status"`
	Completed     int           `json:"completed"`
	Total         int           `json:"total"`
	PassCount     int           `json:"pass_count"`
	FailCount     int           `json:"fail_count"`
	PendingCount  int           `json:"pending_count"`
	Events        []WatchEvent  `json:"events,omitempty"`
	Final         bool          `json:"final"`
}

// SummaryResult combines all PR data for the summary command.
type SummaryResult struct {
	PRNumber     int            `json:"pr_number"`
	Comments     CommentsResult `json:"comments"`
	Checks       ChecksResult   `json:"checks"`
	Reviews      []Review       `json:"reviews"`
	IsMergeReady bool           `json:"is_merge_ready"`
	PRAge        string         `json:"pr_age,omitempty"`
	LastUpdate   string         `json:"last_update,omitempty"`
	ReviewCycles int            `json:"review_cycles,omitempty"`
}
