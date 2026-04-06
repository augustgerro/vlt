# vlt (Vault) CLI Tool

A lightweight CLI assistant to store, categorize, and recall useful commands.

## Features
- **Smart Categorization**: Uses Gemini AI to automatically assign categories and descriptions.
- **Interactive Search**: Uses `fzf` to browse and copy commands to the clipboard.
- **Safe Mode**: Skips execution if the vault file doesn't exist.
- **Configurable**: Settings stored in `~/.config/vlt/config.json`.

## Preview
```bash
# Add a command
vlt "find . -name '*.log' -mtime +7 -delete"
# Output: [Vault: Updated +1]

# Browse and copy (interactive fzf)
vlt
```

## Dependencies
To use all features of `vlt`, ensure the following are installed:
- **Go** (1.18+) - For building the tool.
- **fzf** - For the interactive command browser.
- **Gemini CLI** (`gemini`) - For smart AI categorization.
- **Standard Unix Tools**: `column`, `sed`, `awk` (usually pre-installed on macOS/Linux).
- **pbcopy** (macOS) - To support automatic clipboard copying.

## Installation
```bash
make install
```

## Usage
1. **Add a command**: `vlt "your command here"`
2. **Add last command**: `vlt !!`
3. **Browse commands**: `vlt` (opens interactive fzf)

## Configuration
Edit `~/.config/vlt/config.json` to change the `vault_path`.
