package store

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Frontmatter is the YAML metadata block at the top of a memory file.
type Frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	// Extra preserves any other YAML keys we don't model so round-trip writes
	// don't drop user-added fields.
	Extra map[string]any `yaml:",inline"`
}

// ParseFile splits a memory file into (frontmatter, body). If the file has no
// `---` block at the top, Frontmatter is zero and body is the whole file.
func ParseFile(content []byte) (Frontmatter, string, error) {
	var fm Frontmatter
	if !bytes.HasPrefix(content, []byte("---")) {
		return fm, string(content), nil
	}
	rest := content[3:]
	end := bytes.Index(rest, []byte("\n---"))
	if end < 0 {
		return fm, string(content), nil
	}
	yamlBlock := rest[:end]
	body := rest[end+4:]
	body = bytes.TrimLeft(body, "\n")
	if err := yaml.Unmarshal(yamlBlock, &fm); err != nil {
		return Frontmatter{}, string(content), fmt.Errorf("parse frontmatter: %w", err)
	}
	return fm, string(body), nil
}

// Render assembles a memory file with the given frontmatter and body. If
// frontmatter is zero (no name/type/description/extras), body is returned as-is.
func Render(fm Frontmatter, body string) ([]byte, error) {
	if fm.IsZero() {
		return []byte(body), nil
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(fm); err != nil {
		return nil, fmt.Errorf("encode frontmatter: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("close encoder: %w", err)
	}
	buf.WriteString("---\n")
	buf.WriteString(body)
	return buf.Bytes(), nil
}

// IsZero reports whether the Frontmatter has no meaningful content.
func (f Frontmatter) IsZero() bool {
	return f.Name == "" && f.Description == "" && f.Type == "" && len(f.Extra) == 0
}

// ScaffoldBody returns a starter body for a new memory file.
func ScaffoldBody(fm Frontmatter) string {
	return fmt.Sprintf("# %s\n\n%s\n", fm.Name, fm.Description)
}
