package ui

import "github.com/charmbracelet/lipgloss"

var (
	colSubtle    = lipgloss.AdaptiveColor{Light: "#9099A2", Dark: "#5C6370"}
	colHighlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	colWarn      = lipgloss.AdaptiveColor{Light: "#D08770", Dark: "#E5C07B"}
	colDanger    = lipgloss.AdaptiveColor{Light: "#BF616A", Dark: "#E06C75"}
	colOK        = lipgloss.AdaptiveColor{Light: "#5C9C5C", Dark: "#98C379"}
	colMuted     = lipgloss.AdaptiveColor{Light: "#666", Dark: "#888"}
)

var (
	paneBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colSubtle).
			Padding(0, 1)

	paneBorderActive = paneBorder.
				BorderForeground(colHighlight)

	paneTitle = lipgloss.NewStyle().
			Foreground(colHighlight).
			Bold(true).
			MarginBottom(1)

	itemSelected = lipgloss.NewStyle().
			Foreground(colHighlight).
			Bold(true)

	itemNormal = lipgloss.NewStyle()

	chip = lipgloss.NewStyle().
		Foreground(colMuted).
		Italic(true)

	chipWarn = lipgloss.NewStyle().
			Foreground(colWarn).
			Bold(true)

	chipDanger = lipgloss.NewStyle().
			Foreground(colDanger).
			Bold(true)

	help = lipgloss.NewStyle().
		Foreground(colMuted)

	statusOK = lipgloss.NewStyle().
			Foreground(colOK)

	statusErr = lipgloss.NewStyle().
			Foreground(colDanger)

	modalBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colHighlight).
			Padding(1, 2).
			Width(60)
)
