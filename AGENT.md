# AGENT.md - Development Guide

## Build/Test Commands
- `task build` - Build the server binary to dist/tmux-mcp-server
- `task test` - Run unit tests (./internal/server/..., ./internal/client/..., ./internal/tmux/...)
- `task test-all` - Run all tests (unit + integration)
- `task integration-test` - Run integration tests (./internal/testing/...)
- `go test -v ./internal/server/...` - Run tests for specific package
- `task fmt` - Format code with go fmt and goimports
- `task lint` - Lint code (requires golangci-lint)
- `task run-http` - Run server in HTTP mode on port 8080
- `task run-stdio` - Run server in stdio mode

## Architecture
- **cmd/tmux-mcp-server/main.go** - Main entry point for MCP server
- **internal/server/** - Server utilities and MCP tool handlers
- **internal/tmux/** - Tmux session management wrapper functions
- **internal/client/** - Client implementation
- Uses MCP (Model Context Protocol) with tmux for terminal session management
- Server modes: stdio (default) or HTTP with --http flag

## Code Style
- Error handling: Use `fmt.Errorf` for wrapping, `mcp.NewToolResultError()` for user errors
- Context first parameter in functions accepting `context.Context`
- Factory functions prefixed with `New...` for initialization
- Mutex usage: `sync.RWMutex` for reads, `sync.Mutex` for writes
- Always check session existence before operations
