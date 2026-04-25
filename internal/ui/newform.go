package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var memoryTypes = []string{"user", "feedback", "project", "reference"}

// formStage enumerates the (optional) stages of the new-memory form.
type formStage int

const (
	stageProject formStage = iota
	stageName
	stageType
	stageDescription
)

type newForm struct {
	visible   bool
	stage     formStage
	stages    []formStage // ordered list of stages this form will visit
	nameInput textinput.Model
	descInput textinput.Model
	typeIdx   int

	// Project picker state. Only populated when stages includes stageProject.
	projectLabels []string // displayed names
	projectSlugs  []string // matching snapshot.Project.Slug
	projectIdx    int
}

func newNewForm() newForm {
	name := textinput.New()
	name.Placeholder = "memory name (e.g. supabase api notes)"
	name.CharLimit = 80
	desc := textinput.New()
	desc.Placeholder = "one-line description for the index"
	desc.CharLimit = 200
	return newForm{nameInput: name, descInput: desc}
}

// open initializes the form. If projectChoices is non-empty, a project-picker
// stage is prepended; otherwise the form starts at the name stage and the
// caller's current project is used.
func (f *newForm) open(projectLabels, projectSlugs []string) {
	f.visible = true
	f.typeIdx = 1 // default to "feedback"
	f.projectLabels = projectLabels
	f.projectSlugs = projectSlugs
	f.projectIdx = 0
	f.stages = []formStage{stageName, stageType, stageDescription}
	if len(projectLabels) > 0 {
		f.stages = append([]formStage{stageProject}, f.stages...)
	}
	f.stage = f.stages[0]
	f.nameInput.Reset()
	f.descInput.Reset()
	f.refocus()
}

func (f *newForm) close() {
	f.visible = false
	f.nameInput.Blur()
	f.descInput.Blur()
}

func (f *newForm) update(msg tea.Msg) (tea.Cmd, bool) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "tab", "down":
			f.advance()
			return nil, false
		case "shift+tab", "up":
			f.retreat()
			return nil, false
		case "left":
			if f.stage == stageType {
				f.typeIdx = (f.typeIdx - 1 + len(memoryTypes)) % len(memoryTypes)
				return nil, false
			}
		case "right":
			if f.stage == stageType {
				f.typeIdx = (f.typeIdx + 1) % len(memoryTypes)
				return nil, false
			}
		case "j":
			if f.stage == stageProject {
				f.projectIdx = clamp(f.projectIdx+1, 0, len(f.projectSlugs)-1)
				return nil, false
			}
		case "k":
			if f.stage == stageProject {
				f.projectIdx = clamp(f.projectIdx-1, 0, len(f.projectSlugs)-1)
				return nil, false
			}
		case "enter":
			if f.stage != stageDescription {
				f.advance()
				return nil, false
			}
			return nil, f.complete()
		}
	}

	var cmd tea.Cmd
	switch f.stage {
	case stageName:
		f.nameInput, cmd = f.nameInput.Update(msg)
	case stageDescription:
		f.descInput, cmd = f.descInput.Update(msg)
	}
	return cmd, false
}

func (f *newForm) currentStageIdx() int {
	for i, s := range f.stages {
		if s == f.stage {
			return i
		}
	}
	return 0
}

func (f *newForm) advance() {
	idx := f.currentStageIdx()
	if idx < len(f.stages)-1 {
		f.stage = f.stages[idx+1]
	}
	f.refocus()
}

func (f *newForm) retreat() {
	idx := f.currentStageIdx()
	if idx > 0 {
		f.stage = f.stages[idx-1]
	}
	f.refocus()
}

func (f *newForm) refocus() {
	f.nameInput.Blur()
	f.descInput.Blur()
	switch f.stage {
	case stageName:
		f.nameInput.Focus()
	case stageDescription:
		f.descInput.Focus()
	}
}

func (f *newForm) complete() bool {
	return strings.TrimSpace(f.nameInput.Value()) != ""
}

func (f newForm) values() (projectSlug, name, memType, desc string) {
	if len(f.projectSlugs) > 0 && f.projectIdx >= 0 && f.projectIdx < len(f.projectSlugs) {
		projectSlug = f.projectSlugs[f.projectIdx]
	}
	return projectSlug,
		strings.TrimSpace(f.nameInput.Value()),
		memoryTypes[f.typeIdx],
		strings.TrimSpace(f.descInput.Value())
}

func (f newForm) view(width int) string {
	box := modalBox.Width(width)
	var b strings.Builder
	b.WriteString(paneTitle.Render("New memory"))
	b.WriteString("\n")

	for _, s := range f.stages {
		switch s {
		case stageProject:
			b.WriteString(fieldLabel("Project", f.stage == stageProject))
			b.WriteString("\n")
			b.WriteString(projectPicker(f.projectLabels, f.projectIdx, f.stage == stageProject))
			b.WriteString("\n\n")
		case stageName:
			b.WriteString(fieldLabel("Name", f.stage == stageName))
			b.WriteString("\n")
			b.WriteString(f.nameInput.View())
			b.WriteString("\n\n")
		case stageType:
			b.WriteString(fieldLabel("Type", f.stage == stageType))
			b.WriteString("\n")
			b.WriteString(typePicker(f.typeIdx, f.stage == stageType))
			b.WriteString("\n\n")
		case stageDescription:
			b.WriteString(fieldLabel("Description", f.stage == stageDescription))
			b.WriteString("\n")
			b.WriteString(f.descInput.View())
			b.WriteString("\n\n")
		}
	}

	hintExtras := ""
	if f.stage == stageType {
		hintExtras = "  ←/→ change type"
	} else if f.stage == stageProject {
		hintExtras = "  j/k pick project"
	}
	b.WriteString(help.Render("tab/↓ next" + hintExtras + "  enter submit  esc cancel"))
	return box.Render(b.String())
}

func fieldLabel(s string, active bool) string {
	if active {
		return itemSelected.Render("▸ " + s)
	}
	return lipgloss.NewStyle().Foreground(colMuted).Render("  " + s)
}

func typePicker(idx int, active bool) string {
	parts := make([]string, len(memoryTypes))
	for i, t := range memoryTypes {
		if i == idx {
			if active {
				parts[i] = itemSelected.Render("[" + t + "]")
			} else {
				parts[i] = lipgloss.NewStyle().Foreground(colHighlight).Render("[" + t + "]")
			}
		} else {
			parts[i] = lipgloss.NewStyle().Foreground(colMuted).Render(" " + t + " ")
		}
	}
	return strings.Join(parts, "  ")
}

func projectPicker(labels []string, idx int, active bool) string {
	if len(labels) == 0 {
		return lipgloss.NewStyle().Foreground(colMuted).Render("(no projects available)")
	}
	var lines []string
	max := len(labels)
	start := idx - 3
	if start < 0 {
		start = 0
	}
	end := start + 7
	if end > max {
		end = max
	}
	for i := start; i < end; i++ {
		marker := "  "
		text := labels[i]
		if i == idx {
			marker = "▸ "
			if active {
				text = itemSelected.Render(text)
			} else {
				text = lipgloss.NewStyle().Foreground(colHighlight).Render(text)
			}
		} else {
			text = lipgloss.NewStyle().Foreground(colMuted).Render(text)
		}
		lines = append(lines, marker+text)
	}
	return strings.Join(lines, "\n")
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
