package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseFile(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantType  string
		wantBody  string
		wantHasFM bool
	}{
		{
			name: "complete frontmatter",
			input: `---
name: test memory
description: a test
type: feedback
---

body content here
`,
			wantName:  "test memory",
			wantType:  "feedback",
			wantBody:  "body content here\n",
			wantHasFM: true,
		},
		{
			name:      "no frontmatter",
			input:     "just plain content\n",
			wantName:  "",
			wantBody:  "just plain content\n",
			wantHasFM: false,
		},
		{
			name: "frontmatter with extras",
			input: `---
name: x
description: y
type: project
custom_key: custom_value
---
body
`,
			wantName:  "x",
			wantType:  "project",
			wantBody:  "body\n",
			wantHasFM: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, err := ParseFile([]byte(tt.input))
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}
			if fm.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", fm.Name, tt.wantName)
			}
			if fm.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", fm.Type, tt.wantType)
			}
			if body != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
			if fm.IsZero() == tt.wantHasFM {
				t.Errorf("IsZero = %v, want HasFM = %v", fm.IsZero(), tt.wantHasFM)
			}
		})
	}
}

func TestRenderRoundTrip(t *testing.T) {
	fm := Frontmatter{
		Name:        "round trip",
		Description: "test desc",
		Type:        "user",
	}
	body := "first paragraph\n\nsecond paragraph\n"

	out, err := Render(fm, body)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.HasPrefix(string(out), "---\n") {
		t.Errorf("rendered output should start with ---\\n, got %q", string(out)[:10])
	}

	fm2, body2, err := ParseFile(out)
	if err != nil {
		t.Fatalf("ParseFile after Render: %v", err)
	}
	if fm2.Name != fm.Name || fm2.Description != fm.Description || fm2.Type != fm.Type {
		t.Errorf("frontmatter mismatch: got %+v want %+v", fm2, fm)
	}
	if body2 != body {
		t.Errorf("body mismatch: got %q want %q", body2, body)
	}
}

func TestIndexAddRemoveUpdate(t *testing.T) {
	dir := t.TempDir()
	idx := filepath.Join(dir, "MEMORY.md")

	if err := AddEntry(idx, "foo.md", "first memory"); err != nil {
		t.Fatalf("AddEntry: %v", err)
	}
	if err := AddEntry(idx, "bar.md", "second memory"); err != nil {
		t.Fatalf("AddEntry 2: %v", err)
	}

	entries, err := ParseIndex(idx)
	if err != nil {
		t.Fatalf("ParseIndex: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].File != "foo.md" || entries[0].Description != "first memory" {
		t.Errorf("entry 0 = %+v", entries[0])
	}

	// Duplicate add is a no-op.
	if err := AddEntry(idx, "foo.md", "duplicate"); err != nil {
		t.Fatalf("AddEntry dup: %v", err)
	}
	entries, _ = ParseIndex(idx)
	if len(entries) != 2 {
		t.Errorf("after duplicate add, got %d entries, want 2", len(entries))
	}

	// Update changes description.
	if err := UpdateEntry(idx, "foo.md", "updated desc"); err != nil {
		t.Fatalf("UpdateEntry: %v", err)
	}
	entries, _ = ParseIndex(idx)
	if entries[0].Description != "updated desc" {
		t.Errorf("after update, desc = %q", entries[0].Description)
	}

	// Remove deletes the line.
	if err := RemoveEntry(idx, "foo.md"); err != nil {
		t.Fatalf("RemoveEntry: %v", err)
	}
	entries, _ = ParseIndex(idx)
	if len(entries) != 1 || entries[0].File != "bar.md" {
		t.Errorf("after remove, entries = %+v", entries)
	}
}

func TestRemoveEntryMissingFile(t *testing.T) {
	dir := t.TempDir()
	idx := filepath.Join(dir, "MEMORY.md")
	if err := RemoveEntry(idx, "nope.md"); err != nil {
		t.Errorf("RemoveEntry on missing file should be no-op, got %v", err)
	}
}

func TestLoadAndAudit(t *testing.T) {
	root := t.TempDir()

	// Build a fake project: ~/.claude/projects/-Users-test-foo/memory/
	projDir := filepath.Join(root, "projects", "-Users-test-foo", "memory")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Memory referenced in index.
	good := []byte(`---
name: good
description: in index
type: project
---
body
`)
	mustWrite(t, filepath.Join(projDir, "good.md"), good)

	// Orphan: file exists, not in index.
	orphan := []byte(`---
name: orphan
description: not in index
type: feedback
---
body
`)
	mustWrite(t, filepath.Join(projDir, "orphan.md"), orphan)

	// Stale: old mtime.
	stalePath := filepath.Join(projDir, "stale.md")
	mustWrite(t, stalePath, []byte(`---
name: stale
description: old
type: user
---
body
`))
	old := time.Now().AddDate(0, 0, -120)
	if err := os.Chtimes(stalePath, old, old); err != nil {
		t.Fatal(err)
	}

	// Index references good.md AND a missing dangling.md.
	mustWrite(t, filepath.Join(projDir, IndexFilename), []byte(`# Memory Index

- [good.md](good.md) — in index
- [stale.md](stale.md) — old
- [dangling.md](dangling.md) — does not exist
`))

	// Global CLAUDE.md so loadGlobal has something to find.
	mustWrite(t, filepath.Join(root, "CLAUDE.md"), []byte("# global\n"))

	snap, err := Load(root, 90)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(snap.Projects) < 2 {
		t.Fatalf("expected >=2 projects (global + test), got %d", len(snap.Projects))
	}

	var proj *Project
	for i := range snap.Projects {
		if snap.Projects[i].Slug == "-Users-test-foo" {
			proj = &snap.Projects[i]
			break
		}
	}
	if proj == nil {
		t.Fatal("test project not loaded")
	}

	if got, want := len(proj.Memories), 3; got != want {
		t.Errorf("memories: got %d want %d", got, want)
	}

	kinds := map[IssueKind]int{}
	for _, i := range proj.Issues {
		kinds[i.Kind]++
	}
	if kinds[OrphanFile] != 1 {
		t.Errorf("expected 1 OrphanFile, got %d (issues: %+v)", kinds[OrphanFile], proj.Issues)
	}
	if kinds[DanglingIndexLine] != 1 {
		t.Errorf("expected 1 DanglingIndexLine, got %d", kinds[DanglingIndexLine])
	}
	if kinds[Stale] != 1 {
		t.Errorf("expected 1 Stale, got %d", kinds[Stale])
	}

	// Verify InIndex flag on memories.
	good_m := proj.FindMemory("good.md")
	if good_m == nil || !good_m.InIndex {
		t.Errorf("good.md should be InIndex")
	}
	orphan_m := proj.FindMemory("orphan.md")
	if orphan_m == nil || orphan_m.InIndex {
		t.Errorf("orphan.md should NOT be InIndex")
	}
}

func TestCreateAndDeleteMemory(t *testing.T) {
	root := t.TempDir()
	projDir := filepath.Join(root, "projects", "-Users-test-bar", "memory")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	p := Project{
		Slug:      "-Users-test-bar",
		Dir:       projDir,
		IndexPath: filepath.Join(projDir, IndexFilename),
	}

	path, err := CreateMemory(p, Frontmatter{
		Name:        "My New Memory",
		Description: "scratch",
		Type:        "feedback",
	})
	if err != nil {
		t.Fatalf("CreateMemory: %v", err)
	}
	if filepath.Base(path) != "feedback_my_new_memory.md" {
		t.Errorf("file = %s, want feedback_my_new_memory.md", filepath.Base(path))
	}

	// Index has the new entry.
	entries, _ := ParseIndex(p.IndexPath)
	if len(entries) != 1 || entries[0].File != "feedback_my_new_memory.md" {
		t.Errorf("index after create = %+v", entries)
	}

	// File is parseable round-trip.
	data, _ := os.ReadFile(path)
	fm, _, err := ParseFile(data)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if fm.Name != "My New Memory" || fm.Type != "feedback" {
		t.Errorf("parsed frontmatter = %+v", fm)
	}

	// Creating again with same name appends a numeric suffix.
	path2, err := CreateMemory(p, Frontmatter{Name: "My New Memory", Type: "feedback", Description: "dup"})
	if err != nil {
		t.Fatalf("CreateMemory dup: %v", err)
	}
	if filepath.Base(path2) != "feedback_my_new_memory_2.md" {
		t.Errorf("dup file = %s", filepath.Base(path2))
	}

	// Delete removes file and index line.
	if err := DeleteMemory(p, "feedback_my_new_memory.md"); err != nil {
		t.Fatalf("DeleteMemory: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file still exists after delete")
	}
	entries, _ = ParseIndex(p.IndexPath)
	for _, e := range entries {
		if e.File == "feedback_my_new_memory.md" {
			t.Errorf("index line still present after delete")
		}
	}
}

func TestSlugifyName(t *testing.T) {
	cases := map[string]string{
		"Hello World":     "hello_world",
		"  trim_me  ":     "trim_me",
		"Caps & Symbols!": "caps_symbols",
		"":                "memory",
		"123 numbers":     "123_numbers",
		"already_slugged": "already_slugged",
	}
	for in, want := range cases {
		if got := SlugifyName(in); got != want {
			t.Errorf("SlugifyName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUnslug(t *testing.T) {
	cases := map[string]string{
		"-Users-zhengda-lu-Documents-Github":        "Documents/Github",
		"-Users-zhengda-lu-Documents-Github-myfeed": "Documents/Github/myfeed",
		"-Users-zhengda-lu":                         "Users/zhengda/lu",
		"-some-other-path":                          "some/other/path",
	}
	for in, want := range cases {
		if got := unslug(in); got != want {
			t.Errorf("unslug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFixOrphanAndDangling(t *testing.T) {
	root := t.TempDir()
	projDir := filepath.Join(root, "projects", "-Users-test-fix", "memory")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Memory file with no index entry → orphan.
	mustWrite(t, filepath.Join(projDir, "loner.md"), []byte(`---
name: loner
description: should be in index
type: project
---
body
`))

	// Index pointing at a missing file → dangling.
	mustWrite(t, filepath.Join(projDir, IndexFilename), []byte(`# Memory Index

- [ghost.md](ghost.md) — does not exist
`))

	snap, err := Load(root, 90)
	if err != nil {
		t.Fatal(err)
	}
	proj := snap.FindProject("-Users-test-fix")
	if proj == nil {
		t.Fatal("project not loaded")
	}

	if err := FixOrphan(*proj, "loner.md"); err != nil {
		t.Fatalf("FixOrphan: %v", err)
	}
	if err := FixDangling(*proj, "ghost.md"); err != nil {
		t.Fatalf("FixDangling: %v", err)
	}

	// Reload, both issues should be gone.
	snap, _ = Load(root, 90)
	proj = snap.FindProject("-Users-test-fix")
	if proj.IssueCount() != 0 {
		t.Errorf("after fix, issues: %+v", proj.Issues)
	}
	if got := len(proj.IssuesFor("loner.md")); got != 0 {
		t.Errorf("IssuesFor loner: %d", got)
	}

	// FixOrphan/FixDangling on global → error.
	global := snap.FindProject(GlobalSlug)
	if err := FixOrphan(*global, "x.md"); err == nil {
		t.Error("FixOrphan on global should fail")
	}
	if err := FixDangling(*global, "x.md"); err == nil {
		t.Error("FixDangling on global should fail")
	}

	// IssueKind String() coverage.
	for _, k := range []IssueKind{OrphanFile, DanglingIndexLine, Stale, IssueKind(99)} {
		_ = k.String()
	}
}

func TestUnindexMemory(t *testing.T) {
	root := t.TempDir()
	projDir := filepath.Join(root, "projects", "-Users-test-x", "memory")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	memPath := filepath.Join(projDir, "keep_me.md")
	mustWrite(t, memPath, []byte(`---
name: keep me
description: kept on disk
type: feedback
---
body
`))
	mustWrite(t, filepath.Join(projDir, IndexFilename), []byte(`# Memory Index

- [keep_me.md](keep_me.md) — kept on disk
`))

	p := Project{
		Slug:      "-Users-test-x",
		Dir:       projDir,
		IndexPath: filepath.Join(projDir, IndexFilename),
	}

	if err := UnindexMemory(p, "keep_me.md"); err != nil {
		t.Fatalf("UnindexMemory: %v", err)
	}
	if _, err := os.Stat(memPath); err != nil {
		t.Errorf("file should still exist after unindex: %v", err)
	}
	entries, _ := ParseIndex(p.IndexPath)
	if len(entries) != 0 {
		t.Errorf("after unindex, expected 0 index entries, got %d", len(entries))
	}

	// FixOrphan re-adds it.
	if err := FixOrphan(p, "keep_me.md"); err != nil {
		t.Fatalf("FixOrphan: %v", err)
	}
	entries, _ = ParseIndex(p.IndexPath)
	if len(entries) != 1 || entries[0].File != "keep_me.md" {
		t.Errorf("FixOrphan didn't restore index: %+v", entries)
	}

	// Global rejection.
	g := Project{IsGlobal: true}
	if err := UnindexMemory(g, "x.md"); err == nil {
		t.Error("UnindexMemory on global should fail")
	}
	if err := UnindexMemory(p, ""); err == nil {
		t.Error("UnindexMemory on empty filename should fail")
	}
	if err := UnindexMemory(p, IndexFilename); err == nil {
		t.Error("UnindexMemory on MEMORY.md should fail")
	}
}

func TestCreateMemoryRejectsGlobal(t *testing.T) {
	p := Project{IsGlobal: true}
	if _, err := CreateMemory(p, Frontmatter{Name: "x", Type: "user"}); err == nil {
		t.Error("CreateMemory on global should fail")
	}
	if err := DeleteMemory(p, "x.md"); err == nil {
		t.Error("DeleteMemory on global should fail")
	}
}

func TestCreateMemoryRequiresFields(t *testing.T) {
	p := Project{Slug: "x", Dir: t.TempDir(), IndexPath: filepath.Join(t.TempDir(), "MEMORY.md")}
	if _, err := CreateMemory(p, Frontmatter{Type: "user"}); err == nil {
		t.Error("missing name should fail")
	}
	if _, err := CreateMemory(p, Frontmatter{Name: "x"}); err == nil {
		t.Error("missing type should fail")
	}
}

func TestDeleteMemoryRefuses(t *testing.T) {
	p := Project{Slug: "x", Dir: t.TempDir()}
	if err := DeleteMemory(p, ""); err == nil {
		t.Error("empty filename should fail")
	}
	if err := DeleteMemory(p, IndexFilename); err == nil {
		t.Error("deleting MEMORY.md should fail")
	}
}

func TestLoadGlobalWithRules(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "CLAUDE.md"), []byte("# global\n"))
	mustWrite(t, filepath.Join(root, "rules", "common", "x.md"), []byte("rule x\n"))
	mustWrite(t, filepath.Join(root, "rules", "golang", "y.md"), []byte("rule y\n"))

	snap, err := Load(root, 90)
	if err != nil {
		t.Fatal(err)
	}
	g := snap.FindProject(GlobalSlug)
	if g == nil {
		t.Fatal("global not loaded")
	}
	if len(g.Memories) != 3 {
		t.Errorf("global memories = %d, want 3", len(g.Memories))
	}
}

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
