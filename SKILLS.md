# vlt — CLI Vault Tool

## What This Project Is
A lightweight Go CLI tool that stores, categorizes, and recalls useful shell commands.
Commands are saved to a markdown table file (`~/.cli_vault.md`) and browsed interactively via `fzf`.
Optional Gemini AI integration auto-categorizes commands.

## Architecture
- Single-file Go binary (`main.go`), no external Go dependencies.
- Storage: markdown table with columns `Category | Description | Command`.
- Config: `~/.config/vlt/config.json` (optional, overrides vault path).
- Interactive browse: shells out to `fzf` + `awk` + `pbcopy`.

## Known Issues (Priority Order)
1. ~~**No `go.mod`**~~ — ✅ Fixed.
2. ~~**Shell injection in `showVault()`**~~ — ✅ Fixed. Now uses Go-native file I/O + fzf via stdin.
3. ~~**All errors silently ignored**~~ — ✅ Fixed. Errors are handled and reported.
4. ~~**"Copied!" always prints**~~ — ✅ Fixed. Only prints on successful clipboard copy.
5. ~~**Pipe `|` in commands breaks markdown table**~~ — ✅ Fixed. Pipes escaped as `\|` on store, unescaped on read.
6. ~~**macOS-only**~~ — ✅ Fixed. Cross-platform clipboard (pbcopy/xclip/xsel/wl-copy).
7. ~~**No CLI flags**~~ — ✅ Fixed. `--help` / `-h` added with vault path display.
8. ~~**`fzf` not checked**~~ — ✅ Fixed. Clear error message if fzf missing.
9. ~~**Gemini output parsing fragile**~~ — ✅ Fixed. Validates format + length, falls back to manual.
10. **README says `column` is a dependency** — but code never uses it. (Still needs README update)
11. **Makefile `test` target** is not a real test — just runs the binary. (Needs real tests)
12. ~~**fzf used fuzzy matching**~~ — ✅ Fixed. Now uses `--exact` for full-word search.

## Rules for Working on This Project
- Always run `go vet ./...` and `gofmt -l .` after changes.
- Never interpolate variables into shell command strings. Use Go-native file operations or pass as arguments.
- Handle every error. Do not assign to `_` unless there is a documented reason.
- Validate external tool availability (`fzf`, `pbcopy`, `gemini`) at runtime with clear error messages.
- Check `cmd.Run()` / `cmd.Output()` return values before acting on results.
- Keep it a single-file tool — simplicity is a feature.
- The markdown table format is fragile. If refactoring storage, prefer JSON lines (`~/.cli_vault.jsonl`).
- **Never** add `Co-authored-by`, `Generated-by`, or any trailer/reference to AI or LLMs in commits, code comments, or metadata.
- Use **scope-based** commit style: `[scope] Imperative message`
- Common scopes: `core`, `cli`, `ai`, `storage`, `clipboard`, `security`, `docs`, `build`
- Example: `[cli] Add --help flag with vault path display`

## Building and Testing
```bash
go build -o vlt main.go   # build
go vet ./...               # lint
make install               # build + move to ~/go/bin/
```
There are no real tests yet. When adding tests, use Go's `testing` package in `main_test.go`.

## Recommended Fix Order
1. `go mod init github.com/augustgerro/vlt`
2. Fix `showVault()` — check fzf exists, handle errors, don't hardcode pbcopy/zsh
3. Add proper error handling throughout `main()` and `loadConfig()`
4. Escape or encode pipe chars in stored commands
5. Add `--help` flag
6. Add basic tests for config loading and row generation
7. Update README to remove `column` from deps, add Linux clipboard info
