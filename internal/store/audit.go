package store

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// IssueKind enumerates audit findings.
type IssueKind int

const (
	OrphanFile IssueKind = iota
	DanglingIndexLine
	Stale
)

func (k IssueKind) String() string {
	switch k {
	case OrphanFile:
		return "orphan"
	case DanglingIndexLine:
		return "dangling"
	case Stale:
		return "stale"
	default:
		return "unknown"
	}
}

// Issue describes one audit finding for a project.
type Issue struct {
	Kind   IssueKind
	Target string // memory filename
	Detail string
}

// audit produces issues for a project based on its current memories and index file.
// staleDays<=0 disables stale checks. Global projects skip orphan/dangling checks
// since they have no MEMORY.md.
func audit(p *Project, now time.Time, staleDays int) []Issue {
	var issues []Issue

	if !p.IsGlobal && p.IndexPath != "" {
		entries, _ := ParseIndex(p.IndexPath)
		indexed := make(map[string]bool, len(entries))
		for _, e := range entries {
			indexed[e.File] = true
			full := filepath.Join(p.Dir, e.File)
			if _, err := os.Stat(full); os.IsNotExist(err) {
				issues = append(issues, Issue{
					Kind:   DanglingIndexLine,
					Target: e.File,
					Detail: fmt.Sprintf("MEMORY.md references missing file %s", e.File),
				})
			}
		}
		for _, m := range p.Memories {
			if !indexed[m.File] {
				issues = append(issues, Issue{
					Kind:   OrphanFile,
					Target: m.File,
					Detail: fmt.Sprintf("%s exists but is not in MEMORY.md", m.File),
				})
			}
		}
	}

	if staleDays > 0 {
		for _, m := range p.Memories {
			if m.Stale {
				age := int(now.Sub(m.ModTime).Hours() / 24)
				issues = append(issues, Issue{
					Kind:   Stale,
					Target: m.File,
					Detail: fmt.Sprintf("%s last modified %d days ago", m.File, age),
				})
			}
		}
	}

	return issues
}

// IssueCount returns the number of issues of any kind for this project.
func (p Project) IssueCount() int {
	return len(p.Issues)
}

// IssuesFor returns the issues affecting a specific memory file.
func (p Project) IssuesFor(file string) []Issue {
	var out []Issue
	for _, i := range p.Issues {
		if i.Target == file {
			out = append(out, i)
		}
	}
	return out
}
