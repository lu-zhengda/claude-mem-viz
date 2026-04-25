package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// layout recomputes pane dimensions whenever the window size changes.
func (a *App) layout() {
	if a.width <= 0 || a.height <= 0 {
		return
	}
	titleH := 1
	statusH := 1
	helpH := 1
	bodyH := a.height - titleH - statusH - helpH
	if bodyH < 6 {
		bodyH = 6
	}

	leftW := a.width / 4
	if leftW < 18 {
		leftW = 18
	}
	midW := a.width / 4
	if midW < 22 {
		midW = 22
	}
	rightW := a.width - leftW - midW
	if rightW < 30 {
		rightW = 30
	}

	// Bordered panes consume 2 chars (left+right border) and 2 (top+bottom)
	// plus 2 for horizontal padding (1 each side).
	innerLeftW := leftW - 4
	innerMidW := midW - 4
	innerRightW := rightW - 4
	innerH := bodyH - 4

	if innerLeftW < 4 {
		innerLeftW = 4
	}
	if innerMidW < 4 {
		innerMidW = 4
	}
	if innerRightW < 4 {
		innerRightW = 4
	}
	if innerH < 1 {
		innerH = 1
	}

	a.projects.setHeight(innerH - 2) // -2 leaves room for title line + spacer
	a.memories.setHeight(innerH - 2)
	a.content.Width = innerRightW
	a.content.Height = innerH - 2

	a.refreshContent()
}

// View renders the entire UI.
func (a App) View() string {
	if a.width == 0 || a.height == 0 {
		return "loading…"
	}
	if a.helpOpen {
		return a.renderHelpModal()
	}

	title := a.renderTitle()
	body := a.renderBody()
	statusLine := a.renderStatus()
	helpLine := a.renderHelpBar()

	main := lipgloss.JoinVertical(lipgloss.Left, title, body, statusLine, helpLine)

	if a.search.visible {
		return overlay(main, a.search.view(min(80, a.width-4)))
	}
	if a.form.visible {
		return overlay(main, a.form.view(min(60, a.width-4)))
	}
	if a.confirm.visible {
		return overlay(main, a.confirm.view(min(60, a.width-4)))
	}
	return main
}

func (a App) renderTitle() string {
	count := 0
	for _, p := range a.snap.Projects {
		count += len(p.Memories)
	}
	t := fmt.Sprintf(" claude-mem-viz  %d memories across %d projects", count, len(a.snap.Projects))
	if a.loadErr != nil {
		t += "  " + statusErr.Render("(load error)")
	}
	return lipgloss.NewStyle().
		Width(a.width).
		Foreground(colHighlight).
		Bold(true).
		Render(t)
}

func (a App) renderBody() string {
	leftW := a.width / 4
	if leftW < 18 {
		leftW = 18
	}
	midW := a.width / 4
	if midW < 22 {
		midW = 22
	}
	rightW := a.width - leftW - midW
	if rightW < 30 {
		rightW = 30
	}
	bodyH := a.height - 3
	if bodyH < 8 {
		bodyH = 8
	}

	left := a.renderProjectsPane(leftW, bodyH)
	mid := a.renderMemoriesPane(midW, bodyH)
	right := a.renderContentPane(rightW, bodyH)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, mid, right)
}

func (a App) renderProjectsPane(w, h int) string {
	innerW := w - 4
	innerH := h - 4
	if innerH < 1 {
		innerH = 1
	}
	title := paneTitle.Render(fmt.Sprintf("Projects (%d)", len(a.snap.Projects)))
	list := renderList(a.projects, innerW, a.focus == focusProjects)
	body := lipgloss.JoinVertical(lipgloss.Left, title, list)
	style := paneBorder
	if a.focus == focusProjects {
		style = paneBorderActive
	}
	return style.Width(w - 2).Height(h - 2).Render(body)
}

func (a App) renderMemoriesPane(w, h int) string {
	innerW := w - 4
	innerH := h - 4
	if innerH < 1 {
		innerH = 1
	}
	proj := a.currentProject()
	header := "Memories"
	if proj != nil {
		header = fmt.Sprintf("%s (%d)", proj.Label, len(proj.Memories))
	}
	title := paneTitle.Render(header)
	list := renderList(a.memories, innerW, a.focus == focusMemories)
	body := lipgloss.JoinVertical(lipgloss.Left, title, list)
	style := paneBorder
	if a.focus == focusMemories {
		style = paneBorderActive
	}
	return style.Width(w - 2).Height(h - 2).Render(body)
}

func (a App) renderContentPane(w, h int) string {
	mem := a.currentMemory()
	header := "Content"
	if mem != nil {
		header = mem.File
	}
	title := paneTitle.Render(header)
	body := a.content.View()
	style := paneBorder
	if a.focus == focusContent {
		style = paneBorderActive
	}
	return style.Width(w - 2).Height(h - 2).Render(lipgloss.JoinVertical(lipgloss.Left, title, body))
}

func (a App) renderStatus() string {
	if a.statusMsg == "" {
		return lipgloss.NewStyle().Width(a.width).Render(" ")
	}
	style := statusOK
	prefix := "✓ "
	if a.statusErr {
		style = statusErr
		prefix = "✗ "
	}
	return lipgloss.NewStyle().Width(a.width).Render(" " + style.Render(prefix+a.statusMsg))
}

func (a App) renderHelpBar() string {
	keys := "tab focus  ↑↓ move  enter/e edit  n new  d delete  x unindex  f fix  / search  r reload  ? help  q quit"
	return lipgloss.NewStyle().Width(a.width).Foreground(colMuted).Render(" " + keys)
}

func (a App) renderHelpModal() string {
	var b strings.Builder
	b.WriteString(paneTitle.Render("claude-mem-viz — keys"))
	b.WriteString("\n\n")
	for _, group := range a.keys.fullHelp() {
		var parts []string
		for _, kb := range group {
			parts = append(parts, fmt.Sprintf("%-12s %s", kb.Help().Key, kb.Help().Desc))
		}
		b.WriteString(strings.Join(parts, "\n"))
		b.WriteString("\n\n")
	}
	b.WriteString(paneTitle.Render("memory types"))
	b.WriteString("\n")
	for _, line := range []string{
		"user        facts about the user — role, preferences, knowledge",
		"feedback    corrections / preferences for how Claude should work",
		"project     project context not derivable from code (deadlines, decisions)",
		"reference   pointers to external systems (Linear, dashboards, docs)",
	} {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("\n")
	b.WriteString(help.Render("press ? or esc to close"))
	return modalBox.Width(min(78, a.width-4)).Render(b.String())
}

// overlay renders modal centered on top of base. We do this by stacking the
// modal vertically with the base so we don't need lipgloss.Place's z-ordering.
func overlay(base, modal string) string {
	return lipgloss.JoinVertical(lipgloss.Left, base, modal)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
