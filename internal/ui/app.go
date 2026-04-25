package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/lu-zhengda/claude-mem-viz/internal/editor"
	"github.com/lu-zhengda/claude-mem-viz/internal/store"
)

type focus int

const (
	focusProjects focus = iota
	focusMemories
	focusContent
)

// App is the root Bubble Tea model.
type App struct {
	root      string
	staleDays int

	snap      store.Snapshot
	loadErr   error
	statusMsg string
	statusErr bool

	width, height int

	projects simpleList
	memories simpleList
	content  viewport.Model
	renderer *glamour.TermRenderer

	focus focus

	search   searchModal
	form     newForm
	confirm  confirmModal
	helpOpen bool

	keys keyMap
}

// New constructs the root model and performs an initial load from disk.
func New(root string, staleDays int) (App, error) {
	a := App{
		root:      root,
		staleDays: staleDays,
		focus:     focusProjects,
		keys:      newKeys(),
		search:    newSearchModal(),
		form:      newNewForm(),
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err == nil {
		a.renderer = r
	}

	a.content = viewport.New(0, 0)
	a.reload()
	return a, nil
}

func (a App) Init() tea.Cmd { return nil }

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = m.Width
		a.height = m.Height
		a.layout()
		return a, nil

	case editor.FinishedMsg:
		a.reload()
		if m.Err != nil {
			a.flash(fmt.Sprintf("editor: %v", m.Err), true)
		} else {
			a.flash(fmt.Sprintf("reloaded %s", trimHome(m.Path)), false)
		}
		// Re-select the same memory if still present.
		a.selectByPath(m.Path)
		a.refreshContent()
		return a, nil
	}

	if a.helpOpen {
		if k, ok := msg.(tea.KeyMsg); ok {
			if k.String() == "?" || k.String() == "esc" || k.String() == "q" {
				a.helpOpen = false
			}
		}
		return a, nil
	}

	if a.search.visible {
		return a.updateSearch(msg)
	}
	if a.form.visible {
		return a.updateForm(msg)
	}
	if a.confirm.visible {
		return a.updateConfirm(msg)
	}

	if k, ok := msg.(tea.KeyMsg); ok {
		return a.handleKey(k)
	}
	return a, nil
}

func (a *App) reload() {
	snap, err := store.Load(a.root, a.staleDays)
	a.snap = snap
	a.loadErr = err
	a.rebuildLists()
	a.refreshContent()
}

func (a *App) rebuildLists() {
	// Projects pane.
	pItems := make([]listItem, 0, len(a.snap.Projects))
	for _, p := range a.snap.Projects {
		secondary := ""
		if c := p.IssueCount(); c > 0 {
			secondary = chipWarn.Render(fmt.Sprintf("⚠%d", c))
		}
		pItems = append(pItems, listItem{primary: p.Label, secondary: secondary})
	}
	a.projects.setItems(pItems)

	// Memories pane (for currently selected project).
	a.rebuildMemoriesList()
}

func (a *App) rebuildMemoriesList() {
	proj := a.currentProject()
	if proj == nil {
		a.memories.setItems(nil)
		return
	}
	items := make([]listItem, 0, len(proj.Memories))
	for _, m := range proj.Memories {
		// Build chips right-to-left: type, then age, then status flags.
		var chips []string
		if m.Type != "" {
			chips = append(chips, chip.Render(m.Type))
		}
		if age := ageString(m.ModTime); age != "" {
			chips = append(chips, chip.Render(age))
		}
		if m.Stale {
			chips = append(chips, chipWarn.Render("stale"))
		}
		if !proj.IsGlobal && !m.InIndex {
			chips = append(chips, chipWarn.Render("orphan"))
		}
		secondary := strings.Join(chips, " ")

		primary := m.Name
		if primary == "" {
			primary = m.File
		}
		items = append(items, listItem{primary: primary, secondary: secondary})
	}
	a.memories.setItems(items)
}

func (a App) currentProject() *store.Project {
	if a.projects.cursor < 0 || a.projects.cursor >= len(a.snap.Projects) {
		return nil
	}
	return &a.snap.Projects[a.projects.cursor]
}

func (a App) currentMemory() *store.Memory {
	proj := a.currentProject()
	if proj == nil {
		return nil
	}
	if a.memories.cursor < 0 || a.memories.cursor >= len(proj.Memories) {
		return nil
	}
	return &proj.Memories[a.memories.cursor]
}

func (a *App) refreshContent() {
	mem := a.currentMemory()
	if mem == nil {
		a.content.SetContent(help.Render("(no memory selected)"))
		return
	}
	var body string
	if a.renderer != nil {
		out, err := a.renderer.Render(mem.Body)
		if err != nil {
			body = mem.Body
		} else {
			body = out
		}
	} else {
		body = mem.Body
	}

	header := ""
	if mem.HasFM {
		var fm []string
		fm = append(fm, "---")
		if mem.Name != "" {
			fm = append(fm, "name: "+mem.Name)
		}
		if mem.Description != "" {
			fm = append(fm, "description: "+mem.Description)
		}
		if mem.Type != "" {
			fm = append(fm, "type: "+mem.Type)
		}
		fm = append(fm, "---", "")
		header = lipgloss.NewStyle().Foreground(colMuted).Render(strings.Join(fm, "\n")) + "\n"
	}
	a.content.SetContent(header + body)
	a.content.GotoTop()
}

func (a *App) selectByPath(path string) {
	for i, p := range a.snap.Projects {
		for j, m := range p.Memories {
			if m.Path == path {
				a.projects.setCursor(i)
				a.rebuildMemoriesList()
				a.memories.setCursor(j)
				return
			}
		}
	}
}

func (a *App) flash(msg string, isErr bool) {
	a.statusMsg = msg
	a.statusErr = isErr
}

func trimHome(p string) string {
	if home := homeDir(); home != "" && strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}
