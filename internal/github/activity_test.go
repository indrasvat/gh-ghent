package github

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/indrasvat/gh-ghent/internal/domain"
)

func TestFingerprintDeterministic(t *testing.T) {
	snap := &domain.ActivitySnapshot{
		HeadSHA:      "abc123",
		ThreadCount:  2,
		ReviewCount:  1,
		ThreadIDs:    []string{"t1", "t2"},
		ThreadStates: []bool{false, true},
		ThreadEdits:  []time.Time{time.Unix(1000, 0), time.Unix(2000, 0)},
		ReviewIDs:    []string{"r1"},
		ReviewStates: []string{"APPROVED"},
		ReviewTimes:  []time.Time{time.Unix(3000, 0)},
	}

	h1 := Fingerprint(snap)
	h2 := Fingerprint(snap)
	if h1 != h2 {
		t.Errorf("same snapshot should produce same fingerprint: %q != %q", h1, h2)
	}
}

func TestFingerprintChangesOnNewThread(t *testing.T) {
	base := &domain.ActivitySnapshot{
		HeadSHA:     "abc123",
		ThreadCount: 1,
		ThreadIDs:   []string{"t1"},
	}
	h1 := Fingerprint(base)

	modified := &domain.ActivitySnapshot{
		HeadSHA:     "abc123",
		ThreadCount: 2,
		ThreadIDs:   []string{"t1", "t2"},
	}
	h2 := Fingerprint(modified)

	if h1 == h2 {
		t.Error("new thread should change fingerprint")
	}
}

func TestFingerprintChangesOnEditedThread(t *testing.T) {
	t1 := time.Unix(1000, 0)
	t2 := time.Unix(2000, 0)

	base := &domain.ActivitySnapshot{
		HeadSHA:      "abc123",
		ThreadCount:  1,
		ThreadIDs:    []string{"t1"},
		ThreadStates: []bool{false},
		ThreadEdits:  []time.Time{t1},
	}
	h1 := Fingerprint(base)

	// Same thread ID, but updatedAt changed (bot edited its comment).
	modified := &domain.ActivitySnapshot{
		HeadSHA:      "abc123",
		ThreadCount:  1,
		ThreadIDs:    []string{"t1"},
		ThreadStates: []bool{false},
		ThreadEdits:  []time.Time{t2},
	}
	h2 := Fingerprint(modified)

	if h1 == h2 {
		t.Error("edited thread (changed updatedAt) should change fingerprint")
	}
}

func TestFingerprintChangesOnResolvedThread(t *testing.T) {
	base := &domain.ActivitySnapshot{
		HeadSHA:      "abc123",
		ThreadCount:  1,
		ThreadIDs:    []string{"t1"},
		ThreadStates: []bool{false}, // unresolved
	}
	h1 := Fingerprint(base)

	resolved := &domain.ActivitySnapshot{
		HeadSHA:      "abc123",
		ThreadCount:  1,
		ThreadIDs:    []string{"t1"},
		ThreadStates: []bool{true}, // resolved
	}
	h2 := Fingerprint(resolved)

	if h1 == h2 {
		t.Error("resolved thread should change fingerprint")
	}
}

func TestFingerprintChangesOnNewReview(t *testing.T) {
	base := &domain.ActivitySnapshot{
		HeadSHA:     "abc123",
		ReviewCount: 1,
		ReviewIDs:   []string{"r1"},
	}
	h1 := Fingerprint(base)

	modified := &domain.ActivitySnapshot{
		HeadSHA:     "abc123",
		ReviewCount: 2,
		ReviewIDs:   []string{"r1", "r2"},
	}
	h2 := Fingerprint(modified)

	if h1 == h2 {
		t.Error("new review should change fingerprint")
	}
}

func TestFingerprintChangesOnReviewStateChange(t *testing.T) {
	ts := time.Unix(1000, 0)

	base := &domain.ActivitySnapshot{
		HeadSHA:      "abc123",
		ReviewCount:  1,
		ReviewIDs:    []string{"r1"},
		ReviewStates: []string{"COMMENTED"},
		ReviewTimes:  []time.Time{ts},
	}
	h1 := Fingerprint(base)

	// Same review ID but state changed (bot upgraded review).
	modified := &domain.ActivitySnapshot{
		HeadSHA:      "abc123",
		ReviewCount:  1,
		ReviewIDs:    []string{"r1"},
		ReviewStates: []string{"CHANGES_REQUESTED"},
		ReviewTimes:  []time.Time{ts},
	}
	h2 := Fingerprint(modified)

	if h1 == h2 {
		t.Error("review state change should change fingerprint")
	}
}

func TestFingerprintChangesOnHeadSHA(t *testing.T) {
	base := &domain.ActivitySnapshot{HeadSHA: "abc123"}
	h1 := Fingerprint(base)

	modified := &domain.ActivitySnapshot{HeadSHA: "def456"}
	h2 := Fingerprint(modified)

	if h1 == h2 {
		t.Error("different head SHA should change fingerprint")
	}
}

func TestFingerprintEmptySnapshot(t *testing.T) {
	snap := &domain.ActivitySnapshot{}
	h := Fingerprint(snap)
	if h == "" {
		t.Error("fingerprint should not be empty for empty snapshot")
	}
}

func TestFingerprintChangesOnPRSignal(t *testing.T) {
	base := &domain.ActivitySnapshot{
		HeadSHA:        "abc123",
		PRReviewSignal: domain.PRReviewSignalReviewing,
	}
	h1 := Fingerprint(base)

	modified := &domain.ActivitySnapshot{
		HeadSHA:        "abc123",
		PRReviewSignal: domain.PRReviewSignalApproved,
	}
	h2 := Fingerprint(modified)

	if h1 == h2 {
		t.Error("PR review signal change should change fingerprint")
	}
}

func TestClassifyPRReviewSignal(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		editorType  string
		editorLogin string
		want        domain.PRReviewSignal
	}{
		{
			name:        "codex editor standalone eyes",
			body:        "Implementation notes\n\n👀",
			editorType:  "Bot",
			editorLogin: "chatgpt-codex-connector",
			want:        domain.PRReviewSignalReviewing,
		},
		{
			name:        "codex editor reviewing line",
			body:        "- Codex review 👀",
			editorType:  "Bot",
			editorLogin: "chatgpt-codex-connector",
			want:        domain.PRReviewSignalReviewing,
		},
		{
			name:        "codex editor standalone thumbs up",
			body:        "Ready\n\n👍",
			editorType:  "Bot",
			editorLogin: "chatgpt-codex-connector",
			want:        domain.PRReviewSignalApproved,
		},
		{
			name:        "codex editor complete line",
			body:        "Codex review complete :thumbsup:",
			editorType:  "Bot",
			editorLogin: "chatgpt-codex-connector",
			want:        domain.PRReviewSignalApproved,
		},
		{
			name:        "codex editor eyes dominate earlier thumbs up",
			body:        "Codex review complete 👍\nCodex review 👀",
			editorType:  "Bot",
			editorLogin: "chatgpt-codex-connector",
			want:        domain.PRReviewSignalReviewing,
		},
		{
			name:        "non codex editor standalone thumbs up",
			body:        "👍",
			editorType:  "Bot",
			editorLogin: "coderabbitai",
			want:        domain.PRReviewSignalNone,
		},
		{
			name:        "codex editor incidental thumbs up prose",
			body:        "This feature gives users a thumbs up affordance.",
			editorType:  "Bot",
			editorLogin: "chatgpt-codex-connector",
			want:        domain.PRReviewSignalNone,
		},
		{
			name:        "repo without codex has no signal",
			body:        "👀",
			editorType:  "User",
			editorLogin: "alice",
			want:        domain.PRReviewSignalNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyPRReviewSignal(tt.body, tt.editorType, tt.editorLogin); got != tt.want {
				t.Fatalf("classifyPRReviewSignal() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyPRReactionSignal(t *testing.T) {
	tests := []struct {
		name string
		got  func() domain.PRReviewSignal
		want domain.PRReviewSignal
	}{
		{
			name: "eyes reaction signal",
			got: func() domain.PRReviewSignal {
				return classifyPRReactionSignal(codexReactionConnection(), reactionConnection{})
			},
			want: domain.PRReviewSignalReviewing,
		},
		{
			name: "thumbs-up reaction signal",
			got: func() domain.PRReviewSignal {
				return classifyPRReactionSignal(reactionConnection{}, codexReactionConnection())
			},
			want: domain.PRReviewSignalApproved,
		},
		{
			name: "eyes dominate thumbs-up reaction",
			got: func() domain.PRReviewSignal {
				return classifyPRReactionSignal(codexReactionConnection(), codexReactionConnection())
			},
			want: domain.PRReviewSignalReviewing,
		},
		{
			name: "human eyes do not block codex thumbs-up",
			got: func() domain.PRReviewSignal {
				return classifyPRReactionSignal(humanReactionConnection(), codexReactionConnection())
			},
			want: domain.PRReviewSignalApproved,
		},
		{
			name: "eyes dominate combined signals",
			got: func() domain.PRReviewSignal {
				return combinePRReviewSignals(domain.PRReviewSignalApproved, domain.PRReviewSignalReviewing)
			},
			want: domain.PRReviewSignalReviewing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, tt.got()); diff != "" {
				t.Fatalf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCanFastSettleReview(t *testing.T) {
	tests := []struct {
		name string
		snap *domain.ActivitySnapshot
		want bool
	}{
		{
			name: "approved signal with no unresolved threads",
			snap: &domain.ActivitySnapshot{
				HeadSHA:        "abc123",
				PRReviewSignal: domain.PRReviewSignalApproved,
			},
			want: true,
		},
		{
			name: "incomplete thread probe",
			snap: &domain.ActivitySnapshot{
				HeadSHA:        "abc123",
				PRReviewSignal: domain.PRReviewSignalApproved,
				ThreadCount:    101,
				ThreadIDs:      make([]string, 100),
			},
			want: false,
		},
		{
			name: "unresolved threads",
			snap: &domain.ActivitySnapshot{
				HeadSHA:               "abc123",
				PRReviewSignal:        domain.PRReviewSignalApproved,
				UnresolvedThreadCount: 1,
			},
			want: false,
		},
		{
			name: "changes requested decision",
			snap: &domain.ActivitySnapshot{
				HeadSHA:        "abc123",
				PRReviewSignal: domain.PRReviewSignalApproved,
				ReviewDecision: string(domain.ReviewChangesRequested),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanFastSettleReview(tt.snap); got != tt.want {
				t.Fatalf("CanFastSettleReview() = %v, want %v", got, tt.want)
			}
		})
	}
}

func codexReactionConnection() reactionConnection {
	return reactionConnectionWithUser("Bot", "chatgpt-codex-connector")
}

func humanReactionConnection() reactionConnection {
	return reactionConnectionWithUser("User", "alice")
}

func reactionConnectionWithUser(typeName, login string) reactionConnection {
	return reactionConnection{
		Nodes: []struct {
			User *struct {
				TypeName string `json:"__typename"`
				Login    string `json:"login"`
			} `json:"user"`
		}{
			{
				User: &struct {
					TypeName string `json:"__typename"`
					Login    string `json:"login"`
				}{
					TypeName: typeName,
					Login:    login,
				},
			},
		},
	}
}
