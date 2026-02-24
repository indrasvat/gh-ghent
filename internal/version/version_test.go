package version

import "testing"

func TestShortCommit(t *testing.T) {
	tests := []struct {
		name   string
		commit string
		want   string
	}{
		{name: "full hash", commit: "3817a74abc123def", want: "3817a74"},
		{name: "exactly 7", commit: "3817a74", want: "3817a74"},
		{name: "short", commit: "abc", want: "abc"},
		{name: "unknown", commit: "unknown", want: "unknown"},
		{name: "empty", commit: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := Commit
			Commit = tt.commit
			t.Cleanup(func() { Commit = orig })

			if got := ShortCommit(); got != tt.want {
				t.Errorf("ShortCommit() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShortDate(t *testing.T) {
	tests := []struct {
		name      string
		buildDate string
		want      string
	}{
		{name: "ISO with time", buildDate: "2026-02-24T05:14:27Z", want: "2026-02-24"},
		{name: "date only", buildDate: "2026-02-24", want: "2026-02-24"},
		{name: "unknown", buildDate: "unknown", want: "unknown"},
		{name: "empty", buildDate: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := BuildDate
			BuildDate = tt.buildDate
			t.Cleanup(func() { BuildDate = orig })

			if got := ShortDate(); got != tt.want {
				t.Errorf("ShortDate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestString(t *testing.T) {
	orig := struct{ v, c, d string }{Version, Commit, BuildDate}
	Version = "v1.2.3"
	Commit = "abc1234def5678"
	BuildDate = "2026-01-15T12:00:00Z"
	t.Cleanup(func() {
		Version = orig.v
		Commit = orig.c
		BuildDate = orig.d
	})

	got := String()
	want := "v1.2.3 (commit: abc1234, built: 2026-01-15)"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
