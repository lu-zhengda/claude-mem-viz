package editor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// FinishedMsg is dispatched after the editor closes (with whatever error it produced).
type FinishedMsg struct {
	Path string
	Err  error
}

// Open returns a tea.Cmd that suspends the program, runs $EDITOR on path,
// and emits a FinishedMsg when the editor exits.
func Open(path string) tea.Cmd {
	editor, args := resolveEditor()
	args = append(args, path)
	cmd := exec.Command(editor, args...)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return FinishedMsg{Path: path, Err: err}
	})
}

// resolveEditor picks the first available editor from $EDITOR, $VISUAL,
// then a fallback chain. The returned slice is split on whitespace so
// `EDITOR="code -w"` works.
func resolveEditor() (string, []string) {
	for _, env := range []string{"EDITOR", "VISUAL"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			parts := strings.Fields(v)
			return parts[0], parts[1:]
		}
	}
	for _, fallback := range []string{"vim", "nvim", "nano", "vi"} {
		if _, err := exec.LookPath(fallback); err == nil {
			return fallback, nil
		}
	}
	// Last resort — let exec fail with a clear error.
	return "vi", nil
}

// EnsureWritable returns an error if path cannot be edited.
func EnsureWritable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	return nil
}
