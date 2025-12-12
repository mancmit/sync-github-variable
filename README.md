# Sync GitHub Variables from CSV

Simple tool to sync repository variables to GitHub from a CSV file.

## Setup

### 1. Create GitHub Personal Access Token

1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Select scope: **`repo`** (Full control of private repositories)
4. Or if using fine-grained token: select **Variables** (Read and write) permission
5. Copy the token (only shown once)

### 2. Set Environment Variables

**For Repository-level variables:**
```bash
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxx"
export GITHUB_OWNER="your-owner-or-organization"
export GITHUB_REPO="your-repository"
```

**For Environment-specific variables:**
```bash
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxx"
export GITHUB_OWNER="your-owner-or-organization"
export GITHUB_REPO="your-repository"
export GITHUB_ENVIRONMENT="production"  # or staging, development, etc.
```

## Usage

> **Note**: The tool now includes Diff Mode to compare local and remote variables before syncing.

### Command-line Options

- `--diff` - Show differences between local CSV and GitHub variables, then exit without syncing
- `--backup` - Create a backup of GitHub variables, then exit
- `--no-backup` - Skip automatic backup before syncing (backup is enabled by default)
- No flags - Show diff, auto-backup, ask for confirmation, then sync only changed variables

### Option 1: Backup Mode - Create manual backup

```bash
# Create backup of GitHub variables
./sync-variables --backup
```

This will:
- Fetch all current variables from GitHub
- Save to timestamped backup file in `backups/` directory
- Exit without syncing

### Option 2: View diff only (no sync)

```bash
# Build
go build -o sync-variables

# View differences without syncing
./sync-variables --diff
```

This will:
- Fetch current variables from GitHub
- Compare with your local CSV file
- Display summary and detailed diff
- Exit without making any changes

### Option 3: Normal sync with auto-backup

```bash
# Build
go build -o sync-variables

# Sync with diff preview and auto-backup (default behavior)
./sync-variables

# Sync without auto-backup
./sync-variables --no-backup
```

This will:
- Show diff summary and details
- Ask for confirmation
- **Automatically create a backup before syncing** (unless `--no-backup` is used)
- Sync only new and updated variables (skip unchanged)
- Display results with counts

### Option 4: Run directly (without building)

```bash
# Backup mode
go run . --backup

# View diff only
go run . --diff

# Normal sync with auto-backup
go run .

# Sync without backup
go run . --no-backup
```

**Note**: Use `go run .` to compile all Go files in the package. Running `go run main.go` will fail because it won't include `diff.go` and `backup.go`.

### Option 5: Run with inline env vars

**Repository-level:**
```bash
# Backup
GITHUB_TOKEN="ghp_xxx" GITHUB_OWNER="owner" GITHUB_REPO="repo" go run . --backup

# View diff
GITHUB_TOKEN="ghp_xxx" GITHUB_OWNER="owner" GITHUB_REPO="repo" go run . --diff

# Sync with auto-backup
GITHUB_TOKEN="ghp_xxx" GITHUB_OWNER="owner" GITHUB_REPO="repo" go run .

# Sync without backup
GITHUB_TOKEN="ghp_xxx" GITHUB_OWNER="owner" GITHUB_REPO="repo" go run . --no-backup
```

**Environment-specific:**
```bash
# Backup
GITHUB_TOKEN="ghp_xxx" GITHUB_OWNER="owner" GITHUB_REPO="repo" GITHUB_ENVIRONMENT="production" go run . --backup

# View diff
GITHUB_TOKEN="ghp_xxx" GITHUB_OWNER="owner" GITHUB_REPO="repo" GITHUB_ENVIRONMENT="production" go run . --diff

# Sync with auto-backup
GITHUB_TOKEN="ghp_xxx" GITHUB_OWNER="owner" GITHUB_REPO="repo" GITHUB_ENVIRONMENT="production" go run .
```

## Backup Features

The tool includes comprehensive backup functionality:

**Automatic Backup (Default):**
- Automatically creates a backup before every sync operation
- Can be disabled with `--no-backup` flag
- Asks for confirmation if backup fails

**Manual Backup:**
- Use `--backup` flag to create a backup without syncing
- Useful for scheduled backups or before manual changes

**Backup Storage:**
- All backups saved to `backups/` directory
- Timestamped filenames: `backup_OWNER_REPO_[ENV_]TIMESTAMP.csv`

**Backup Use Cases:**
- Regular backups before sync operations
- Disaster recovery and rollback capability
- Audit trail of variable changes over time
- Safe testing of variable modifications
- Export current state for documentation or sharing (use backup CSV files)

## Diff Mode Feature

The tool now includes a powerful diff feature that compares your local CSV with GitHub variables:

### Diff Display

The diff shows:
- ✨ **New** - Variables in CSV but not in GitHub (will be created)
- 🔄 **Updated** - Variables with different values (will be updated)
- ✅ **Unchanged** - Variables with same values (skipped during sync)
- ⚠️ **Deleted** - Variables in GitHub but not in CSV (informational only, not deleted)

### Color-coded Output

- 🟢 Green - New variables
- 🟡 Yellow - Updated variables (shows old → new)
- ⚪ Gray - Unchanged variables
- 🔴 Red - Deleted variables (shown but not actually deleted)

### Smart Sync

The tool only syncs variables that need changes:
- Skips unchanged variables entirely
- Faster execution with fewer API calls
- Saves GitHub API rate limits
- Clear output showing exactly what was created/updated
- Tracks and reports failed syncs separately


## CSV File Format

The `variables.csv` file should have the following format:

```csv
Key,Value,Note
Key1,Val1,
Key2,Val2,
Key3,Val3,Optional note
```

- Column 1: Variable name
- Column 2: Variable value
- Column 3: Note (not used, just for reference)

## Notes

- This tool creates/updates **variables** (not secrets)
- Automatically detects changes and only syncs what's needed:
  - If variable doesn't exist → Create new
  - If variable exists with different value → Update
  - If variable exists with same value → Skip (no API call)
- Variables in GitHub but not in CSV are shown but NOT deleted

### Where to view variables:

**Repository-level variables:**
- Repository → Settings → Secrets and variables → Actions → Variables tab

**Environment-specific variables:**
- Repository → Settings → Environments → Select environment → Variables section

### Differences:

| Type | Scope | Use Case |
|------|-------|----------|
| **Repository** | All workflows | General variables used across all environments |
| **Environment** | Specific environment only | Environment-specific values (prod/staging/dev) |

### Example Use Cases:

- **Repository variables**: API endpoints, feature flags, general configs
- **Environment variables**: Database URLs, API keys per environment, deployment targets

## Example Output

### Backup Mode Output

```
🎯 Target: Repository owner/repo
💾 Backup Mode: Creating backup of GitHub variables...
✅ Backup saved: backups/backup_owner_repo_2024-12-12_14-30-45.csv
```

### Diff Mode Output

```
📝 Read 10 variables from CSV file
🔍 Fetching current variables from GitHub...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 DIFF SUMMARY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✨ New:       3 variable(s)
🔄 Updated:   2 variable(s)
✅ Unchanged: 5 variable(s)
⚠️  Deleted:   1 variable(s) (in GitHub, not in CSV)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📝 DETAILED CHANGES:

[NEW VARIABLES]
+ API_ENDPOINT = https://api.example.com
+ FEATURE_FLAG = enabled
+ DEBUG_MODE = false

[UPDATED VARIABLES]
~ DATABASE_URL:
  - postgres://old-host:5432/db
  + postgres://new-host:5432/db
~ API_KEY:
  - old_key_value
  + new_key_value

[UNCHANGED]
5 variable(s) with no changes

[DELETED - in GitHub but not in CSV]
Note: These will NOT be deleted from GitHub
- OLD_CONFIG = some_value

ℹ️  Diff mode: No changes were made
```

### Normal Sync Output (with Auto-Backup)

```
[... diff output as above ...]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 SYNC CONFIGURATION
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Repository:  myorg/myrepo
Environment: (none)
Target:      Repository-level variables
Token:       ghp_****xxxx
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📦 Will sync 5 variable(s) (3 new, 2 updated)

⚠️  Do you want to proceed with the sync? (yes/no): yes

💾 Creating backup before sync...
✅ Backup saved: backups/backup_myorg_myrepo_2024-12-12_14-35-20.csv

🚀 Starting sync...

✅ Created variable: API_ENDPOINT
✅ Created variable: FEATURE_FLAG
✅ Created variable: DEBUG_MODE
✅ Updated variable: DATABASE_URL
✅ Updated variable: API_KEY

🎉 Completed! Created 3, Updated 2, Total 5 variables
```

