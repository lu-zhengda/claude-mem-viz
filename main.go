package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lu-zhengda/claude-mem-viz/internal/store"
	"github.com/lu-zhengda/claude-mem-viz/internal/ui"
)

func main() {
	defaultRoot := filepath.Join(os.Getenv("HOME"), ".claude")
	root := flag.String("root", defaultRoot, "Path to .claude root directory")
	staleDays := flag.Int("stale-days", 90, "Memories older than this many days are flagged stale")
	listOnly := flag.Bool("list", false, "List projects + memories and exit (non-interactive)")
	flag.Parse()

	if _, err := os.Stat(*root); err != nil {
		fmt.Fprintf(os.Stderr, "claude-mem-viz: cannot access %s: %v\n", *root, err)
		os.Exit(1)
	}

	if *listOnly {
		runList(*root, *staleDays)
		return
	}

	model, err := ui.New(*root, *staleDays)
	if err != nil {
		fmt.Fprintf(os.Stderr, "claude-mem-viz: %v\n", err)
		os.Exit(1)
	}

	prog := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "claude-mem-viz: %v\n", err)
		os.Exit(1)
	}
}

func runList(root string, staleDays int) {
	snap, err := store.Load(root, staleDays)
	if err != nil {
		fmt.Fprintf(os.Stderr, "claude-mem-viz: load: %v\n", err)
		os.Exit(1)
	}
	for _, p := range snap.Projects {
		marker := ""
		if c := p.IssueCount(); c > 0 {
			marker = fmt.Sprintf(" [warn:%d]", c)
		}
		fmt.Printf("\n## %s (%d memories)%s\n", p.Label, len(p.Memories), marker)
		for _, m := range p.Memories {
			tag := m.Type
			if tag == "" {
				tag = "-"
			}
			flags := ""
			if !p.IsGlobal && !m.InIndex {
				flags += " orphan"
			}
			if m.Stale {
				flags += " stale"
			}
			fmt.Printf("  - [%s] %s%s — %s\n", tag, m.File, flags, m.Description)
		}
		for _, iss := range p.Issues {
			if iss.Kind == store.DanglingIndexLine {
				fmt.Printf("  ! dangling: %s\n", iss.Detail)
			}
		}
	}
}
