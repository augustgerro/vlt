package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Config struct {
	VaultPath string `json:"vault_path"`
}

type VaultEntry struct {
	Category    string
	Description string
	Command     string
}

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func loadConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("cannot determine home directory: %w", err)
	}
	configPath := filepath.Join(home, ".config", "vlt", "config.json")

	conf := Config{
		VaultPath: filepath.Join(home, ".cli_vault.md"),
	}

	file, err := os.Open(configPath)
	if err == nil {
		defer file.Close()
		if err := json.NewDecoder(file).Decode(&conf); err != nil {
			return conf, fmt.Errorf("invalid config JSON: %w", err)
		}
	}
	return conf, nil
}

func parseVault(vaultPath string) ([]VaultEntry, error) {
	data, err := os.ReadFile(vaultPath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var entries []VaultEntry

	for i, line := range lines {
		if i < 2 {
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "|") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}
		cat := strings.TrimSpace(parts[1])
		desc := strings.TrimSpace(parts[2])
		// Command may contain unescaped | — rejoin remaining fields
		cmd := strings.TrimSpace(strings.Join(parts[3:len(parts)-1], "|"))
		cmd = strings.Trim(cmd, "`")
		cmd = strings.ReplaceAll(cmd, "\\|", "|")

		if cat == "" && desc == "" && cmd == "" {
			continue
		}
		entries = append(entries, VaultEntry{Category: cat, Description: desc, Command: cmd})
	}
	return entries, nil
}

func uniqueCategories(entries []VaultEntry) []string {
	seen := map[string]bool{}
	var cats []string
	for _, e := range entries {
		if !seen[e.Category] {
			seen[e.Category] = true
			cats = append(cats, e.Category)
		}
	}
	return cats
}

func clipboardCmd() (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "pbcopy", nil
	case "linux":
		for _, tool := range []string{"xclip", "xsel", "wl-copy"} {
			if isCommandAvailable(tool) {
				if tool == "xclip" {
					return tool, []string{"-selection", "clipboard"}
				}
				return tool, nil
			}
		}
	}
	return "", nil
}

func showHelp(conf Config) {
	fmt.Println(`vlt — CLI Vault Tool

Usage:
  vlt                       Browse saved commands (interactive fzf)
  vlt "command"             Add a command (manual category/description)
  vlt --ai "command"        Add with AI categorization (currently: Gemini)
  vlt --help, -h            Show this help

Options:
  --ai, -a    Use AI to auto-categorize the command

Browse Hotkeys:
  →/Ctrl-F   Navigate to categories
  ←          Navigate back
  Ctrl-E     Edit selected entry ($EDITOR)
  Ctrl-D     Delete selected entry
  ↵           Copy selected command to clipboard
  Esc         Exit

Vault file: ` + conf.VaultPath + `

Dependencies:
  fzf        (required)  Interactive command browser
  gemini     (optional)  AI backend for --ai flag`)

	clipTool, _ := clipboardCmd()
	if clipTool != "" {
		fmt.Printf("  %-10s (found)     Clipboard support\n", clipTool)
	} else {
		fmt.Println("  clipboard  (missing)   Install pbcopy/xclip/xsel/wl-copy for clipboard")
	}
}

func main() {
	conf, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	// Auto-create vault if it doesn't exist
	if _, err := os.Stat(conf.VaultPath); os.IsNotExist(err) {
		fmt.Printf("Vault file not found. Creating a new one at: %s\n", conf.VaultPath)
		f, err := os.Create(conf.VaultPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating vault file: %v\n", err)
			os.Exit(1)
		}
		f.WriteString("| Category | Description | Command |\n| :--- | :--- | :--- |\n")
		f.Close()
	}

	// Hidden flags for CLI scripting
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "--_list":
			catFilter := ""
			if len(os.Args) >= 4 && os.Args[2] == "--_cat" {
				catFilter = strings.Join(os.Args[3:], " ")
			}
			listEntries(conf.VaultPath, catFilter)
			return
		case "--_cats":
			entries, err := parseVault(conf.VaultPath)
			if err != nil {
				os.Exit(1)
			}
			for _, cat := range uniqueCategories(entries) {
				fmt.Println(cat)
			}
			return
		case "--_delete":
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				deleteEntry(conf.VaultPath, scanner.Text())
			}
			return
		}
	}

	if len(os.Args) == 1 {
		showVault(conf.VaultPath)
		return
	}

	// Parse flags
	useAI := false
	args := []string{}
	for _, a := range os.Args[1:] {
		switch a {
		case "--help", "-h":
			showHelp(conf)
			return
		case "--ai", "-a":
			useAI = true
		default:
			args = append(args, a)
		}
	}

	if len(args) == 0 {
		showHelp(conf)
		return
	}

	cmdToStore := strings.Join(args, " ")
	category := "Misc"
	description := "Manual entry"

	if useAI {
		if !isCommandAvailable("gemini") {
			fmt.Fprintln(os.Stderr, "Gemini CLI not found. Install: https://github.com/google-gemini/gemini-cli")
			fmt.Println("Falling back to manual input.")
			readManualInput(&category, &description)
		} else {
			fmt.Printf("Categorizing with Gemini AI: %s...\n", cmdToStore)
			prompt := fmt.Sprintf(
				"Analyze this CLI command: '%s'. Return ONLY one line in format: Category | Short Description (max 5 words). No explanation.",
				cmdToStore,
			)
			out, err := exec.Command("gemini", "-p", prompt).Output()
			if err == nil {
				line := strings.TrimSpace(strings.Split(string(out), "\n")[0])
				parts := strings.SplitN(line, "|", 2)
				if len(parts) == 2 && len(parts[0]) < 30 && len(parts[1]) < 60 {
					category = strings.TrimSpace(parts[0])
					description = strings.TrimSpace(parts[1])
				} else {
					fmt.Println("AI response unexpected, falling back to manual input.")
					readManualInput(&category, &description)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Gemini error: %v, falling back to manual input.\n", err)
				readManualInput(&category, &description)
			}
		}
	} else {
		readManualInput(&category, &description)
	}

	// Escape pipe characters so they don't break the markdown table
	escapedCmd := strings.ReplaceAll(cmdToStore, "|", "\\|")
	newRow := fmt.Sprintf("| %s | %s | `%s` |\n", category, description, escapedCmd)

	f, err := os.OpenFile(conf.VaultPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening vault: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if _, err := f.WriteString(newRow); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to vault: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[Vault: Updated +1]")
}

func readManualInput(category, description *string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Category (e.g. Git, Docker): ")
	c, _ := reader.ReadString('\n')
	*category = strings.TrimSpace(c)

	fmt.Print("Description (max 5 words): ")
	d, _ := reader.ReadString('\n')
	*description = strings.TrimSpace(d)
}

func listEntries(vaultPath, catFilter string) {
	entries, err := parseVault(vaultPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading vault: %v\n", err)
		os.Exit(1)
	}

	maxCat, maxDesc := 0, 0
	for _, e := range entries {
		if len(e.Category) > maxCat {
			maxCat = len(e.Category)
		}
		if len(e.Description) > maxDesc {
			maxDesc = len(e.Description)
		}
	}

	sep := "│"
	for _, e := range entries {
		if catFilter != "" && e.Category != catFilter {
			continue
		}
		if catFilter != "" {
			fmt.Printf(" %-*s %s %s\n", maxDesc, e.Description, sep, e.Command)
		} else {
			fmt.Printf(" %-*s %s %-*s %s %s\n",
				maxCat, e.Category, sep,
				maxDesc, e.Description, sep,
				e.Command)
		}
	}
}

func editEntry(vaultPath, fzfLine string) {
	sep := "│"
	parts := strings.Split(fzfLine, sep)
	if len(parts) < 2 {
		return
	}

	// Parse fields from the fzf display line (2 or 3 columns)
	var origCat, origDesc, origCmd string
	if len(parts) >= 3 {
		origCat = strings.TrimSpace(parts[0])
		origDesc = strings.TrimSpace(parts[1])
		origCmd = strings.TrimSpace(parts[2])
	} else {
		origDesc = strings.TrimSpace(parts[0])
		origCmd = strings.TrimSpace(parts[1])
	}

	// Locate the original vault line by command to get the real category
	// (needed when viewing in filtered/2-col mode)
	data, err := os.ReadFile(vaultPath)
	if err != nil {
		return
	}
	vaultLines := strings.Split(string(data), "\n")
	matchedLine := ""
	matchedIdx := -1
	cmdEscaped := "`" + strings.ReplaceAll(origCmd, "|", "\\|") + "`"
	cmdRaw := "`" + origCmd + "`"
	for i, line := range vaultLines {
		if strings.Contains(line, cmdEscaped) || strings.Contains(line, cmdRaw) {
			matchedLine = line
			matchedIdx = i
			break
		}
	}
	if matchedIdx < 0 {
		return
	}

	// Re-parse the matched vault line to get full accurate fields
	vParts := strings.Split(matchedLine, "|")
	if len(vParts) >= 4 {
		origCat = strings.TrimSpace(vParts[1])
		origDesc = strings.TrimSpace(vParts[2])
		origCmd = strings.TrimSpace(strings.Join(vParts[3:len(vParts)-1], "|"))
		origCmd = strings.Trim(origCmd, "`")
		origCmd = strings.ReplaceAll(origCmd, "\\|", "|")
	}

	// Write temp file for editor
	tmp, err := os.CreateTemp("", "vlt-edit-*.txt")
	if err != nil {
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	fmt.Fprintf(tmp, "# Edit vault entry — save and close to apply, delete all lines to cancel\n\n")
	fmt.Fprintf(tmp, "Category:    %s\n", origCat)
	fmt.Fprintf(tmp, "Description: %s\n", origDesc)
	fmt.Fprintf(tmp, "Command:     %s\n", origCmd)
	tmp.Close()

	// Open $EDITOR (fallback: nano → vi)
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		for _, e := range []string{"nano", "vi", "vim"} {
			if isCommandAvailable(e) {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		fmt.Fprintln(os.Stderr, "No editor found. Set $EDITOR.")
		return
	}

	editorCmd := exec.Command(editor, tmpPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	if err := editorCmd.Run(); err != nil {
		return
	}

	// Parse edited file
	edited, err := os.ReadFile(tmpPath)
	if err != nil {
		return
	}
	newCat, newDesc, newCmd := origCat, origDesc, origCmd
	for _, line := range strings.Split(string(edited), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		if after, ok := strings.CutPrefix(line, "Category:"); ok {
			newCat = strings.TrimSpace(after)
		} else if after, ok := strings.CutPrefix(line, "Description:"); ok {
			newDesc = strings.TrimSpace(after)
		} else if after, ok := strings.CutPrefix(line, "Command:"); ok {
			newCmd = strings.TrimSpace(after)
		}
	}

	// Nothing changed — skip write
	if newCat == origCat && newDesc == origDesc && newCmd == origCmd {
		return
	}

	escapedCmd := strings.ReplaceAll(newCmd, "|", "\\|")
	newRow := fmt.Sprintf("| %s | %s | `%s` |", newCat, newDesc, escapedCmd)
	vaultLines[matchedIdx] = newRow

	if err := os.WriteFile(vaultPath, []byte(strings.Join(vaultLines, "\n")), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing vault: %v\n", err)
	}
}

func deleteEntry(vaultPath, fzfLine string) {
	sep := "│"
	parts := strings.Split(fzfLine, sep)
	if len(parts) < 2 {
		return
	}
	command := strings.TrimSpace(parts[len(parts)-1])

	// Match vault line by command — try both escaped (\|) and unescaped (|) variants
	vaultCmdEscaped := "`" + strings.ReplaceAll(command, "|", "\\|") + "`"
	vaultCmdRaw := "`" + command + "`"

	data, err := os.ReadFile(vaultPath)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	var result []string
	deleted := false

	for _, line := range lines {
		if !deleted && (strings.Contains(line, vaultCmdEscaped) || strings.Contains(line, vaultCmdRaw)) {
			deleted = true
			continue
		}
		result = append(result, line)
	}

	if deleted {
		if err := os.WriteFile(vaultPath, []byte(strings.Join(result, "\n")), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing vault: %v\n", err)
		}
	}
}

func showVault(vaultPath string) {
	if !isCommandAvailable("fzf") {
		fmt.Fprintln(os.Stderr, "Error: fzf is required but not installed. Install it: brew install fzf")
		os.Exit(1)
	}

	// State machine: "all" → "categories" → "filtered"
	state := "all"
	catFilter := ""

	for {
		switch state {
		case "all":
			action, selected := runVaultFzf(vaultPath, "")
			switch action {
			case "ctrl-d":
				if selected != "" {
					deleteEntry(vaultPath, selected)
				}
			case "ctrl-e":
				if selected != "" {
					editEntry(vaultPath, selected)
				}
			case "ctrl-f", "right":
				state = "categories"
			case "enter":
				if selected != "" {
					copyCommand(selected)
				}
				return
			default:
				return
			}

		case "categories":
			action, picked := runCategoryPicker(vaultPath, catFilter)
			switch action {
			case "right", "enter":
				if picked != "" {
					catFilter = picked
					state = "filtered"
				}
			case "left":
				state = "all"
			default:
				return
			}

		case "filtered":
			action, selected := runVaultFzf(vaultPath, catFilter)
			switch action {
			case "ctrl-d":
				if selected != "" {
					deleteEntry(vaultPath, selected)
				}
			case "ctrl-e":
				if selected != "" {
					editEntry(vaultPath, selected)
				}
			case "left":
				state = "categories"
			case "ctrl-f", "right":
				state = "categories"
			case "enter":
				if selected != "" {
					copyCommand(selected)
				}
				return
			default:
				return
			}
		}
	}
}

func runVaultFzf(vaultPath, catFilter string) (action, selected string) {
	entries, err := parseVault(vaultPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading vault: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Println("Vault is empty. Add commands with: vlt \"your command\"")
		os.Exit(0)
	}

	maxCat, maxDesc := 0, 0
	for _, e := range entries {
		if len(e.Category) > maxCat {
			maxCat = len(e.Category)
		}
		if len(e.Description) > maxDesc {
			maxDesc = len(e.Description)
		}
	}

	sep := "│"
	var fzfLines []string
	for _, e := range entries {
		if catFilter != "" && e.Category != catFilter {
			continue
		}
		if catFilter != "" {
			fzfLines = append(fzfLines, fmt.Sprintf(" %-*s %s %s",
				maxDesc, e.Description, sep, e.Command))
		} else {
			fzfLines = append(fzfLines, fmt.Sprintf(" %-*s %s %-*s %s %s",
				maxCat, e.Category, sep,
				maxDesc, e.Description, sep,
				e.Command))
		}
	}

	if len(fzfLines) == 0 {
		fmt.Println("No entries in this category.")
		return "esc", ""
	}

	var header string
	if catFilter != "" {
		header = " \033[33m←\033[0m Back  \033[33mctrl-e\033[0m Edit  \033[33mctrl-d\033[0m Delete  \033[33m↵\033[0m Copy  \033[33mEsc\033[0m Exit"
	} else {
		header = " \033[33m→\033[0m Categories  \033[33mctrl-e\033[0m Edit  \033[33mctrl-d\033[0m Delete  \033[33m↵\033[0m Copy  \033[33mEsc\033[0m Exit"
	}

	preview := `echo {} | awk -F '│' '{for(i=1;i<=NF;i++) gsub(/^[ \t]+|[ \t]+$/,"",$i); if(NF>=3) printf "\033[1;36m📂 Category:\033[0m    %s\n\033[1;33m📝 Description:\033[0m %s\n\033[1;32m⚡ Command:\033[0m     %s\n",$1,$2,$3; else printf "\033[1;33m📝 Description:\033[0m %s\n\033[1;32m⚡ Command:\033[0m     %s\n",$1,$2}'`

	borderLabel := " 🔐 Vault "
	if catFilter != "" {
		borderLabel = fmt.Sprintf(" 🔐 Vault › %s ", catFilter)
	}

	fzfArgs := []string{
		"--reverse",
		"--exact",
		"--ansi",
		"--delimiter", sep,
		"--border", "rounded",
		"--border-label", borderLabel,
		"--header", header,
		"--preview", preview,
		"--preview-window", "up:5:wrap",
		"--expect", "ctrl-d,ctrl-e,ctrl-f,right,left",
	}

	cmd := exec.Command("fzf", fzfArgs...)
	cmd.Stdin = strings.NewReader(strings.Join(fzfLines, "\n"))
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return "esc", ""
	}

	lines := strings.SplitN(strings.TrimRight(string(out), "\n"), "\n", 2)
	key := strings.TrimSpace(lines[0])
	line := ""
	if len(lines) >= 2 {
		line = lines[1]
	}

	if key == "" {
		return "enter", line
	}
	return key, line
}

func runCategoryPicker(vaultPath, highlight string) (action, picked string) {
	entries, err := parseVault(vaultPath)
	if err != nil {
		return "esc", ""
	}
	cats := uniqueCategories(entries)
	if len(cats) == 0 {
		return "esc", ""
	}

	// Count entries per category
	catCount := map[string]int{}
	for _, e := range entries {
		catCount[e.Category]++
	}

	// Find position of highlighted category (1-indexed for fzf pos())
	highlightPos := 0
	if highlight != "" {
		for i, c := range cats {
			if c == highlight {
				highlightPos = i + 1
				break
			}
		}
	}

	var lines []string
	for _, c := range cats {
		lines = append(lines, fmt.Sprintf(" %-20s (%d)", c, catCount[c]))
	}

	header := " \033[33m←\033[0m Back  \033[33m→/↵\033[0m Open  \033[33mEsc\033[0m Exit"

	fzfArgs := []string{
		"--reverse",
		"--exact",
		"--ansi",
		"--border", "rounded",
		"--border-label", " 📂 Categories ",
		"--header", header,
		"--expect", "right,left",
	}
	if highlightPos > 0 {
		fzfArgs = append(fzfArgs, "--sync", "--bind", fmt.Sprintf("start:pos(%d)", highlightPos))
	}

	cmd := exec.Command("fzf", fzfArgs...)
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return "esc", ""
	}

	outLines := strings.SplitN(strings.TrimRight(string(out), "\n"), "\n", 2)
	key := strings.TrimSpace(outLines[0])
	sel := ""
	if len(outLines) >= 2 {
		sel = outLines[1]
	}

	// Extract category name (strip count suffix)
	catName := ""
	if sel != "" {
		sel = strings.TrimSpace(sel)
		if idx := strings.LastIndex(sel, "("); idx > 0 {
			catName = strings.TrimSpace(sel[:idx])
		} else {
			catName = sel
		}
	}

	switch key {
	case "left":
		return "left", ""
	case "right":
		return "right", catName
	default:
		// Enter
		if catName != "" {
			return "enter", catName
		}
		return "esc", ""
	}
}

func copyCommand(fzfLine string) {
	sep := "│"
	parts := strings.Split(fzfLine, sep)
	command := strings.TrimSpace(parts[len(parts)-1])

	clipTool, clipArgs := clipboardCmd()
	if clipTool == "" {
		fmt.Println(command)
		fmt.Fprintln(os.Stderr, "No clipboard tool found. Command printed above.")
		return
	}

	clip := exec.Command(clipTool, clipArgs...)
	clip.Stdin = strings.NewReader(command)
	if err := clip.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Clipboard error: %v\n", err)
		fmt.Println(command)
		return
	}
	fmt.Println("Command copied to clipboard!")
}
