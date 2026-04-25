package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up, Down      key.Binding
	Left, Right   key.Binding
	Enter         key.Binding
	Tab, ShiftTab key.Binding
	Edit          key.Binding
	New           key.Binding
	Delete        key.Binding
	Unindex       key.Binding
	Search        key.Binding
	Reload        key.Binding
	Fix           key.Binding
	Help          key.Binding
	Quit          key.Binding
	Cancel        key.Binding
	Confirm       key.Binding
}

func newKeys() keyMap {
	return keyMap{
		Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:     key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "back")),
		Right:    key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "forward")),
		Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next pane")),
		ShiftTab: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev pane")),
		Edit:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		New:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
		Delete:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Unindex:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "unindex (keep file)")),
		Search:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Reload:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
		Fix:      key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "fix")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Cancel:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		Confirm:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	}
}

func (k keyMap) shortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Tab, k.Enter, k.Edit, k.New, k.Delete, k.Search, k.Reload, k.Help, k.Quit}
}

func (k keyMap) fullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right, k.Tab, k.ShiftTab},
		{k.Enter, k.Edit, k.New, k.Delete, k.Unindex, k.Fix},
		{k.Search, k.Reload, k.Help, k.Quit},
	}
}
