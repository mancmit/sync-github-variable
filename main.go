package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	githubAPIURL = "https://api.github.com"
)

// Shared HTTP client with timeout for all API requests
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

type Variable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Command-line flags
var (
	diffMode   = flag.Bool("diff", false, "Show diff and exit without syncing")
	backupMode = flag.Bool("backup", false, "Create backup and exit without syncing")
	noBackup   = flag.Bool("no-backup", false, "Skip automatic backup before syncing")
)

func main() {
	// Parse command-line flags
	flag.Parse()

	// Get information from environment variables
	token := os.Getenv("GITHUB_TOKEN")
	owner := os.Getenv("GITHUB_OWNER")
	repo := os.Getenv("GITHUB_REPO")
	environment := os.Getenv("GITHUB_ENVIRONMENT") // Optional: for environment-specific variables

	if token == "" || owner == "" || repo == "" {
		fmt.Println("❌ Missing required information!")
		fmt.Println("Please set the following environment variables:")
		fmt.Println("  GITHUB_TOKEN        - GitHub Personal Access Token")
		fmt.Println("  GITHUB_OWNER        - Owner/organization name")
		fmt.Println("  GITHUB_REPO         - Repository name")
		fmt.Println("  GITHUB_ENVIRONMENT  - (Optional) Environment name (e.g., production, staging)")
		os.Exit(1)
	}

	fmt.Println("^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^")

	// Display sync target
	if environment != "" {
		fmt.Printf("🎯 Target: Environment '%s' in %s/%s\n", environment, owner, repo)
	} else {
		fmt.Printf("🎯 Target: Repository %s/%s\n", owner, repo)
	}

	// Handle manual backup mode
	if *backupMode {
		handleBackupMode(token, owner, repo, environment)
		return
	}

	// Read CSV file
	variables, err := readCSV("variables.csv")
	if err != nil {
		fmt.Printf("❌ Error reading CSV file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📝 Read %d variables from CSV file\n", len(variables))

	// Fetch current GitHub variables
	fmt.Println("🔍 Fetching current variables from GitHub...")
	remoteVariables, err := FetchGitHubVariables(token, owner, repo, environment)
	if err != nil {
		fmt.Printf("❌ Error fetching GitHub variables: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Fetched %d variables from GitHub\n", len(remoteVariables))

	// Compare local and remote variables
	diffResult := CompareSets(variables, remoteVariables)

	// Display diff summary and details
	DisplayDiffSummary(diffResult)
	DisplayDetailedDiff(diffResult)

	// If --diff flag is set, exit after showing diff
	if *diffMode {
		fmt.Println("ℹ️  Diff mode: No changes were made")
		os.Exit(0)
	}

	// Calculate variables to sync (only new and updated)
	variablesToSync := []Variable{}
	variablesToSync = append(variablesToSync, diffResult.New...)
	for _, updated := range diffResult.Updated {
		variablesToSync = append(variablesToSync, Variable{
			Name:  updated.Name,
			Value: updated.NewValue,
		})
	}

	// If nothing to sync, exit
	if len(variablesToSync) == 0 {
		fmt.Println("\n✅ No changes to sync. All variables are up to date!")
		os.Exit(0)
	}

	// Show confirmation before syncing
	if !confirmSync(owner, repo, environment, token, diffResult) {
		fmt.Println("\n❌ Sync cancelled by user")
		os.Exit(0)
	}

	// Auto-backup before syncing (unless disabled)
	if !*noBackup {
		fmt.Println("\n💾 Creating backup before sync...")
		backupFile, err := BackupGitHubVariables(token, owner, repo, environment)
		if err != nil {
			fmt.Printf("⚠️  Warning: Failed to create backup: %v\n", err)
			fmt.Print("Continue without backup? (yes/no): ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))
			if input != "yes" && input != "y" {
				fmt.Println("❌ Sync cancelled")
				os.Exit(0)
			}
		} else {
			fmt.Printf("✅ Backup saved: %s\n", backupFile)
		}
	}

	fmt.Println("\n🚀 Starting sync...\n")

	// Create a map of new variables for O(1) lookup
	newVarMap := make(map[string]bool)
	for _, v := range diffResult.New {
		newVarMap[v.Name] = true
	}

	// Sync only the changed variables
	newCount := 0
	updateCount := 0
	failedCount := 0
	for _, variable := range variablesToSync {
		if variable.Name == "" {
			continue
		}

		err := syncVariable(token, owner, repo, environment, variable)
		if err != nil {
			fmt.Printf("❌ Error syncing variable '%s': %v\n", variable.Name, err)
			failedCount++
		} else {
			// Check if this is a new or updated variable using map lookup (O(1))
			if newVarMap[variable.Name] {
				fmt.Printf("✅ Created variable: %s\n", variable.Name)
				newCount++
			} else {
				fmt.Printf("✅ Updated variable: %s\n", variable.Name)
				updateCount++
			}
		}
	}

	// Display final results
	fmt.Println()
	if failedCount > 0 {
		fmt.Printf("🎉 Completed! Created %d, Updated %d, Failed %d, Total %d variables\n", 
			newCount, updateCount, failedCount, newCount+updateCount+failedCount)
	} else {
		fmt.Printf("🎉 Completed! Created %d, Updated %d, Total %d variables\n", 
			newCount, updateCount, newCount+updateCount)
	}
}

func readCSV(filename string) ([]Variable, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	
	// Read header (skip first line)
	_, err = reader.Read()
	if err != nil {
		return nil, err
	}

	variables := []Variable{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if len(record) >= 2 {
			key := strings.TrimSpace(record[0])
			value := strings.TrimSpace(record[1])
			
			if key != "" {
				variables = append(variables, Variable{
					Name:  key,
					Value: value,
				})
			}
		}
	}

	return variables, nil
}

func confirmSync(owner, repo, environment, token string, diff DiffResult) bool {
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📋 SYNC CONFIGURATION")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	// Display target information
	fmt.Printf("Repository:  %s/%s\n", owner, repo)
	if environment != "" {
		fmt.Printf("Environment: %s\n", environment)
		fmt.Printf("Target:      Environment-specific variables\n")
	} else {
		fmt.Printf("Environment: (none)\n")
		fmt.Printf("Target:      Repository-level variables\n")
	}
	
	// Mask token for display
	maskedToken := maskToken(token)
	fmt.Printf("Token:       %s\n", maskedToken)
	
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	// Display sync summary
	totalToSync := len(diff.New) + len(diff.Updated)
	fmt.Printf("\n📦 Will sync %d variable(s) (%d new, %d updated)\n", 
		totalToSync, len(diff.New), len(diff.Updated))
	
	// Ask for confirmation
	fmt.Print("\n⚠️  Do you want to proceed with the sync? (yes/no): ")
	
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "yes" || input == "y"
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	// Show first 4 and last 4 characters
	return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}

func syncVariable(token, owner, repo, environment string, variable Variable) error {
	// Check if variable already exists
	exists, err := checkVariableExists(token, owner, repo, environment, variable.Name)
	if err != nil {
		return err
	}

	if exists {
		// Update existing variable
		return updateVariable(token, owner, repo, environment, variable)
	}
	
	// Create new variable
	return createVariable(token, owner, repo, environment, variable)
}

func checkVariableExists(token, owner, repo, environment, name string) (bool, error) {
	var url string
	if environment != "" {
		// Environment-specific variable
		url = fmt.Sprintf("%s/repos/%s/%s/environments/%s/variables/%s", githubAPIURL, owner, repo, environment, name)
	} else {
		// Repository-level variable
		url = fmt.Sprintf("%s/repos/%s/%s/actions/variables/%s", githubAPIURL, owner, repo, name)
	}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200, nil
}

func createVariable(token, owner, repo, environment string, variable Variable) error {
	var url string
	if environment != "" {
		// Environment-specific variable
		url = fmt.Sprintf("%s/repos/%s/%s/environments/%s/variables", githubAPIURL, owner, repo, environment)
	} else {
		// Repository-level variable
		url = fmt.Sprintf("%s/repos/%s/%s/actions/variables", githubAPIURL, owner, repo)
	}
	
	payload := map[string]string{
		"name":  variable.Name,
		"value": variable.Value,
	}
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func updateVariable(token, owner, repo, environment string, variable Variable) error {
	var url string
	if environment != "" {
		// Environment-specific variable
		url = fmt.Sprintf("%s/repos/%s/%s/environments/%s/variables/%s", githubAPIURL, owner, repo, environment, variable.Name)
	} else {
		// Repository-level variable
		url = fmt.Sprintf("%s/repos/%s/%s/actions/variables/%s", githubAPIURL, owner, repo, variable.Name)
	}
	
	payload := map[string]string{
		"name":  variable.Name,
		"value": variable.Value,
	}
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// handleBackupMode creates a backup of GitHub variables
func handleBackupMode(token, owner, repo, environment string) {
	fmt.Println("💾 Backup Mode: Creating backup of GitHub variables...")
	
	backupFile, err := BackupGitHubVariables(token, owner, repo, environment)
	if err != nil {
		fmt.Printf("❌ Error creating backup: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("✅ Backup saved: %s\n", backupFile)
}

