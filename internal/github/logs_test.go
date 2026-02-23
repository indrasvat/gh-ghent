package github

import (
	"os"
	"strings"
	"testing"
)

func TestExtractErrorLines_GoTestFailures(t *testing.T) {
	log := `2025-01-15T10:30:15.0000000Z ok  	github.com/owner/repo/internal/utils	0.245s
2025-01-15T10:30:16.0000000Z ok  	github.com/owner/repo/internal/config	0.312s
2025-01-15T10:30:20.0000000Z --- FAIL: TestParseConfig (0.00s)
2025-01-15T10:30:20.1000000Z     config_test.go:42: expected "production", got "development"
2025-01-15T10:30:20.2000000Z --- FAIL: TestValidateInput (0.01s)
2025-01-15T10:30:20.3000000Z     validate_test.go:15: validation should have returned error
2025-01-15T10:30:21.0000000Z FAIL	github.com/owner/repo/internal/handler	0.523s`

	result := ExtractErrorLines(log)

	checks := []string{
		"FAIL: TestParseConfig",
		"config_test.go:42:",
		"FAIL: TestValidateInput",
		"validate_test.go:15:",
		"FAIL\tgithub.com/owner/repo/internal/handler",
	}
	for _, want := range checks {
		if !strings.Contains(result, want) {
			t.Errorf("result missing %q\nresult:\n%s", want, result)
		}
	}
}

func TestExtractErrorLines_LintErrors(t *testing.T) {
	log := `Running golangci-lint...
internal/api/handler.go:42:5: unused variable 'x' (deadcode)
internal/api/handler.go:55:1: function 'processData' is too complex (revive)
All checks passed for internal/utils
Done.`

	result := ExtractErrorLines(log)

	if !strings.Contains(result, "handler.go:42:5") {
		t.Errorf("result missing lint error location\nresult:\n%s", result)
	}
	if !strings.Contains(result, "handler.go:55:1") {
		t.Errorf("result missing second lint error\nresult:\n%s", result)
	}
}

func TestExtractErrorLines_CompileErrors(t *testing.T) {
	log := `# github.com/owner/repo/internal/server
./server.go:23:15: cannot use x (variable of type string) as int value in assignment
./server.go:45:2: undefined: processRequest
Error: process completed with exit code 2`

	result := ExtractErrorLines(log)

	if !strings.Contains(result, "server.go:23:15") {
		t.Errorf("result missing compile error\nresult:\n%s", result)
	}
	if !strings.Contains(result, "server.go:45:2") {
		t.Errorf("result missing second compile error\nresult:\n%s", result)
	}
	if !strings.Contains(result, "Error: process completed") {
		t.Errorf("result missing error prefix line\nresult:\n%s", result)
	}
}

func TestExtractErrorLines_GitHubActionsError(t *testing.T) {
	log := `Setting up Go 1.22...
Running tests...
##[error]Process completed with exit code 1.`

	result := ExtractErrorLines(log)

	if !strings.Contains(result, "##[error]Process completed") {
		t.Errorf("result missing ##[error] prefix line\nresult:\n%s", result)
	}
}

func TestExtractErrorLines_CleanLog(t *testing.T) {
	log := `Setting up Go 1.22...
Running tests...
ok  	github.com/owner/repo/internal/utils	0.245s
ok  	github.com/owner/repo/internal/config	0.312s
All tests passed.`

	result := ExtractErrorLines(log)

	if result != "" {
		t.Errorf("expected empty result for clean log, got:\n%s", result)
	}
}

func TestExtractErrorLines_ANSIStripping(t *testing.T) {
	log := "\x1b[31m--- FAIL: TestSomething (0.00s)\x1b[0m\n" +
		"\x1b[31m    test.go:10: assertion failed\x1b[0m"

	result := ExtractErrorLines(log)

	if strings.Contains(result, "\x1b") {
		t.Error("result still contains ANSI escape sequences")
	}
	if !strings.Contains(result, "FAIL: TestSomething") {
		t.Errorf("result missing test failure line\nresult:\n%s", result)
	}
}

func TestExtractErrorLines_TimestampStripping(t *testing.T) {
	log := "2025-01-15T10:30:20.0000000Z --- FAIL: TestParseConfig (0.00s)\n" +
		"2025-01-15T10:30:20.1000000Z     config_test.go:42: expected X, got Y"

	result := ExtractErrorLines(log)

	if strings.Contains(result, "2025-01-15T") {
		t.Error("result still contains timestamp prefix")
	}
	if !strings.Contains(result, "FAIL: TestParseConfig") {
		t.Errorf("result missing test failure\nresult:\n%s", result)
	}
}

func TestExtractErrorLines_Truncation(t *testing.T) {
	// Generate a log with many error lines
	var lines []string
	for i := range 200 {
		lines = append(lines, "Error: something went wrong on line "+strings.Repeat("x", 10)+" "+string(rune('0'+i%10)))
	}
	log := strings.Join(lines, "\n")

	result := ExtractErrorLines(log)

	resultLines := strings.Split(result, "\n")
	if len(resultLines) > maxExcerptLines {
		t.Errorf("result has %d lines, want at most %d", len(resultLines), maxExcerptLines)
	}
}

func TestExtractErrorLines_ContextLines(t *testing.T) {
	log := `line 1: setup
line 2: running tests
line 3: --- FAIL: TestSomething (0.00s)
line 4: cleanup
line 5: done`

	result := ExtractErrorLines(log)

	// Should include the line before and after the error
	if !strings.Contains(result, "line 2: running tests") {
		t.Errorf("result missing context line before error\nresult:\n%s", result)
	}
	if !strings.Contains(result, "line 4: cleanup") {
		t.Errorf("result missing context line after error\nresult:\n%s", result)
	}
}

func TestExtractErrorLines_GapMarkers(t *testing.T) {
	log := `line 1: start
line 2: setup complete
line 3: Error: first problem
line 4: recovered
line 5: running more tests
line 6: running more tests
line 7: running more tests
line 8: running more tests
line 9: Error: second problem
line 10: end`

	result := ExtractErrorLines(log)

	if !strings.Contains(result, "...") {
		t.Errorf("result missing gap marker between non-adjacent error regions\nresult:\n%s", result)
	}
}

func TestExtractErrorLines_PanicLine(t *testing.T) {
	log := `goroutine 1 [running]:
panic: runtime error: index out of range [5] with length 3
main.go:42 +0x1a8`

	result := ExtractErrorLines(log)

	if !strings.Contains(result, "panic: runtime error") {
		t.Errorf("result missing panic line\nresult:\n%s", result)
	}
}

func TestExtractErrorLines_FatalLine(t *testing.T) {
	log := `Starting server...
fatal error: all goroutines are asleep - deadlock!
goroutine 1 [chan receive]:`

	result := ExtractErrorLines(log)

	if !strings.Contains(result, "fatal error:") {
		t.Errorf("result missing fatal error line\nresult:\n%s", result)
	}
}

func TestExtractErrorLines_Fixture(t *testing.T) {
	data, err := os.ReadFile("../../testdata/rest/job_log.txt")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	result := ExtractErrorLines(string(data))

	checks := []string{
		"FAIL: TestParseConfig",
		"config_test.go:42:",
		"FAIL: TestValidateInput",
		"validate_test.go:15:",
		"##[error]Process completed with exit code 1.",
	}
	for _, want := range checks {
		if !strings.Contains(result, want) {
			t.Errorf("fixture result missing %q\nresult:\n%s", want, result)
		}
	}

	// Timestamps should be stripped
	if strings.Contains(result, "2025-01-15T") {
		t.Error("fixture result still contains timestamps")
	}

	// ANSI codes should not be present (fixture has none, but verify clean output)
	if strings.Contains(result, "\x1b") {
		t.Error("fixture result contains ANSI escape sequences")
	}
}

func TestCleanLine(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain line",
			in:   "hello world",
			want: "hello world",
		},
		{
			name: "timestamp prefix",
			in:   "2025-01-15T10:30:20.0000000Z --- FAIL: Test",
			want: "--- FAIL: Test",
		},
		{
			name: "ANSI codes",
			in:   "\x1b[31mError\x1b[0m: something broke",
			want: "Error: something broke",
		},
		{
			name: "both timestamp and ANSI",
			in:   "2025-01-15T10:30:20.0000000Z \x1b[31mFATAL\x1b[0m error",
			want: "FATAL error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanLine(tt.in)
			if got != tt.want {
				t.Errorf("cleanLine(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsErrorLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{name: "error keyword", line: "something error occurred", want: true},
		{name: "FAIL keyword", line: "--- FAIL: TestSomething", want: true},
		{name: "fatal keyword", line: "fatal error: deadlock", want: true},
		{name: "panic keyword", line: "panic: nil pointer", want: true},
		{name: "Error: prefix", line: "Error: process failed", want: true},
		{name: "FAIL: prefix", line: "FAIL: tests did not pass", want: true},
		{name: "##[error] prefix", line: "##[error]Process completed with exit code 1.", want: true},
		{name: "file:line pattern", line: "  handler.go:42:5: unused variable", want: true},
		{name: "clean line", line: "ok  github.com/owner/repo  0.245s", want: false},
		{name: "empty line", line: "", want: false},
		{name: "setup line", line: "Setting up Go 1.22...", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isErrorLine(tt.line)
			if got != tt.want {
				t.Errorf("isErrorLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}
