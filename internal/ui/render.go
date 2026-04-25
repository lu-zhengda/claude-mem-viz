package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderList draws a simpleList into a fixed width/height. width includes the
// pane padding budget already (caller subtracts borders).
func renderList(l simpleList, width int, active bool) string {
	if l.height <= 0 {
		return ""
	}
	var rows []string
	end := l.offset + l.height
	if end > len(l.items) {
		end = len(l.items)
	}
	for i := l.offset; i < end; i++ {
		item := l.items[i]
		row := renderRow(item, i == l.cursor && active, width)
		rows = append(rows, row)
	}
	for len(rows) < l.height {
		rows = append(rows, lipgloss.NewStyle().Width(width).Render(""))
	}
	return strings.Join(rows, "\n")
}

func renderRow(item listItem, selected bool, width int) string {
	cursor := "  "
	if selected {
		cursor = "▸ "
	}
	primary := item.primary
	secondary := item.secondary

	// Reserve space for cursor + secondary (with one-space gap).
	available := width - len(cursor) - 1
	if secondary != "" {
		available -= visibleLen(secondary) + 1
	}
	if available < 0 {
		available = 0
	}

	primary = truncate(primary, available)
	primaryStyled := primary
	if selected {
		primaryStyled = itemSelected.Render(primary)
	} else if item.disabled {
		primaryStyled = lipgloss.NewStyle().Foreground(colMuted).Render(primary)
	} else {
		primaryStyled = itemNormal.Render(primary)
	}

	// Layout: cursor + primary + spacer + secondary, padded to width.
	left := cursor + primaryStyled
	if secondary == "" {
		return lipgloss.NewStyle().Width(width).Render(left)
	}

	leftLen := len(cursor) + visibleLen(primary)
	pad := width - leftLen - visibleLen(secondary)
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + secondary
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if visibleLen(s) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	return s[:w-1] + "…"
}

// visibleLen approximates display width. Doesn't account for ANSI codes (we
// only ever pass plain strings here) or wide chars beyond ASCII.
// visibleLen returns the on-screen width of s. lipgloss.Width handles ANSI
// escape sequences and wide chars correctly, so layout math doesn't drift when
// secondary chips are styled.
func visibleLen(s string) int {
	return lipgloss.Width(s)
}
