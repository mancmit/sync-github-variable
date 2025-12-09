package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ANSI color codes for terminal output
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorGray   = "\033[90m"
	ColorBold   = "\033[1m"
)

// DiffResult represents the differences between local and remote variables
type DiffResult struct {
	New       []Variable       // Variables in CSV but not in GitHub (will be created)
	Updated   []VariableChange // Variables that exist but values differ (will be updated)
	Unchanged []Variable       // Variables with same values (no action)
	Deleted   []Variable       // Variables in GitHub but not in CSV (informational only)
}

// VariableChange represents a variable that will be updated
type VariableChange struct {
	Name     string
	OldValue string // Current value in GitHub
	NewValue string // New value from CSV
}

// GitHubVariablesResponse represents the GitHub API response for listing variables
type GitHubVariablesResponse struct {
	TotalCount int        `json:"total_count"`
	Variables  []Variable `json:"variables"`
}

// FetchGitHubVariables fetches all current variables from GitHub with pagination support
// GitHub API returns max 30 items by default, 100 max per page
func FetchGitHubVariables(token, owner, repo, environment string) ([]Variable, error) {
	var baseURL string
	if environment != "" {
		// Environment-specific variable
		baseURL = fmt.Sprintf("%s/repos/%s/%s/environments/%s/variables", githubAPIURL, owner, repo, environment)
	} else {
		// Repository-level variable
		baseURL = fmt.Sprintf("%s/repos/%s/%s/actions/variables", githubAPIURL, owner, repo)
	}

	allVariables := []Variable{}
	page := 1
	perPage := 100 // Maximum allowed by GitHub API

	for {
		url := fmt.Sprintf("%s?per_page=%d&page=%d", baseURL, perPage, page)
		
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		var response GitHubVariablesResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, err
		}

		// Add variables from this page
		allVariables = append(allVariables, response.Variables...)

		// Check if we've fetched all variables
		// Break if: no more variables OR we've fetched all (total_count)
		if len(response.Variables) == 0 || len(allVariables) >= response.TotalCount {
			break
		}

		page++
	}

	return allVariables, nil
}

// CompareSets compares local CSV variables with remote GitHub variables
func CompareSets(local, remote []Variable) DiffResult {
	result := DiffResult{
		New:       []Variable{},
		Updated:   []VariableChange{},
		Unchanged: []Variable{},
		Deleted:   []Variable{},
	}

	// Create a map of remote variables for quick lookup
	remoteMap := make(map[string]string)
	for _, v := range remote {
		remoteMap[v.Name] = v.Value
	}

	// Check each local variable
	for _, localVar := range local {
		if localVar.Name == "" {
			continue
		}

		remoteValue, exists := remoteMap[localVar.Name]
		if !exists {
			// Variable doesn't exist in GitHub - will be created
			result.New = append(result.New, localVar)
		} else if remoteValue != localVar.Value {
			// Variable exists but value is different - will be updated
			result.Updated = append(result.Updated, VariableChange{
				Name:     localVar.Name,
				OldValue: remoteValue,
				NewValue: localVar.Value,
			})
		} else {
			// Variable exists with same value - no action needed
			result.Unchanged = append(result.Unchanged, localVar)
		}
	}

	// Create a map of local variables for checking deleted ones
	localMap := make(map[string]bool)
	for _, v := range local {
		if v.Name != "" {
			localMap[v.Name] = true
		}
	}

	// Find variables in GitHub but not in CSV
	for _, remoteVar := range remote {
		if !localMap[remoteVar.Name] {
			result.Deleted = append(result.Deleted, remoteVar)
		}
	}

	return result
}

// DisplayDiffSummary displays a summary table of the diff
func DisplayDiffSummary(diff DiffResult) {
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 DIFF SUMMARY")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	fmt.Printf("%s✨ New:%s       %d variable(s)\n", ColorGreen, ColorReset, len(diff.New))
	fmt.Printf("%s🔄 Updated:%s   %d variable(s)\n", ColorYellow, ColorReset, len(diff.Updated))
	fmt.Printf("%s✅ Unchanged:%s %d variable(s)\n", ColorGray, ColorReset, len(diff.Unchanged))
	
	if len(diff.Deleted) > 0 {
		fmt.Printf("%s⚠️  Deleted:%s   %d variable(s) (in GitHub, not in CSV)\n", ColorRed, ColorReset, len(diff.Deleted))
	}
	
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// DisplayDetailedDiff displays detailed line-by-line diff
func DisplayDetailedDiff(diff DiffResult) {
	fmt.Println("\n📝 DETAILED CHANGES:")
	fmt.Println()

	// Display new variables
	if len(diff.New) > 0 {
		fmt.Printf("%s[NEW VARIABLES]%s\n", ColorGreen+ColorBold, ColorReset)
		for _, v := range diff.New {
			value := truncateValue(v.Value, 80)
			fmt.Printf("%s+ %s = %s%s\n", ColorGreen, v.Name, value, ColorReset)
		}
		fmt.Println()
	}

	// Display updated variables
	if len(diff.Updated) > 0 {
		fmt.Printf("%s[UPDATED VARIABLES]%s\n", ColorYellow+ColorBold, ColorReset)
		for _, change := range diff.Updated {
			oldValue := truncateValue(change.OldValue, 60)
			newValue := truncateValue(change.NewValue, 60)
			fmt.Printf("%s~ %s:%s\n", ColorYellow, change.Name, ColorReset)
			fmt.Printf("  %s- %s%s\n", ColorRed, oldValue, ColorReset)
			fmt.Printf("  %s+ %s%s\n", ColorGreen, newValue, ColorReset)
		}
		fmt.Println()
	}

	// Display unchanged count (don't list all of them)
	if len(diff.Unchanged) > 0 {
		fmt.Printf("%s[UNCHANGED]%s\n", ColorGray, ColorReset)
		fmt.Printf("%s%d variable(s) with no changes%s\n", ColorGray, len(diff.Unchanged), ColorReset)
		fmt.Println()
	}

	// Display deleted variables (informational)
	if len(diff.Deleted) > 0 {
		fmt.Printf("%s[DELETED - in GitHub but not in CSV]%s\n", ColorRed+ColorBold, ColorReset)
		fmt.Printf("%sNote: These will NOT be deleted from GitHub%s\n", ColorGray, ColorReset)
		for _, v := range diff.Deleted {
			value := truncateValue(v.Value, 80)
			fmt.Printf("%s- %s = %s%s\n", ColorRed, v.Name, value, ColorReset)
		}
		fmt.Println()
	}
}

// truncateValue truncates a string to maxLen characters with ellipsis
func truncateValue(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen-3] + "..."
}

