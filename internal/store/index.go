package store

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const indexHeader = "# Memory Index\n"

// indexLineRE matches `- [foo.md](foo.md) — description` (or with -- / - separator).
var indexLineRE = regexp.MustCompile(`^- \[([^\]]+)\]\(([^)]+)\)(?:\s+(?:—|--|-)\s+(.*))?\s*$`)

// IndexEntry is one parsed line of MEMORY.md.
type IndexEntry struct {
	File        string // link target, e.g. "foo.md"
	Label       string // link text, usually same as File
	Description string
}

// ParseIndex reads MEMORY.md and returns the entries it contains. Lines that
// don't match the entry pattern are ignored (the # Memory Index header etc).
func ParseIndex(path string) ([]IndexEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []IndexEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		m := indexLineRE.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		entries = append(entries, IndexEntry{
			Label:       m[1],
			File:        m[2],
			Description: m[3],
		})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// AddEntry appends an entry to MEMORY.md, creating the file with a header if
// missing. If an entry with the same File already exists, it's a no-op.
func AddEntry(indexPath, file, description string) error {
	existing, err := ParseIndex(indexPath)
	if err != nil {
		return err
	}
	for _, e := range existing {
		if e.File == file {
			return nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		return err
	}

	var buf bytes.Buffer
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		buf.WriteString(indexHeader)
		buf.WriteString("\n")
	}
	fmt.Fprintf(&buf, "- [%s](%s) — %s\n", file, file, description)

	f, err := os.OpenFile(indexPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(buf.Bytes())
	return err
}

// RemoveEntry deletes the line for the given file from MEMORY.md. No-op if the
// index doesn't exist or the entry isn't there.
func RemoveEntry(indexPath, file string) error {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var out bytes.Buffer
	changed := false
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := sc.Text()
		m := indexLineRE.FindStringSubmatch(line)
		if m != nil && m[2] == file {
			changed = true
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	if err := sc.Err(); err != nil {
		return err
	}

	if !changed {
		return nil
	}
	return os.WriteFile(indexPath, out.Bytes(), 0o644)
}

// UpdateEntry rewrites the description for an existing entry. If the entry
// isn't present, it's added.
func UpdateEntry(indexPath, file, description string) error {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return AddEntry(indexPath, file, description)
		}
		return err
	}

	var out bytes.Buffer
	found := false
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := sc.Text()
		m := indexLineRE.FindStringSubmatch(line)
		if m != nil && m[2] == file {
			fmt.Fprintf(&out, "- [%s](%s) — %s\n", file, file, description)
			found = true
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	if err := sc.Err(); err != nil {
		return err
	}

	if !found {
		// Append rather than rewrite the whole file again.
		s := strings.TrimRight(out.String(), "\n")
		out.Reset()
		out.WriteString(s)
		out.WriteByte('\n')
		fmt.Fprintf(&out, "- [%s](%s) — %s\n", file, file, description)
	}
	return os.WriteFile(indexPath, out.Bytes(), 0o644)
}
