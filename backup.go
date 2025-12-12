package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ExportVariablesToCSV exports GitHub variables to a CSV file
func ExportVariablesToCSV(variables []Variable, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	err = writer.Write([]string{"Key", "Value", "Note"})
	if err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write variables
	for _, v := range variables {
		err = writer.Write([]string{v.Name, v.Value, ""})
		if err != nil {
			return fmt.Errorf("failed to write variable %s: %w", v.Name, err)
		}
	}

	return nil
}

// BackupGitHubVariables creates a backup of GitHub variables to a timestamped file
func BackupGitHubVariables(token, owner, repo, environment string) (string, error) {
	// Fetch current GitHub variables
	variables, err := FetchGitHubVariables(token, owner, repo, environment)
	if err != nil {
		return "", fmt.Errorf("failed to fetch variables: %w", err)
	}

	// Create backup directory if it doesn't exist
	backupDir := "backups"
	err = os.MkdirAll(backupDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate timestamped filename
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	var filename string
	if environment != "" {
		filename = filepath.Join(backupDir, fmt.Sprintf("backup_%s_%s_%s_%s.csv", owner, repo, environment, timestamp))
	} else {
		filename = filepath.Join(backupDir, fmt.Sprintf("backup_%s_%s_%s.csv", owner, repo, timestamp))
	}

	// Export to CSV
	err = ExportVariablesToCSV(variables, filename)
	if err != nil {
		return "", fmt.Errorf("failed to export backup: %w", err)
	}

	return filename, nil
}

