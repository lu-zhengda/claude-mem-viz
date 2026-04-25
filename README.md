# claude-mem-viz

A small TUI for browsing and editing [Claude Code](https://claude.com/claude-code) auto-memory files under `~/.claude/`.

Claude stores per-project memory as plain markdown under `~/.claude/projects/<slug>/memory/`, indexed by a `MEMORY.md` file. Editing these by hand means navigating slugified paths, remembering frontmatter shape, and keeping the index in sync. `claude-mem-viz` makes that easy:

- Three-pane layout: projects → memories → content
- View frontmatter + glamour-rendered markdown body
- Edit any memory in `$EDITOR` (suspend/resume the TUI)
- Create new memories with the right frontmatter scaffold and auto-index
- Delete memories and keep `MEMORY.md` consistent
- Fuzzy search across every memory in every project
- Audit warnings for orphan files, dangling index lines, and stale memories

## Quick start

```sh
brew tap lu-zhengda/tap
brew install claude-mem-viz
claude-mem-viz
```

Or with `go install`:

```sh
go install github.com/lu-zhengda/claude-mem-viz@latest
```

Or build from source:

```sh
git clone https://github.com/lu-zhengda/claude-mem-viz
cd claude-mem-viz
go build .
./claude-mem-viz
```

## Usage

```sh
claude-mem-viz                     # open TUI on ~/.claude
claude-mem-viz --root /alt/path    # use a different .claude root
claude-mem-viz --stale-days 60     # flag memories older than 60 days
```

## Keys

| Key | Action |
| --- | --- |
| `j`/`k`, `↑`/`↓` | move within pane |
| `tab` / `shift+tab` | switch focus across panes |
| `enter` / `e` | edit selected memory in `$EDITOR` |
| `→` / `←` | move focus right / left across panes |
| `n` | new memory (project → name → type → description) |
| `d` | delete memory (file + index entry, with confirm) |
| `x` | unindex memory — keep file on disk, remove from `MEMORY.md` |
| `f` | fix audit issue on selected memory (orphan / dangling) |
| `/` | fuzzy search across all memories |
| `r` | reload from disk |
| `?` | help (includes memory-type legend) |
| `q` / `ctrl+c` | quit |

When `n` is pressed from the `<global>` view, the form prepends a project-picker stage so you can scaffold a new memory in any project without changing focus first.

`x` (unindex) lets you "park" a memory: the file stays on disk but Claude won't load it through `MEMORY.md`. Press `f` on the resulting orphan entry to put it back into the index later.

## Audit behaviors

`claude-mem-viz` flags three kinds of inconsistency, shown as `⚠N` next to a project and as a chip on the affected memory:

- **orphan** — file exists in the memory dir but not referenced in `MEMORY.md`
- **dangling** — `MEMORY.md` references a file that doesn't exist
- **stale** — file mtime is older than `--stale-days` (default 90)

Press `f` on an orphan or dangling entry to auto-fix the index.

## Layout

```
┌─ Projects (12) ──────┬─ Memories (3) ─────────┬─ Content ───────────────┐
│ ▸ <global>           │   project_liteoauthllm │ ---                     │
│   Documents/Github   │ ▸ MEMORY.md (index)    │ name: liteoauthllm ...  │
│   .../myfeed         │                        │ type: project           │
│   .../mnemonik       │                        │ ---                     │
│   .../litemem        │                        │ liteoauthllm is a ...   │
└──────────────────────┴────────────────────────┴─────────────────────────┘
 tab focus  ↑↓ move  enter open  e edit  n new  d delete  / search  ? help  q quit
```

## License

MIT
