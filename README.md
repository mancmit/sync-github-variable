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

> **Note**: The tool will display a confirmation screen before syncing. You must type `yes` or `y` to proceed.

### Option 1: Compile and run

```bash
# Build
go build -o sync-variables main.go

# Run
./sync-variables
```

### Option 2: Run directly

```bash
go run main.go
```

### Option 3: Run with inline env vars

**Repository-level:**
```bash
GITHUB_TOKEN="ghp_xxx" GITHUB_OWNER="owner" GITHUB_REPO="repo" go run main.go
```

**Environment-specific:**
```bash
GITHUB_TOKEN="ghp_xxx" GITHUB_OWNER="owner" GITHUB_REPO="repo" GITHUB_ENVIRONMENT="production" go run main.go
```

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
- If variable exists → Update
- If variable doesn't exist → Create new

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

