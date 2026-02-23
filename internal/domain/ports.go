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

// Formatter formats output for pipe mode.
type Formatter interface {
	FormatComments(w io.Writer, result *CommentsResult) error
	FormatGroupedComments(w io.Writer, result *GroupedCommentsResult) error
	FormatChecks(w io.Writer, result *ChecksResult) error
	FormatReply(w io.Writer, result *ReplyResult) error
	FormatResolveResults(w io.Writer, result *ResolveResults) error
	FormatSummary(w io.Writer, result *SummaryResult) error
	FormatCompactSummary(w io.Writer, result *SummaryResult) error
	FormatWatchStatus(w io.Writer, status *WatchStatus) error
}
