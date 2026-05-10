package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// GlobalSlug is the synthetic slug used for ~/.claude (CLAUDE.md + rules/).
const GlobalSlug = "<global>"

// IndexFilename is the per-project memory index file.
const IndexFilename = "MEMORY.md"

// Snapshot is an immutable view of all memory state on disk at one point in time.
type Snapshot struct {
	Root      string
	StaleDays int
	Projects  []Project
}

// Project groups the memory files under one ~/.claude/projects/<slug>/memory/
// directory, plus the synthetic <global> project.
type Project struct {
	Slug      string
	Label     string
	Dir       string
	IndexPath string // "" for <global> (no MEMORY.md)
	IsGlobal  bool
	Memories  []Memory
	Issues    []Issue
}

// Memory is one parsed memory file.
type Memory struct {
	Path        string
	File        string // basename (or rel path for global rules)
	Name        string
	Description string
	Type        string
	Body        string
	ModTime     time.Time
	InIndex     bool
	HasFM       bool
	Stale       bool
	// External is true when the file lives outside the project's memory dir.
	// Currently used for the project's own CLAUDE.md (lives in the project
	// root). External memories cannot be deleted, unindexed, or audit-fixed
	// through this tool — only viewed and edited.
	External bool
}

// Load walks ~/.claude and returns a fresh snapshot. Errors reading individual
// files are tolerated — affected memories are still listed but marked HasFM=false.
func Load(root string, staleDays int) (Snapshot, error) {
	snap := Snapshot{Root: root, StaleDays: staleDays}
	home := os.Getenv("HOME")

	global, err := loadGlobal(root)
	if err != nil {
		return snap, fmt.Errorf("load global: %w", err)
	}
	snap.Projects = append(snap.Projects, global)

	projectsDir := filepath.Join(root, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil && !os.IsNotExist(err) {
		return snap, fmt.Errorf("read projects dir: %w", err)
	}

	var projs []Project
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		memDir := filepath.Join(projectsDir, e.Name(), "memory")
		if _, err := os.Stat(memDir); err != nil {
			continue
		}
		p, err := loadProject(e.Name(), memDir, home)
		if err != nil {
			return snap, fmt.Errorf("load project %s: %w", e.Name(), err)
		}
		projs = append(projs, p)
	}

	sort.Slice(projs, func(i, j int) bool {
		return projs[i].Label < projs[j].Label
	})
	snap.Projects = append(snap.Projects, projs...)

	now := time.Now()
	for i := range snap.Projects {
		markStale(&snap.Projects[i], now, staleDays)
		snap.Projects[i].Issues = audit(&snap.Projects[i], now, staleDays)
	}

	return snap, nil
}

func loadGlobal(root string) (Project, error) {
	p := Project{
		Slug:     GlobalSlug,
		Label:    "<global>",
		Dir:      root,
		IsGlobal: true,
	}

	claudeMD := filepath.Join(root, "CLAUDE.md")
	if mem, err := readMemory(claudeMD); err == nil {
		mem.Name = "CLAUDE.md"
		mem.Description = "Global instructions"
		p.Memories = append(p.Memories, mem)
	}

	rulesDir := filepath.Join(root, "rules")
	_ = filepath.WalkDir(rulesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		mem, err := readMemory(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		mem.File = rel
		if mem.Name == "" {
			mem.Name = rel
		}
		if mem.Description == "" {
			mem.Description = "Global rule"
		}
		p.Memories = append(p.Memories, mem)
		return nil
	})

	sort.Slice(p.Memories, func(i, j int) bool {
		return p.Memories[i].Path < p.Memories[j].Path
	})
	return p, nil
}

func loadProject(slug, memDir, home string) (Project, error) {
	p := Project{
		Slug:      slug,
		Label:     unslug(slug, home),
		Dir:       memDir,
		IndexPath: filepath.Join(memDir, IndexFilename),
	}

	indexEntries, _ := ParseIndex(p.IndexPath)
	indexed := make(map[string]bool, len(indexEntries))
	for _, e := range indexEntries {
		indexed[e.File] = true
	}

	entries, err := os.ReadDir(memDir)
	if err != nil {
		return p, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if e.Name() == IndexFilename {
			continue
		}
		path := filepath.Join(memDir, e.Name())
		mem, err := readMemory(path)
		if err != nil {
			continue
		}
		mem.InIndex = indexed[e.Name()]
		p.Memories = append(p.Memories, mem)
	}

	sort.Slice(p.Memories, func(i, j int) bool {
		return p.Memories[i].File < p.Memories[j].File
	})

	// Project's own CLAUDE.md (lives in the project root, not the memory dir).
	// Prepended after sorting so it always shows first.
	if claudeMD := findProjectClaudeMD(slug, home); claudeMD != "" {
		if mem, err := readMemory(claudeMD); err == nil {
			mem.External = true
			if mem.Name == "" {
				mem.Name = "CLAUDE.md"
			}
			if mem.Description == "" {
				mem.Description = "Project instructions"
			}
			p.Memories = append([]Memory{mem}, p.Memories...)
		}
	}

	return p, nil
}

// findProjectClaudeMD returns the path to <project>/CLAUDE.md if it exists,
// inferring the project path from the slug via slugToPath.
func findProjectClaudeMD(slug, home string) string {
	projPath := slugToPath(slug, home)
	if projPath == "" {
		return ""
	}
	candidate := filepath.Join(projPath, "CLAUDE.md")
	if _, err := os.Stat(candidate); err != nil {
		return ""
	}
	return candidate
}

// slugToPath inverts the slugging Claude Code applies to absolute project
// paths. Slugging maps both "/" and "." to "-", so the inverse is ambiguous
// from the slug alone (e.g. "claude-mem-viz" vs "claude/mem/viz"). We resolve
// the ambiguity by walking the real filesystem under $HOME: at each level we
// try every directory entry whose slugified name is a prefix of the remaining
// slug, longest first, backtracking on dead ends. Returns "" if the slug
// doesn't sit under $HOME or no path on disk reproduces it.
func slugToPath(slug, home string) string {
	if home == "" {
		return ""
	}
	slugHome := slugifyPath(home)
	if !strings.HasPrefix(slug, slugHome) {
		return ""
	}
	rest := slug[len(slugHome):]
	if rest == "" {
		return home
	}
	if !strings.HasPrefix(rest, "-") {
		return ""
	}
	return resolveSlugRemainder(home, strings.TrimPrefix(rest, "-"))
}

// resolveSlugRemainder recursively matches `remaining` against subdirectories
// of `current`, trying longest slug match first and backtracking on failure.
func resolveSlugRemainder(current, remaining string) string {
	if remaining == "" {
		return current
	}
	entries, err := os.ReadDir(current)
	if err != nil {
		return ""
	}
	type cand struct {
		name, slug string
	}
	var cands []cand
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		s := slugifyPath(e.Name())
		if remaining == s || strings.HasPrefix(remaining, s+"-") {
			cands = append(cands, cand{e.Name(), s})
		}
	}
	sort.Slice(cands, func(i, j int) bool {
		return len(cands[i].slug) > len(cands[j].slug)
	})
	for _, c := range cands {
		next := strings.TrimPrefix(strings.TrimPrefix(remaining, c.slug), "-")
		if got := resolveSlugRemainder(filepath.Join(current, c.name), next); got != "" {
			return got
		}
	}
	return ""
}

func slugifyPath(p string) string {
	s := strings.ReplaceAll(p, "/", "-")
	s = strings.ReplaceAll(s, ".", "-")
	return s
}

func readMemory(path string) (Memory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Memory{}, err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return Memory{}, err
	}
	fm, body, _ := ParseFile(data)
	return Memory{
		Path:        path,
		File:        filepath.Base(path),
		Name:        fm.Name,
		Description: fm.Description,
		Type:        fm.Type,
		Body:        body,
		ModTime:     stat.ModTime(),
		HasFM:       !fm.IsZero(),
	}, nil
}

func markStale(p *Project, now time.Time, staleDays int) {
	if staleDays <= 0 {
		return
	}
	cutoff := now.AddDate(0, 0, -staleDays)
	for i := range p.Memories {
		if p.Memories[i].ModTime.Before(cutoff) {
			p.Memories[i].Stale = true
		}
	}
}

// unslug returns a short display label for a slug, e.g.
// "-Users-zhengda-lu-Documents-Github-myfeed" → "Documents/Github/myfeed".
// When the slug resolves to a real directory under home we use the actual
// filesystem path so dashed/dotted directory names render correctly. The
// home-less fallback treats every "-" as "/" — lossy, but the best we can
// do without disk access.
func unslug(slug, home string) string {
	if home != "" {
		if real := slugToPath(slug, home); real != "" {
			if rel := strings.TrimPrefix(real, home); rel != real {
				return strings.TrimPrefix(rel, "/")
			}
		}
	}
	s := strings.TrimPrefix(slug, "-")
	s = strings.ReplaceAll(s, "-", "/")
	parts := strings.SplitN(s, "/", 4)
	if len(parts) == 4 && parts[0] == "Users" {
		return parts[3]
	}
	return s
}

// FindProject returns the project matching slug, or nil.
func (s Snapshot) FindProject(slug string) *Project {
	for i := range s.Projects {
		if s.Projects[i].Slug == slug {
			return &s.Projects[i]
		}
	}
	return nil
}

// FindMemory returns the memory at the given file path, or nil.
func (p Project) FindMemory(file string) *Memory {
	for i := range p.Memories {
		if p.Memories[i].File == file {
			return &p.Memories[i]
		}
	}
	return nil
}
