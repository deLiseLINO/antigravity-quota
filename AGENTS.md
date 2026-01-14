# AGENTS.md

This file provides guidance for AI coding agents working with this repository.

## Project Overview

AGQ (Antigravity Quota Monitor) is a TUI application for monitoring Antigravity account quotas. Built with Go using the Bubble Tea framework (Elm architecture).

**Tech Stack:**
- Go 1.25
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [Bubbles](https://github.com/charmbracelet/bubbles) - UI components

## Build & Run Commands

```bash
# Build binary
go build -o agq ./cmd/agq

# Install to $GOPATH/bin
go install ./cmd/agq

# Run directly
go run ./cmd/agq

# Clean
rm agq

# Format code
go fmt ./...

# Lint
go vet ./...

# Run all tests (currently no tests exist)
go test ./...

# Run tests in a specific package
go test ./internal/auth

# Run a single test
go test ./internal/auth -run TestLoginFlow

# Tidy dependencies
go mod tidy
```

**Workflow:** Always run `go install ./cmd/agq` after making changes to test locally.

## Project Structure

```
cmd/agq/main.go           - Entry point
internal/
  ├── api/client.go       - Antigravity API client
  ├── auth/
  │   ├── oauth.go        - OAuth 2.0 flow
  │   └── token.go        - Token refresh & user info
  ├── config/tokens.go    - Token file management
  └── ui/
      ├── model.go        - Bubble Tea model & update logic
      ├── view.go         - UI rendering
      ├── messages.go     - Message types
      └── styles.go       - Lipgloss styles
```

**Token Storage:** `~/.cli-proxy-api/antigravity_token_<email>.json`

## Code Style Guidelines

### Imports

Group imports in this order (with blank lines between groups):
1. Standard library
2. Third-party packages
3. Internal packages

```go
import (
    "fmt"
    "os"
    
    tea "github.com/charmbracelet/bubbletea"
    
    "github.com/deLiseLINO/antigravity-quota/internal/auth"
    "github.com/deLiseLINO/antigravity-quota/internal/config"
)
```

### Formatting

- **Tabs for indentation** (gofmt default)
- **Line length:** No hard limit, but keep it readable (~100-120 chars)
- **Use gofmt:** All code must be formatted with `go fmt`

### Naming Conventions

- **Packages:** Short, lowercase, single word (e.g., `auth`, `config`, `ui`)
- **Exported types:** PascalCase (e.g., `TokenFile`, `AllModelsData`)
- **Unexported types:** camelCase (e.g., `modelQuota`, `authState`)
- **Constants:** camelCase for unexported, PascalCase for exported (e.g., `googleTokenURL`, `ModeGroups`)
- **Functions:** PascalCase for exported, camelCase for unexported

### Types

- **Prefer structs over maps** for structured data
- **Use struct tags** for JSON marshaling: `json:"field_name"`
- **Embed types** when appropriate (e.g., `progress.Model` in `ui.Model`)

Example:
```go
type TokenFile struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    Expired      string `json:"expired"`
    Email        string `json:"email"`
    FilePath     string `json:"-"` // Not serialized
}
```

### Error Handling

- **Always check errors** - never ignore with `_`
- **Wrap errors with context** using `fmt.Errorf("context: %w", err)`
- **Return errors, don't panic** (except in truly unrecoverable situations)
- **Use sentinel errors** for specific conditions (if needed)

Example:
```go
if err != nil {
    return nil, fmt.Errorf("failed to create request: %w", err)
}
```

### Comments

- **Package comments:** Every package should have a doc comment
- **Exported symbols:** Must have doc comments starting with the symbol name
- **Implementation notes:** Use `//` for inline comments

Example:
```go
// CallAPI fetches quota data from the Antigravity API using the provided token.
// It automatically handles token refresh if expired.
func CallAPI(token string) (AllModelsData, error) {
    // ...
}
```

## Architecture Notes

### Bubble Tea Pattern

The app follows the Elm architecture with three core functions:

1. **Init()** - Initial command (fetch data on startup)
2. **Update(msg)** - Handle messages and update state
3. **View()** - Render current state to string

**State Management:**
- All state lives in `ui.Model`
- State updates are immutable (return new model)
- Side effects are handled via `tea.Cmd`

### OAuth Flow

1. Start local HTTP server on random port
2. Open browser for Google OAuth 2.0
3. Exchange code for access/refresh tokens
4. Fetch user email for token file naming
5. Save to `~/.cli-proxy-api/antigravity_token_<email>.json`

**Important:** `fmt.Println` must not be used during TUI operation - it corrupts the display. Only use before `tea.NewProgram()` starts.

### OAuth Credentials

The OAuth credentials in `internal/auth/oauth.go` are **intentionally public** and shared across the CLIProxyAPI ecosystem. Do not treat them as secrets - they're the same credentials used by CLIProxyAPI, ProxyPal, CCS, and VibeProxy.

## Known Issues

- **Delete Confirmation UI:** May occasionally fail to clear the deletion confirmation dialog immediately after successful deletion due to state update timing.

## Testing

Currently no tests exist. When adding tests:
- Place test files next to the code they test
- Name test files `*_test.go`
- Use table-driven tests for multiple cases
- Mock external dependencies (HTTP, filesystem)

## DO NOT

- ❌ Use `fmt.Println` or `log.Print` during TUI operation
- ❌ Ignore errors with `_`
- ❌ Commit binaries (`agq` is gitignored)
- ❌ Hardcode file paths (use `os.UserHomeDir()` + `filepath.Join()`)
- ❌ Create your own OAuth credentials (use the shared ones)

## Best Practices

- ✅ Run `go fmt ./...` before committing
- ✅ Run `go vet ./...` to catch common mistakes
- ✅ Keep functions small and focused
- ✅ Use constants for magic strings/numbers
- ✅ Handle token expiration gracefully (auto-refresh)
- ✅ Use context for HTTP requests with timeouts
