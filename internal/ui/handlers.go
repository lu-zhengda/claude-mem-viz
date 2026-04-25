package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lu-zhengda/claude-mem-viz/internal/editor"
	"github.com/lu-zhengda/claude-mem-viz/internal/store"
)

func (a App) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(k, a.keys.Quit):
		return a, tea.Quit
	case key.Matches(k, a.keys.Help):
		a.helpOpen = true
		return a, nil
	case key.Matches(k, a.keys.Reload):
		a.reload()
		a.flash("reloaded from disk", false)
		return a, nil
	case key.Matches(k, a.keys.Tab):
		a.focus = (a.focus + 1) % 3
		return a, nil
	case key.Matches(k, a.keys.ShiftTab):
		a.focus = (a.focus + 2) % 3
		return a, nil
	case key.Matches(k, a.keys.Search):
		a.search.open(a.snap)
		return a, nil
	}

	switch a.focus {
	case focusProjects:
		return a.updateProjects(k)
	case focusMemories:
		return a.updateMemories(k)
	case focusContent:
		return a.updateContent(k)
	}
	return a, nil
}

func (a App) updateProjects(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(k, a.keys.Up):
		a.projects.moveUp()
		a.rebuildMemoriesList()
		a.memories.setCursor(0)
		a.refreshContent()
	case key.Matches(k, a.keys.Down):
		a.projects.moveDown()
		a.rebuildMemoriesList()
		a.memories.setCursor(0)
		a.refreshContent()
	case key.Matches(k, a.keys.Enter), key.Matches(k, a.keys.Right):
		a.focus = focusMemories
	case key.Matches(k, a.keys.New):
		a.openNewForm()
	}
	return a, nil
}

func (a App) updateMemories(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(k, a.keys.Up):
		a.memories.moveUp()
		a.refreshContent()
	case key.Matches(k, a.keys.Down):
		a.memories.moveDown()
		a.refreshContent()
	case key.Matches(k, a.keys.Left):
		a.focus = focusProjects
	case key.Matches(k, a.keys.Right):
		a.focus = focusContent
	case key.Matches(k, a.keys.Edit), key.Matches(k, a.keys.Enter):
		mem := a.currentMemory()
		if mem == nil {
			return a, nil
		}
		if err := editor.EnsureWritable(mem.Path); err != nil {
			a.flash(fmt.Sprintf("cannot edit: %v", err), true)
			return a, nil
		}
		return a, editor.Open(mem.Path)
	case key.Matches(k, a.keys.New):
		a.openNewForm()
	case key.Matches(k, a.keys.Delete):
		mem := a.currentMemory()
		proj := a.currentProject()
		if mem == nil || proj == nil {
			return a, nil
		}
		if proj.IsGlobal {
			a.flash("can't delete global memories", true)
			return a, nil
		}
		if mem.External {
			a.flash("can't delete the project's CLAUDE.md from this tool", true)
			return a, nil
		}
		a.confirm.open(
			"Delete memory?",
			fmt.Sprintf("Remove %s from %s and drop its MEMORY.md line.\nThis cannot be undone.", mem.File, proj.Label),
		)
	case key.Matches(k, a.keys.Unindex):
		mem := a.currentMemory()
		proj := a.currentProject()
		if mem == nil || proj == nil {
			return a, nil
		}
		if proj.IsGlobal {
			a.flash("global has no index to unindex from", true)
			return a, nil
		}
		if mem.External {
			a.flash("CLAUDE.md isn't in the index — nothing to unindex", false)
			return a, nil
		}
		if !mem.InIndex {
			a.flash(fmt.Sprintf("%s is already orphaned", mem.File), false)
			return a, nil
		}
		if err := store.UnindexMemory(*proj, mem.File); err != nil {
			a.flash(fmt.Sprintf("unindex failed: %v", err), true)
			return a, nil
		}
		a.flash(fmt.Sprintf("unindexed %s (file kept; press f to re-add)", mem.File), false)
		a.reload()
	case key.Matches(k, a.keys.Fix):
		a.fixCurrentIssue()
	}
	return a, nil
}

func (a App) updateContent(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(k, a.keys.Left):
		a.focus = focusMemories
	case key.Matches(k, a.keys.Edit):
		mem := a.currentMemory()
		if mem != nil {
			return a, editor.Open(mem.Path)
		}
	}
	var cmd tea.Cmd
	a.content, cmd = a.content.Update(k)
	return a, cmd
}

func (a App) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			a.search.close()
			return a, nil
		case "down":
			a.search.moveDown()
			return a, nil
		case "up":
			a.search.moveUp()
			return a, nil
		case "enter":
			if hit, ok := a.search.selected(); ok {
				a.jumpToHit(hit)
			}
			a.search.close()
			return a, nil
		}
	}
	cmd := a.search.update(msg)
	return a, cmd
}

func (a App) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		if k.String() == "esc" {
			a.form.close()
			return a, nil
		}
	}
	cmd, complete := a.form.update(msg)
	if complete {
		return a.submitNewMemory(), cmd
	}
	return a, cmd
}

func (a App) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return a, nil
	}
	switch k.String() {
	case "y", "Y", "enter":
		mem := a.currentMemory()
		proj := a.currentProject()
		a.confirm.close()
		if mem != nil && proj != nil {
			if err := store.DeleteMemory(*proj, mem.File); err != nil {
				a.flash(fmt.Sprintf("delete failed: %v", err), true)
			} else {
				a.flash(fmt.Sprintf("deleted %s", mem.File), false)
				a.reload()
			}
		}
	case "n", "N", "esc":
		a.confirm.close()
	}
	return a, nil
}

func (a App) submitNewMemory() App {
	pickedSlug, name, memType, desc := a.form.values()
	a.form.close()

	var proj *store.Project
	if pickedSlug != "" {
		proj = a.snap.FindProject(pickedSlug)
	} else {
		proj = a.currentProject()
	}
	if proj == nil || proj.IsGlobal {
		a.flash("pick a non-global project to create memories", true)
		return a
	}
	path, err := store.CreateMemory(*proj, store.Frontmatter{
		Name:        name,
		Type:        memType,
		Description: desc,
	})
	if err != nil {
		a.flash(fmt.Sprintf("create failed: %v", err), true)
		return a
	}
	a.flash(fmt.Sprintf("created %s", trimHome(path)), false)
	a.reload()
	a.selectByPath(path)
	a.refreshContent()
	return a
}

// openNewForm decides whether the form needs a project-picker stage and opens
// it. From a non-global project the picker is skipped; from <global> we collect
// every non-global project so the user can pick.
func (a *App) openNewForm() {
	proj := a.currentProject()
	if proj != nil && !proj.IsGlobal {
		a.form.open(nil, nil)
		return
	}
	var labels, slugs []string
	for _, p := range a.snap.Projects {
		if p.IsGlobal {
			continue
		}
		labels = append(labels, p.Label)
		slugs = append(slugs, p.Slug)
	}
	if len(slugs) == 0 {
		a.flash("no project memory dirs exist yet", true)
		return
	}
	a.form.open(labels, slugs)
}

func (a *App) jumpToHit(hit searchHit) {
	for i, p := range a.snap.Projects {
		if p.Slug != hit.ProjectSlug {
			continue
		}
		a.projects.setCursor(i)
		a.rebuildMemoriesList()
		for j, m := range p.Memories {
			if m.File == hit.File {
				a.memories.setCursor(j)
				break
			}
		}
		a.focus = focusContent
		a.refreshContent()
		return
	}
}

func (a *App) fixCurrentIssue() {
	proj := a.currentProject()
	mem := a.currentMemory()
	if proj == nil || mem == nil || proj.IsGlobal {
		return
	}
	if mem.External {
		a.flash("CLAUDE.md isn't tracked in MEMORY.md — nothing to fix", false)
		return
	}
	issues := proj.IssuesFor(mem.File)
	if len(issues) == 0 {
		a.flash("no issues to fix on this memory", false)
		return
	}
	for _, iss := range issues {
		switch iss.Kind {
		case store.OrphanFile:
			if err := store.FixOrphan(*proj, mem.File); err != nil {
				a.flash(fmt.Sprintf("fix failed: %v", err), true)
				return
			}
		case store.DanglingIndexLine:
			if err := store.FixDangling(*proj, mem.File); err != nil {
				a.flash(fmt.Sprintf("fix failed: %v", err), true)
				return
			}
		}
	}
	a.flash(fmt.Sprintf("fixed %d issue(s) on %s", len(issues), mem.File), false)
	a.reload()
}

// homeDir returns $HOME or empty.
func homeDir() string {
	return os.Getenv("HOME")
}
