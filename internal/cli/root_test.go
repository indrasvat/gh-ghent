package cli

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRootHasSubcommands(t *testing.T) {
	cmd := NewRootCmd()

	want := []string{"checks", "comments", "reply", "resolve", "summary"}
	var got []string
	for _, sub := range cmd.Commands() {
		got = append(got, sub.Name())
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("subcommands mismatch (-want +got):\n%s", diff)
	}
}

func TestVersionFlag(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("--version failed: %v", err)
	}

	out := buf.String()
	if out == "" {
		t.Error("--version produced no output")
	}
	if !bytes.Contains([]byte(out), []byte("ghent")) {
		t.Errorf("--version output missing 'ghent': %q", out)
	}
}

func TestGlobalFlagsAccessibleFromSubcommand(t *testing.T) {
	cmd := NewRootCmd()

	tests := []struct {
		name string
		flag string
	}{
		{name: "repo", flag: "repo"},
		{name: "format", flag: "format"},
		{name: "verbose", flag: "verbose"},
		{name: "no-tui", flag: "no-tui"},
		{name: "pr", flag: "pr"},
	}

	for _, sub := range cmd.Commands() {
		for _, tt := range tests {
			t.Run(sub.Name()+"/"+tt.name, func(t *testing.T) {
				f := sub.InheritedFlags()
				if f.Lookup(tt.flag) == nil {
					t.Errorf("subcommand %q does not inherit flag %q", sub.Name(), tt.flag)
				}
			})
		}
	}
}

func TestChecksLocalFlags(t *testing.T) {
	cmd := NewRootCmd()
	checks, _, err := cmd.Find([]string{"checks"})
	if err != nil {
		t.Fatalf("finding checks subcommand: %v", err)
	}

	for _, flag := range []string{"logs", "watch"} {
		if checks.Flags().Lookup(flag) == nil {
			t.Errorf("checks missing local flag %q", flag)
		}
	}
}

func TestResolveLocalFlags(t *testing.T) {
	cmd := NewRootCmd()
	resolve, _, err := cmd.Find([]string{"resolve"})
	if err != nil {
		t.Fatalf("finding resolve subcommand: %v", err)
	}

	for _, flag := range []string{"thread", "all", "unresolve"} {
		if resolve.Flags().Lookup(flag) == nil {
			t.Errorf("resolve missing local flag %q", flag)
		}
	}
}

func TestReplyLocalFlags(t *testing.T) {
	cmd := NewRootCmd()
	reply, _, err := cmd.Find([]string{"reply"})
	if err != nil {
		t.Fatalf("finding reply subcommand: %v", err)
	}

	for _, flag := range []string{"thread", "body", "body-file"} {
		if reply.Flags().Lookup(flag) == nil {
			t.Errorf("reply missing local flag %q", flag)
		}
	}
}

func TestReplyThreadRequired(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"reply", "--body", "hello"})

	err := cmd.Execute()
	if err == nil {
		t.Error("reply without --thread should fail")
	}
}

func TestSubcommandStubsReturnNotImplemented(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "comments", args: []string{"comments"}},
		{name: "checks", args: []string{"checks"}},
		{name: "resolve", args: []string{"resolve"}},
		{name: "summary", args: []string{"summary"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCmd()
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err == nil {
				t.Errorf("%s should return not-implemented error", tt.name)
			}
		})
	}
}

func TestNoTUIOverridesIsTTY(t *testing.T) {
	// Reset global state before test
	Flags = GlobalFlags{}

	cmd := NewRootCmd()
	cmd.SetArgs([]string{"--no-tui", "comments"})

	// Execute will run PersistentPreRunE which sets Flags
	_ = cmd.Execute()

	if Flags.IsTTY {
		t.Error("IsTTY should be false when --no-tui is set")
	}
	if !Flags.NoTUI {
		t.Error("NoTUI should be true when --no-tui is set")
	}
}
