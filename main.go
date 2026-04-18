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

func showVault(vaultPath string) {
	if !isCommandAvailable("fzf") {
		fmt.Fprintln(os.Stderr, "Error: fzf is required but not installed. Install it: brew install fzf")
		os.Exit(1)
	}

	data, err := os.ReadFile(vaultPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading vault: %v\n", err)
		os.Exit(1)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) <= 2 {
		fmt.Println("Vault is empty. Add commands with: vlt \"your command\"")
		return
	}

	// Feed only data rows (skip header + separator) into fzf via stdin
	dataRows := strings.Join(lines[2:], "\n")

	fzfArgs := []string{
		"--reverse",
		"--no-preview",
		"--exact",
		"--header", "ENTER to copy command, ESC to exit",
	}
	cmd := exec.Command("fzf", fzfArgs...)
	cmd.Stdin = strings.NewReader(dataRows)
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		// User pressed ESC or fzf exited with non-zero — silently exit
		return
	}

	selected := strings.TrimSpace(string(out))
	if selected == "" {
		return
	}

	// Extract command column (4th field between pipes)
	parts := strings.Split(selected, "|")
	if len(parts) < 4 {
		fmt.Fprintln(os.Stderr, "Error: could not parse selected row.")
		return
	}
	command := strings.TrimSpace(parts[3])
	// Strip backticks
	command = strings.Trim(command, "`")
	// Unescape pipes
	command = strings.ReplaceAll(command, "\\|", "|")

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
