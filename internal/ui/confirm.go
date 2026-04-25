package ui

import (
	"strings"
)

type confirmModal struct {
	visible bool
	title   string
	message string
}

func (c *confirmModal) open(title, message string) {
	c.visible = true
	c.title = title
	c.message = message
}

func (c *confirmModal) close() { c.visible = false }

func (c confirmModal) view(width int) string {
	box := modalBox.Width(width)
	var b strings.Builder
	b.WriteString(chipDanger.Render(c.title))
	b.WriteString("\n\n")
	b.WriteString(c.message)
	b.WriteString("\n\n")
	b.WriteString(help.Render("y/enter confirm  n/esc cancel"))
	return box.Render(b.String())
}
