package tui

import (
	"context"
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
)

// clipboardCopyMsg is the result of an async clipboard copy operation.
type clipboardCopyMsg struct {
	text string
	err  error
}

// copyToClipboard returns a tea.Cmd that copies the given text to the system clipboard.
func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.CommandContext(context.Background(), "pbcopy")
		default:
			// Linux: try xclip first, fall back to xsel.
			if _, err := exec.LookPath("xclip"); err == nil {
				cmd = exec.CommandContext(context.Background(), "xclip", "-selection", "clipboard")
			} else {
				cmd = exec.CommandContext(context.Background(), "xsel", "--clipboard", "--input")
			}
		}
		cmd.Stdin = nil // will set via pipe
		pipe, err := cmd.StdinPipe()
		if err != nil {
			return clipboardCopyMsg{text: text, err: err}
		}
		if err := cmd.Start(); err != nil {
			return clipboardCopyMsg{text: text, err: err}
		}
		if _, err := pipe.Write([]byte(text)); err != nil {
			return clipboardCopyMsg{text: text, err: err}
		}
		pipe.Close()
		err = cmd.Wait()
		return clipboardCopyMsg{text: text, err: err}
	}
}
