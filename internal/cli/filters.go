package cli

import "github.com/indrasvat/gh-ghent/internal/domain"

// recountThreads recalculates all counters on a CommentsResult after in-place filtering.
func recountThreads(result *domain.CommentsResult) {
	var unresolved, resolved, bot, unanswered int
	for _, t := range result.Threads {
		if t.IsResolved {
			resolved++
		} else {
			unresolved++
		}
		if t.IsBotOriginated() {
			bot++
		}
		if t.IsUnanswered() {
			unanswered++
		}
	}
	result.TotalCount = len(result.Threads)
	result.ResolvedCount = resolved
	result.UnresolvedCount = unresolved
	result.BotThreadCount = bot
	result.UnansweredCount = unanswered
}

// FilterThreadsByBot filters threads by bot authorship in-place.
// When botsOnly is true, keeps only bot-originated threads.
// When humansOnly is true, keeps only human-originated threads.
// Caller must ensure botsOnly and humansOnly are not both true.
func FilterThreadsByBot(result *domain.CommentsResult, botsOnly, humansOnly bool) {
	if result == nil || (!botsOnly && !humansOnly) {
		return
	}

	filtered := result.Threads[:0]
	for _, t := range result.Threads {
		isBot := t.IsBotOriginated()
		if botsOnly && isBot {
			filtered = append(filtered, t)
		} else if humansOnly && !isBot {
			filtered = append(filtered, t)
		}
	}
	result.Threads = filtered
	recountThreads(result)
}

// FilterThreadsByUnanswered keeps only threads with no replies (single comment).
func FilterThreadsByUnanswered(result *domain.CommentsResult) {
	if result == nil {
		return
	}

	filtered := result.Threads[:0]
	for _, t := range result.Threads {
		if t.IsUnanswered() {
			filtered = append(filtered, t)
		}
	}
	result.Threads = filtered
	recountThreads(result)
}
