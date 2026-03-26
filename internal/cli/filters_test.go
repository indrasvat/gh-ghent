package cli

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func makeThread(id, path, author string, isBot bool, numComments int) domain.ReviewThread {
	comments := make([]domain.Comment, numComments)
	for i := range comments {
		a := author
		bot := isBot
		if i > 0 {
			a = "replier"
			bot = false
		}
		comments[i] = domain.Comment{
			ID:        id + "-c" + string(rune('0'+i)),
			Author:    a,
			IsBot:     bot,
			Body:      "comment body",
			CreatedAt: time.Now(),
		}
	}
	return domain.ReviewThread{
		ID:       id,
		Path:     path,
		Line:     10,
		Comments: comments,
	}
}

func TestRecountThreads(t *testing.T) {
	t.Parallel()

	result := &domain.CommentsResult{
		Threads: []domain.ReviewThread{
			makeThread("t1", "a.go", "coderabbitai", true, 1),
			makeThread("t2", "b.go", "alice", false, 2),
			makeThread("t3", "c.go", "copilot", true, 1),
		},
	}

	recountThreads(result)

	if result.TotalCount != 3 {
		t.Errorf("TotalCount = %d, want 3", result.TotalCount)
	}
	if result.BotThreadCount != 2 {
		t.Errorf("BotThreadCount = %d, want 2", result.BotThreadCount)
	}
	if result.UnansweredCount != 2 {
		t.Errorf("UnansweredCount = %d, want 2", result.UnansweredCount)
	}
}

func TestFilterThreadsByBot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		botsOnly   bool
		humansOnly bool
		wantIDs    []string
		wantBot    int
	}{
		{
			name:    "no filter",
			wantIDs: []string{"t-bot1", "t-human1", "t-bot2", "t-human2"},
			wantBot: -1, // no-op: counters unchanged from initial (0)
		},
		{
			name:     "bots only",
			botsOnly: true,
			wantIDs:  []string{"t-bot1", "t-bot2"},
			wantBot:  2,
		},
		{
			name:       "humans only",
			humansOnly: true,
			wantIDs:    []string{"t-human1", "t-human2"},
			wantBot:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := &domain.CommentsResult{
				Threads: []domain.ReviewThread{
					makeThread("t-bot1", "a.go", "coderabbitai", true, 2),
					makeThread("t-human1", "b.go", "alice", false, 1),
					makeThread("t-bot2", "c.go", "copilot", true, 1),
					makeThread("t-human2", "d.go", "bob", false, 3),
				},
			}

			FilterThreadsByBot(result, tt.botsOnly, tt.humansOnly)

			var gotIDs []string
			for _, t := range result.Threads {
				gotIDs = append(gotIDs, t.ID)
			}
			if diff := cmp.Diff(tt.wantIDs, gotIDs); diff != "" {
				t.Errorf("thread IDs mismatch (-want +got):\n%s", diff)
			}
			if tt.wantBot >= 0 && result.BotThreadCount != tt.wantBot {
				t.Errorf("BotThreadCount = %d, want %d", result.BotThreadCount, tt.wantBot)
			}
		})
	}
}

func TestFilterThreadsByBot_NilResult(t *testing.T) {
	t.Parallel()
	// Should not panic.
	FilterThreadsByBot(nil, true, false)
}

func TestFilterThreadsByUnanswered(t *testing.T) {
	t.Parallel()

	result := &domain.CommentsResult{
		Threads: []domain.ReviewThread{
			makeThread("t1", "a.go", "bot1", true, 1),   // unanswered (1 comment)
			makeThread("t2", "b.go", "alice", false, 3), // answered (3 comments)
			makeThread("t3", "c.go", "bot2", true, 1),   // unanswered
			makeThread("t4", "d.go", "bob", false, 2),   // answered (2 comments)
		},
	}

	FilterThreadsByUnanswered(result)

	var gotIDs []string
	for _, t := range result.Threads {
		gotIDs = append(gotIDs, t.ID)
	}
	wantIDs := []string{"t1", "t3"}
	if diff := cmp.Diff(wantIDs, gotIDs); diff != "" {
		t.Errorf("thread IDs mismatch (-want +got):\n%s", diff)
	}
	if result.UnansweredCount != 2 {
		t.Errorf("UnansweredCount = %d, want 2", result.UnansweredCount)
	}
}

func TestFilterThreadsByUnanswered_NilResult(t *testing.T) {
	t.Parallel()
	// Should not panic.
	FilterThreadsByUnanswered(nil)
}

func TestFilterComposable_BotsAndUnanswered(t *testing.T) {
	t.Parallel()

	result := &domain.CommentsResult{
		Threads: []domain.ReviewThread{
			makeThread("t1", "a.go", "coderabbitai", true, 1), // bot, unanswered
			makeThread("t2", "b.go", "coderabbitai", true, 2), // bot, answered
			makeThread("t3", "c.go", "alice", false, 1),       // human, unanswered
			makeThread("t4", "d.go", "bob", false, 3),         // human, answered
		},
	}

	// Apply bots-only first, then unanswered.
	FilterThreadsByBot(result, true, false)
	FilterThreadsByUnanswered(result)

	var gotIDs []string
	for _, t := range result.Threads {
		gotIDs = append(gotIDs, t.ID)
	}
	wantIDs := []string{"t1"}
	if diff := cmp.Diff(wantIDs, gotIDs); diff != "" {
		t.Errorf("thread IDs mismatch (-want +got):\n%s", diff)
	}
}

func TestFilterEmptyResult(t *testing.T) {
	t.Parallel()

	result := &domain.CommentsResult{
		Threads: []domain.ReviewThread{
			makeThread("t1", "a.go", "alice", false, 2),
		},
	}

	// Filter for bots on a result with no bot threads.
	FilterThreadsByBot(result, true, false)

	if len(result.Threads) != 0 {
		t.Errorf("expected 0 threads, got %d", len(result.Threads))
	}
	if result.BotThreadCount != 0 {
		t.Errorf("BotThreadCount = %d, want 0", result.BotThreadCount)
	}
}
