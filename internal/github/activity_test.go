package github

import (
	"testing"
	"time"

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
