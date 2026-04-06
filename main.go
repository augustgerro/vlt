package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Config struct {
	VaultPath string `json:"vault_path"`
}

func loadConfig() (Config, error) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "vlt", "config.json")
	
	conf := Config{
		VaultPath: filepath.Join(home, ".cli_vault.md"),
	}

	file, err := os.Open(configPath)
	if err != nil {
		return conf, fmt.Errorf("config not found at %s", configPath)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.Decode(&conf)
	return conf, nil
}

func main() {
	conf, err := loadConfig()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Auto-create vault if it doesn't exist
	if _, err := os.Stat(conf.VaultPath); os.IsNotExist(err) {
		fmt.Printf("Vault file not found. Creating a new one at: %s\n", conf.VaultPath)
		f, err := os.Create(conf.VaultPath)
		if err != nil {
			fmt.Printf("Error creating vault file: %v\n", err)
			os.Exit(1)
		}
		f.WriteString("| Category | Description | Command |\n| :--- | :--- | :--- |\n")
		f.Close()
	}

	if len(os.Args) == 1 {
		showVault(conf.VaultPath)
		return
	}

	cmdToStore := strings.Join(os.Args[1:], " ")
	fmt.Printf("Categorizing: %s...\n", cmdToStore)

	prompt := fmt.Sprintf("Analyze this CLI command: '%s'. Return ONLY one line in format: Category | Short Description (max 5 words).", cmdToStore)
	out, err := exec.Command("gemini", "-p", prompt).Output()
	if err != nil {
		fmt.Printf("Error calling Gemini: %v\n", err)
		return
	}

	parts := strings.Split(string(out), "|")
	category := "Misc"
	description := "Manual entry"
	if len(parts) >= 2 {
		category = strings.TrimSpace(parts[0])
		description = strings.TrimSpace(parts[1])
	}

	newRow := fmt.Sprintf("| %s | %s | `%s` |\n", category, description, cmdToStore)

	f, err := os.OpenFile(conf.VaultPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening vault for writing: %v\n", err)
		return
	}
	defer f.Close()

	f.WriteString(newRow)
	fmt.Println("[Vault: Updated +1]")
}

func showVault(vaultPath string) {
	// 1. Cat the file
	// 2. Remove leading/trailing pipes and trim spaces
	// 3. Format with column -t for clean table view
	// 4. Pass to fzf with header and no preview
	script := fmt.Sprintf(`cat %s | sed 's/^|[[:space:]]*//;s/[[:space:]]*|[[:space:]]*$//' | column -t -s '|' | fzf --header-lines=2 --reverse --no-preview --header "ENTER to copy command, ESC to exit" | awk '{print $NF}' | sed 's/^\x60//;s/\x60$//' | pbcopy`, vaultPath)
	
	cmd := exec.Command("zsh", "-c", script)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Run()
	fmt.Println("Command copied to clipboard!")
}
