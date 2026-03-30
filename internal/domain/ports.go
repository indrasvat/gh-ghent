package domain

import (
	"context"
	"io"
)

// ThreadFetcher fetches review threads for a PR.
type ThreadFetcher interface {
	FetchThreads(ctx context.Context, owner, repo string, pr int) (*CommentsResult, error)
}

// CheckFetcher fetches CI check runs for a PR.
type CheckFetcher interface {
	FetchChecks(ctx context.Context, owner, repo string, pr int) (*ChecksResult, error)
}

// ThreadResolver resolves or unresolves review threads.
type ThreadResolver interface {
	ResolveThread(ctx context.Context, threadID string) (*ResolveResult, error)
	UnresolveThread(ctx context.Context, threadID string) (*ResolveResult, error)
}

// ThreadReplier posts replies to review threads.
type ThreadReplier interface {
	ReplyToThread(ctx context.Context, owner, repo string, pr int, threadID, body string) (*ReplyResult, error)
}

// ReviewFetcher fetches PR reviews (approvals, change requests).
type ReviewFetcher interface {
	FetchReviews(ctx context.Context, owner, repo string, pr int) ([]Review, error)
}

// ReviewDismisser dismisses stale blocking pull request reviews.
type ReviewDismisser interface {
	DismissReview(ctx context.Context, owner, repo string, pr int, review Review, message string) (*DismissResult, error)
}

// ActivityProber probes lightweight review activity for settlement detection.
type ActivityProber interface {
	ProbeActivity(ctx context.Context, owner, repo string, pr int) (*ActivitySnapshot, error)
}

// Formatter formats output for pipe mode.
type Formatter interface {
	FormatComments(w io.Writer, result *CommentsResult) error
	FormatGroupedComments(w io.Writer, result *GroupedCommentsResult) error
	FormatChecks(w io.Writer, result *ChecksResult) error
	FormatReply(w io.Writer, result *ReplyResult) error
	FormatResolveResults(w io.Writer, result *ResolveResults) error
	FormatDismissResults(w io.Writer, result *DismissResults) error
	FormatStatus(w io.Writer, result *StatusResult) error
	FormatCompactStatus(w io.Writer, result *StatusResult) error
	FormatWatchStatus(w io.Writer, status *WatchStatus) error
}
