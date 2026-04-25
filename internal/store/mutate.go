package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// nameSlug enforces the filename pattern Claude uses: lowercase alnum + underscores.
var (
	nameSlug    = regexp.MustCompile(`[^a-z0-9_]+`)
	dupUnderbar = regexp.MustCompile(`_+`)
)

// SlugifyName converts a free-form name into a safe memory filename stem.
func SlugifyName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "_")
	s = nameSlug.ReplaceAllString(s, "_")
	s = dupUnderbar.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		s = "memory"
	}
	return s
}

// CreateMemory writes a new memory file inside the project's memory dir and adds
// a line to MEMORY.md. Returns the absolute path of the new file.
func CreateMemory(p Project, fm Frontmatter) (string, error) {
	if p.IsGlobal {
		return "", errors.New("cannot create new memories in <global>")
	}
	if fm.Name == "" {
		return "", errors.New("frontmatter.Name is required")
	}
	if fm.Type == "" {
		return "", errors.New("frontmatter.Type is required")
	}

	stem := SlugifyName(fm.Name)
	if fm.Type != "" {
		stem = fmt.Sprintf("%s_%s", fm.Type, stem)
	}
	file := stem + ".md"
	path := filepath.Join(p.Dir, file)

	// Don't clobber an existing file. Append a numeric suffix instead.
	for i := 2; ; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			break
		} else if err != nil {
			return "", err
		}
		file = fmt.Sprintf("%s_%d.md", stem, i)
		path = filepath.Join(p.Dir, file)
	}

	if err := os.MkdirAll(p.Dir, 0o755); err != nil {
		return "", err
	}

	body := ScaffoldBody(fm)
	data, err := Render(fm, body)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}

	if err := AddEntry(p.IndexPath, file, fm.Description); err != nil {
		return path, fmt.Errorf("write memory ok but index update failed: %w", err)
	}
	return path, nil
}

// DeleteMemory removes a memory file and its MEMORY.md entry.
func DeleteMemory(p Project, file string) error {
	if p.IsGlobal {
		return errors.New("cannot delete <global> memories from this tool")
	}
	if file == "" || file == IndexFilename {
		return errors.New("refusing to delete index or empty filename")
	}
	path := filepath.Join(p.Dir, file)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return RemoveEntry(p.IndexPath, file)
}

// UnindexMemory removes the MEMORY.md line for a memory but leaves the file
// on disk. The memory becomes an orphan (Claude won't load it from the index)
// and can be re-added later via FixOrphan.
func UnindexMemory(p Project, file string) error {
	if p.IsGlobal {
		return errors.New("global has no MEMORY.md")
	}
	if file == "" || file == IndexFilename {
		return errors.New("refusing to unindex empty filename or the index itself")
	}
	return RemoveEntry(p.IndexPath, file)
}

// FixOrphan adds a missing MEMORY.md line for an existing file.
func FixOrphan(p Project, file string) error {
	if p.IsGlobal {
		return errors.New("global has no MEMORY.md")
	}
	mem := p.FindMemory(file)
	desc := ""
	if mem != nil {
		desc = mem.Description
	}
	return AddEntry(p.IndexPath, file, desc)
}

// FixDangling removes a MEMORY.md line for a non-existent file.
func FixDangling(p Project, file string) error {
	if p.IsGlobal {
		return errors.New("global has no MEMORY.md")
	}
	return RemoveEntry(p.IndexPath, file)
}
