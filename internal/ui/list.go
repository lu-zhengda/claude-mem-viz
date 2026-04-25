package ui

// simpleList is a stripped-down vertical list with cursor + scroll offset.
// We avoid bubbles/list so '/' search and 'd' delete don't collide with its
// built-in filter/delete bindings.
type simpleList struct {
	items  []listItem
	cursor int
	offset int
	height int
}

type listItem struct {
	primary   string
	secondary string // dim suffix (chip/badge), right-aligned
	disabled  bool
}

func (l *simpleList) setItems(items []listItem) {
	l.items = items
	if l.cursor >= len(items) {
		l.cursor = max(0, len(items)-1)
	}
	l.clampOffset()
}

func (l *simpleList) selected() (listItem, int, bool) {
	if l.cursor < 0 || l.cursor >= len(l.items) {
		return listItem{}, -1, false
	}
	return l.items[l.cursor], l.cursor, true
}

func (l *simpleList) moveUp() {
	if l.cursor > 0 {
		l.cursor--
	}
	l.clampOffset()
}

func (l *simpleList) moveDown() {
	if l.cursor < len(l.items)-1 {
		l.cursor++
	}
	l.clampOffset()
}

func (l *simpleList) setCursor(i int) {
	if i < 0 {
		i = 0
	}
	if i >= len(l.items) {
		i = len(l.items) - 1
	}
	l.cursor = i
	l.clampOffset()
}

func (l *simpleList) setHeight(h int) {
	l.height = h
	l.clampOffset()
}

func (l *simpleList) clampOffset() {
	if l.height <= 0 {
		l.offset = 0
		return
	}
	if l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+l.height {
		l.offset = l.cursor - l.height + 1
	}
	if l.offset < 0 {
		l.offset = 0
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
