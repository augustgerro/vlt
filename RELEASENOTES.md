# Release Notes

## v0.3.1

### New Features
- **Edit entries with `Ctrl-E`** — opens selected entry in `$EDITOR` (Unix-way, like `git commit` or `kubectl edit`)
  - Temp file with editable `Category`, `Description`, `Command` fields
  - Vault updated in-place; no write if nothing changed
  - Works in both full-list and filtered views
- **Smart editor detection** — `$EDITOR` → `$VISUAL` → `nvim` → `vim` → `nano` → `vi`

## v0.3.0

### New Features
- **Hierarchical drill-down navigation** — browse vault like a file manager:
  `All entries → Categories → Filtered entries` with `→`/`←` arrow keys
- **Category picker** with entry counts per category (`Kubernetes (3)`)
- **Aligned table columns** — fixed-width fields with `│` separators for readability
- **Preview panel** — shows Category, Description, and Command for the highlighted entry
- **Delete entries** — `Ctrl-D` removes selected entry from vault
- **Contextual UI** — border label shows breadcrumb (`🔐 Vault › Kubernetes`),
  header hints change per navigation level
- **Cursor memory** — returning from a category keeps the cursor on that category

### Architecture
- Replaced fragile fzf `--bind` shell actions with `--expect` + Go event loop
  (state machine pattern) — eliminates all shell escaping issues
- fzf is used purely as a rendering/selection engine; all logic stays in Go

### Bug Fixes
- **Fixed commands with `|` being truncated** — `parseVault` now rejoins pipe-containing commands
- **Fixed delete not matching pipe commands** — tries both escaped and unescaped variants

## v0.2.0

### Breaking Changes
- **AI categorization is now opt-in.** Use `--ai` or `-a` flag to enable it.
  Previously Gemini was called automatically on every `vlt "command"` invocation.
  ```bash
  # Before (v0.1.0)
  vlt "docker ps -a"           # auto-called Gemini

  # After (v0.2.0)
  vlt "docker ps -a"           # manual input (default)
  vlt --ai "docker ps -a"     # uses Gemini AI
  ```

### New Features
- `--help` / `-h` flag showing usage, vault file path, and detected dependencies
- `--ai` / `-a` flag for explicit AI categorization
- Cross-platform clipboard: auto-detects `pbcopy` (macOS), `xclip`, `xsel`, `wl-copy` (Linux)
- fzf exact matching (`--exact`) for reliable full-word search

### Bug Fixes
- **Fixed shell injection vulnerability** in `showVault()` — replaced shell string interpolation with Go-native file I/O
- **Fixed "Copied!" always printing** — now only prints on successful clipboard copy; ESC exits silently
- **Fixed pipe characters breaking vault** — `|` in commands is now escaped as `\|` on store and unescaped on read
- **Fixed Gemini output parsing** — validates AI response format and length, falls back to manual on bad output
- **Fixed fzf not checked** — clear error message if fzf is not installed
- **Fixed all silently ignored errors** — `UserHomeDir`, config loading, JSON decode, file operations

### Maintenance
- Added `go.mod` (was missing — not a valid Go module before)
- Removed hardcoded `zsh` dependency — no longer shells out
- Removed hardcoded `pbcopy` — cross-platform clipboard detection
- Added `SKILLS.md` for LLM-assisted development context

## v0.1.0

Initial release.
- Store CLI commands in a markdown vault file
- Browse with fzf, copy to clipboard
- Auto-categorize with Gemini AI
