package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"

	"github.com/lu-zhengda/claude-mem-viz/internal/store"
)

type searchHit struct {
	ProjectSlug string
	File        string
	ProjectLbl  string
	Snippet     string
	Score       int
}

type searchModal struct {
	input   textinput.Model
	hits    []searchHit
	cursor  int
	corpus  []searchableEntry
	visible bool
}

type searchableEntry struct {
	ProjectSlug string
	File        string
	ProjectLbl  string
	Haystack    string // lowercase concat of name+description+body for fuzzy
}

func newSearchModal() searchModal {
	ti := textinput.New()
	ti.Placeholder = "type to search across all memories…"
	ti.Prompt = "/ "
	ti.CharLimit = 80
	return searchModal{input: ti}
}

func (s *searchModal) open(snap store.Snapshot) {
	s.input.Reset()
	s.input.Focus()
	s.cursor = 0
	s.hits = nil
	s.corpus = buildCorpus(snap)
	s.visible = true
}

func (s *searchModal) close() {
	s.input.Blur()
	s.visible = false
}

func buildCorpus(snap store.Snapshot) []searchableEntry {
	var out []searchableEntry
	for _, p := range snap.Projects {
		for _, m := range p.Memories {
			h := strings.ToLower(m.Name + " " + m.Description + " " + m.Body)
			out = append(out, searchableEntry{
				ProjectSlug: p.Slug,
				File:        m.File,
				ProjectLbl:  p.Label,
				Haystack:    h,
			})
		}
	}
	return out
}

func (s *searchModal) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	s.recompute()
	return cmd
}

func (s *searchModal) recompute() {
	q := strings.ToLower(strings.TrimSpace(s.input.Value()))
	if q == "" {
		s.hits = nil
		s.cursor = 0
		return
	}
	haystacks := make([]string, len(s.corpus))
	for i, e := range s.corpus {
		haystacks[i] = e.Haystack
	}
	matches := fuzzy.Find(q, haystacks)
	hits := make([]searchHit, 0, len(matches))
	for _, m := range matches {
		entry := s.corpus[m.Index]
		snippet := snippetAround(entry.Haystack, q, 60)
		hits = append(hits, searchHit{
			ProjectSlug: entry.ProjectSlug,
			File:        entry.File,
			ProjectLbl:  entry.ProjectLbl,
			Snippet:     snippet,
			Score:       m.Score,
		})
	}
	if len(hits) > 50 {
		hits = hits[:50]
	}
	s.hits = hits
	if s.cursor >= len(hits) {
		s.cursor = max(0, len(hits)-1)
	}
}

func snippetAround(s, needle string, width int) string {
	idx := strings.Index(s, needle)
	if idx < 0 {
		if len(s) <= width {
			return s
		}
		return s[:width] + "…"
	}
	start := idx - width/2
	if start < 0 {
		start = 0
	}
	end := start + width
	if end > len(s) {
		end = len(s)
	}
	prefix := ""
	suffix := ""
	if start > 0 {
		prefix = "…"
	}
	if end < len(s) {
		suffix = "…"
	}
	snip := s[start:end]
	snip = strings.ReplaceAll(snip, "\n", " ")
	return prefix + snip + suffix
}

func (s *searchModal) moveDown() {
	if s.cursor < len(s.hits)-1 {
		s.cursor++
	}
}

func (s *searchModal) moveUp() {
	if s.cursor > 0 {
		s.cursor--
	}
}

func (s searchModal) selected() (searchHit, bool) {
	if s.cursor < 0 || s.cursor >= len(s.hits) {
		return searchHit{}, false
	}
	return s.hits[s.cursor], true
}

func (s searchModal) view(width int) string {
	box := modalBox.Width(width)
	var b strings.Builder
	b.WriteString(paneTitle.Render("Search"))
	b.WriteString("\n")
	b.WriteString(s.input.View())
	b.WriteString("\n\n")
	if len(s.hits) == 0 {
		if s.input.Value() == "" {
			b.WriteString(help.Render("type a query to search across every memory"))
		} else {
			b.WriteString(help.Render("no matches"))
		}
	} else {
		for i, h := range s.hits {
			marker := "  "
			line := fmt.Sprintf("%s%s — %s", marker, h.ProjectLbl, h.File)
			snipLine := "    " + lipgloss.NewStyle().Foreground(colMuted).Render(h.Snippet)
			if i == s.cursor {
				marker = "▸ "
				line = itemSelected.Render(fmt.Sprintf("%s%s — %s", marker, h.ProjectLbl, h.File))
			}
			b.WriteString(line)
			b.WriteString("\n")
			b.WriteString(snipLine)
			b.WriteString("\n")
			if i >= 8 {
				more := len(s.hits) - i - 1
				if more > 0 {
					b.WriteString(help.Render(fmt.Sprintf("…and %d more", more)))
					b.WriteString("\n")
				}
				break
			}
		}
	}
	b.WriteString("\n")
	b.WriteString(help.Render("↑/↓ move  enter open  esc cancel"))
	return box.Render(b.String())
}
