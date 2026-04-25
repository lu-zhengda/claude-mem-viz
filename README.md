# claude-mem-viz

> Ever wonder what Claude Code actually remembers about you and your projects?
> When was each memory last updated? Are some of them stale? Did Claude write
> a file but forget to add it to the index, so it's quietly being ignored?

`claude-mem-viz` is a small TUI that opens up Claude Code's auto-memory at
`~/.claude/` so you can browse, edit, prune, and audit it like any other
filesystem — across every project on your machine, all in one place.

![claude-mem-viz screenshot](docs/screenshot.png)

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

Try it without touching your real memory:

```sh
./claude-mem-viz --root docs/demo-claude
```

## Usage

```sh
claude-mem-viz                     # open TUI on ~/.claude
claude-mem-viz --root /alt/path    # use a different .claude root
claude-mem-viz --stale-days 60     # flag memories older than 60 days
claude-mem-viz --list              # non-interactive dump
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

When `n` is pressed from the `<global>` view, the form prepends a
project-picker stage so you can scaffold a new memory in any project without
changing focus first.

`x` (unindex) lets you "park" a memory: the file stays on disk but Claude
won't load it through `MEMORY.md`. Press `f` on the resulting orphan entry to
put it back into the index later.

## Audit warnings

`claude-mem-viz` flags three kinds of inconsistency between the files on disk
and the `MEMORY.md` index, shown as `⚠N` next to a project and as a chip on
each affected memory:

- **orphan** — file exists in the memory dir but not referenced in `MEMORY.md`
  (Claude won't load it)
- **dangling** — `MEMORY.md` references a file that doesn't exist (broken link)
- **stale** — file mtime is older than `--stale-days` (default 90)

Press `f` on an orphan or dangling entry to auto-fix the index.

## License

MIT
