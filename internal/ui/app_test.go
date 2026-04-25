package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// fakeRoot builds a minimal ~/.claude layout with one project + one memory and
// returns its path.
func fakeRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	memDir := filepath.Join(root, "projects", "-Users-test-foo", "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(memDir, "feedback_test.md"), []byte(`---
name: test mem
description: hello world
type: feedback
---

body content here
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte(`# Memory Index

- [feedback_test.md](feedback_test.md) — hello world
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# global rules\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestAppRenders(t *testing.T) {
	root := fakeRoot(t)
	app, err := New(root, 90)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Send a window-size message so the app picks layout dimensions.
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(App)

	out := app.View()
	if out == "" || out == "loading…" {
		t.Fatalf("View returned empty/loading: %q", out)
	}
	if !strings.Contains(out, "claude-mem-viz") {
		t.Errorf("title missing from view: %q", out[:200])
	}
	if !strings.Contains(out, "<global>") {
		t.Errorf("<global> project missing: %q", out[:200])
	}
	if !strings.Contains(out, "test/foo") {
		t.Errorf("project label missing: %q", out)
	}
}

func TestAppFocusCycles(t *testing.T) {
	root := fakeRoot(t)
	app, _ := New(root, 90)
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(App)

	if app.focus != focusProjects {
		t.Errorf("initial focus = %v, want focusProjects", app.focus)
	}

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = model.(App)
	if app.focus != focusMemories {
		t.Errorf("after tab, focus = %v, want focusMemories", app.focus)
	}

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = model.(App)
	if app.focus != focusContent {
		t.Errorf("after tab tab, focus = %v, want focusContent", app.focus)
	}

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyTab})
	app = model.(App)
	if app.focus != focusProjects {
		t.Errorf("after tab x3 (wrap), focus = %v", app.focus)
	}
}

func TestAppNavigateAndOpenSearch(t *testing.T) {
	root := fakeRoot(t)
	app, _ := New(root, 90)
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(App)

	// '/' opens the search modal.
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	app = model.(App)
	if !app.search.visible {
		t.Fatal("search should be visible after '/'")
	}

	// Type "hello" and verify hits include feedback_test.md.
	for _, r := range "hello" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = model.(App)
	}
	if len(app.search.hits) == 0 {
		t.Fatal("expected search hits for 'hello'")
	}
	found := false
	for _, h := range app.search.hits {
		if h.File == "feedback_test.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("feedback_test.md not in hits: %+v", app.search.hits)
	}

	// Esc closes search.
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = model.(App)
	if app.search.visible {
		t.Error("search should close on esc")
	}
}

func TestAppCreateMemoryFlow(t *testing.T) {
	root := fakeRoot(t)
	app, _ := New(root, 90)
	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(App)

	// Move to second project (skip <global>).
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)

	// Press 'n' to open new form.
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	app = model.(App)
	if !app.form.visible {
		t.Fatal("new form should be visible")
	}

	// Type a name.
	for _, r := range "scratch note" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		app = model.(App)
	}
	// Submit by pressing enter 3x (advance past name, type, then submit on description).
	for i := 0; i < 3; i++ {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
		app = model.(App)
	}

	// File should now exist.
	expected := filepath.Join(root, "projects", "-Users-test-foo", "memory", "feedback_scratch_note.md")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("new memory not created at %s: %v", expected, err)
	}

	// MEMORY.md should reference it.
	idx, _ := os.ReadFile(filepath.Join(root, "projects", "-Users-test-foo", "memory", "MEMORY.md"))
	if !strings.Contains(string(idx), "feedback_scratch_note.md") {
		t.Errorf("MEMORY.md missing new entry: %s", idx)
	}
}
