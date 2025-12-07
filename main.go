package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	githubAPIURL = "https://api.github.com"
)

type Variable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func main() {
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

	// Read CSV file
	variables, err := readCSV("variables.csv")
	if err != nil {
		fmt.Printf("❌ Error reading CSV file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📝 Read %d variables from CSV file\n\n", len(variables))

	// Show confirmation before syncing
	if !confirmSync(owner, repo, environment, token, variables) {
		fmt.Println("\n❌ Sync cancelled by user")
		os.Exit(0)
	}

	fmt.Println("\n🚀 Starting sync...\n")

	// Sync each variable to GitHub
	successCount := 0
	for _, variable := range variables {
		if variable.Name == "" {
			continue
		}

		err := syncVariable(token, owner, repo, environment, variable)
		if err != nil {
			fmt.Printf("❌ Error syncing variable '%s': %v\n", variable.Name, err)
		} else {
			fmt.Printf("✅ Synced variable: %s\n", variable.Name)
			successCount++
		}
	}

	fmt.Printf("\n🎉 Completed! Synced %d/%d variables\n", successCount, len(variables))
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

func confirmSync(owner, repo, environment, token string, variables []Variable) bool {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
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
	
	// Display variables to be synced
	fmt.Printf("\n📦 Variables to sync (%d):\n", len(variables))
	for i, v := range variables {
		if v.Name == "" {
			continue
		}
		valuePreview := v.Value
		if len(valuePreview) > 50 {
			valuePreview = valuePreview[:47] + "..."
		}
		fmt.Printf("  %2d. %s = %s\n", i+1, v.Name, valuePreview)
	}
	
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

	client := &http.Client{}
	resp, err := client.Do(req)
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

	client := &http.Client{}
	resp, err := client.Do(req)
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

	client := &http.Client{}
	resp, err := client.Do(req)
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

