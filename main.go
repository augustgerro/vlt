package main

import (
	"bufio"
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

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func loadConfig() (Config, error) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "vlt", "config.json")
	
	conf := Config{
		VaultPath: filepath.Join(home, ".cli_vault.md"),
	}

	file, err := os.Open(configPath)
	if err == nil {
		defer file.Close()
		decoder := json.NewDecoder(file)
		decoder.Decode(&conf)
	}
	return conf, nil
}

func main() {
	conf, _ := loadConfig()

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
	category := "Misc"
	description := "Manual entry"

	if isCommandAvailable("gemini") {
		fmt.Printf("Categorizing with Gemini AI: %s...\n", cmdToStore)
		prompt := fmt.Sprintf("Analyze this CLI command: '%s'. Return ONLY one line in format: Category | Short Description (max 5 words).", cmdToStore)
		out, err := exec.Command("gemini", "-p", prompt).Output()
		if err == nil {
			parts := strings.Split(string(out), "|")
			if len(parts) >= 2 {
				category = strings.TrimSpace(parts[0])
				description = strings.TrimSpace(parts[1])
			}
		}
	} else {
		fmt.Println("Gemini CLI not found. Please enter details manually:")
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Category (e.g. Git, Docker): ")
		category, _ = reader.ReadString('\n')
		category = strings.TrimSpace(category)
		
		fmt.Print("Description (max 5 words): ")
		description, _ = reader.ReadString('\n')
		description = strings.TrimSpace(description)
	}

	newRow := fmt.Sprintf("| %s | %s | `%s` |\n", category, description, cmdToStore)

	f, err := os.OpenFile(conf.VaultPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening vault: %v\n", err)
		return
	}
	defer f.Close()

	f.WriteString(newRow)
	fmt.Println("[Vault: Updated +1]")
}

func showVault(vaultPath string) {
	// 1. Get the line from fzf
	// 2. Extract the 3rd column (Command) correctly using sed/awk
	// 3. Clean up backticks and copy to clipboard
	script := fmt.Sprintf(`cat %s | fzf --header-lines=2 --reverse --no-preview --header "ENTER to copy command, ESC to exit" | awk -F'|' '{print $4}' | sed 's/^[[:space:]]*\x60//;s/\x60[[:space:]]*$//' | xargs | pbcopy`, vaultPath)
	
	cmd := exec.Command("zsh", "-c", script)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Run()
	fmt.Println("Command copied to clipboard!")
}
