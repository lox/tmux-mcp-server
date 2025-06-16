package server

import (
	"context"
	"fmt"
	"os"

	"github.com/lox/tmux-mcp-server/internal/tmux"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Config holds server configuration
type Config struct {
	UseHTTP bool
	Port    string
}

// NewServer creates a new TTY MCP server
func NewServer(config Config) (*server.MCPServer, error) {
	// Check if tmux is available
	fmt.Fprintf(os.Stderr, "🔍 Checking tmux availability...\n")
	if err := tmux.CheckTmuxAvailable(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Tmux check failed: %v\n", err)
		return nil, fmt.Errorf("tmux check failed: %v\nPlease install tmux: brew install tmux (macOS) or apt-get install tmux (Ubuntu)", err)
	}
	fmt.Fprintf(os.Stderr, "✅ Tmux is available\n")

	// Create a new MCP server (stdio only for now)
	fmt.Fprintf(os.Stderr, "🖥️ Creating MCP server...\n")
	s := server.NewMCPServer(
		"TTY MCP Server",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)
	fmt.Fprintf(os.Stderr, "✅ MCP server created\n")

	// Register tools
	fmt.Fprintf(os.Stderr, "🛠️ Registering tools...\n")
	if err := registerTools(s); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to register tools: %v\n", err)
		return nil, fmt.Errorf("failed to register tools: %v", err)
	}
	fmt.Fprintf(os.Stderr, "✅ Tools registered successfully\n")

	return s, nil
}

func registerTools(s *server.MCPServer) error {
	// start_session tool
	startSessionTool := mcp.NewTool("start_session",
		mcp.WithDescription("Start a new terminal session using tmux"),
		mcp.WithString("session_name",
			mcp.Required(),
			mcp.Description("Name of the session to create"),
		),
		mcp.WithString("command",
			mcp.Description("Optional command to run (defaults to shell)"),
		),
		mcp.WithString("working_directory",
			mcp.Description("Working directory for the session"),
		),
	)
	s.AddTool(startSessionTool, startSessionHandler)

	// send_keys tool
	sendKeysTool := mcp.NewTool("send_keys",
		mcp.WithDescription("Send keystrokes to a terminal session"),
		mcp.WithString("session_name",
			mcp.Required(),
			mcp.Description("Name of the session"),
		),
		mcp.WithString("keys",
			mcp.Required(),
			mcp.Description("Keys to send to the session"),
		),
	)
	s.AddTool(sendKeysTool, sendKeysHandler)

	// view_session tool
	viewSessionTool := mcp.NewTool("view_session",
		mcp.WithDescription("View the current screen content of a terminal session"),
		mcp.WithString("session_name",
			mcp.Required(),
			mcp.Description("Name of the session"),
		),
	)
	s.AddTool(viewSessionTool, viewSessionHandler)

	// list_sessions tool
	listSessionsTool := mcp.NewTool("list_sessions",
		mcp.WithDescription("List all active terminal sessions"),
	)
	s.AddTool(listSessionsTool, listSessionsHandler)

	// send_commands tool (enhanced)
	sendCommandsTool := mcp.NewTool("send_commands",
		mcp.WithDescription("Send a sequence of commands and keystrokes to a terminal session"),
		mcp.WithString("session_name",
			mcp.Required(),
			mcp.Description("Name of the session"),
		),
		mcp.WithArray("commands",
			mcp.Required(),
			mcp.Description("Array of commands to execute. Literals are typed as-is, <COMMAND> are special keys/actions"),
		),
		mcp.WithNumber("default_delay_ms",
			mcp.Description("Default delay between commands in milliseconds (default: 100)"),
		),
		mcp.WithBoolean("capture_screen",
			mcp.Description("Whether to capture and return the screen content after execution (default: true)"),
		),
	)
	s.AddTool(sendCommandsTool, sendCommandsHandler)

	// join_session tool
	joinSessionTool := mcp.NewTool("join_session",
		mcp.WithDescription("Join an existing terminal session"),
		mcp.WithString("session_name",
			mcp.Required(),
			mcp.Description("Name of the existing session to join"),
		),
		mcp.WithString("new_session_name",
			mcp.Description("Name for this client's view of the session (optional)"),
		),
	)
	s.AddTool(joinSessionTool, joinSessionHandler)

	// close_session tool
	closeSessionTool := mcp.NewTool("close_session",
		mcp.WithDescription("Close a terminal session"),
		mcp.WithString("session_name",
			mcp.Required(),
			mcp.Description("Name of the session to close"),
		),
	)
	s.AddTool(closeSessionTool, closeSessionHandler)

	return nil
}

func startSessionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionName, err := request.RequireString("session_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	command := request.GetString("command", "")
	workingDir := request.GetString("working_directory", "")

	err = tmux.StartSession(sessionName, command, workingDir)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to start session: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Session '%s' started successfully", sessionName)), nil
}

func sendKeysHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionName, err := request.RequireString("session_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	keys, err := request.RequireString("keys")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	err = tmux.SendKeys(sessionName, keys)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send keys: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Keys sent to session '%s'", sessionName)), nil
}

func viewSessionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionName, err := request.RequireString("session_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	content, err := tmux.CapturePane(sessionName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to capture session: %v", err)), nil
	}

	return mcp.NewToolResultText(content), nil
}

func listSessionsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessions, err := tmux.ListSessions()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list sessions: %v", err)), nil
	}

	return mcp.NewToolResultText(sessions), nil
}

func sendCommandsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionName, err := request.RequireString("session_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	commandsSlice, err := request.RequireStringSlice("commands")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	defaultDelayMs := request.GetFloat("default_delay_ms", 100)
	captureScreen := request.GetBool("capture_screen", true)

	result, err := tmux.SendCommands(sessionName, commandsSlice, int(defaultDelayMs), captureScreen)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send commands: %v", err)), nil
	}

	return mcp.NewToolResultText(result), nil
}

func joinSessionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionName, err := request.RequireString("session_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	newSessionName := request.GetString("new_session_name", "")

	err = tmux.JoinSession(sessionName, newSessionName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to join session: %v", err)), nil
	}

	if newSessionName != "" {
		return mcp.NewToolResultText(fmt.Sprintf("Joined session '%s' as '%s'", sessionName, newSessionName)), nil
	} else {
		return mcp.NewToolResultText(fmt.Sprintf("Joined session '%s'", sessionName)), nil
	}
}

func closeSessionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionName, err := request.RequireString("session_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	err = tmux.KillSession(sessionName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to close session: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Session '%s' closed successfully", sessionName)), nil
}

// Serve starts the server
func Serve(s *server.MCPServer) error {
	return server.ServeStdio(s)
}

// ParseArgs parses command line arguments
func ParseArgs(args []string) Config {
	config := Config{
		UseHTTP: false,
		Port:    "8080",
	}

	for i, arg := range args {
		switch arg {
		case "--http":
			config.UseHTTP = true
		case "--port":
			if i+1 < len(args) {
				config.Port = args[i+1]
			}
		}
	}

	return config
}
