# Release Notes

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
