package server

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lox/tmux-mcp-server/internal/tmux"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// gitOperation tracks an active git operation
type gitOperation struct {
	SessionName    string
	Command        string
	StartTime      time.Time
	WorkingDir     string
	Status         string // "active", "finished", "error"
}

var (
	gitOperations = make(map[string]*gitOperation)
	gitOpsMutex   sync.RWMutex
)

// registerGitTools adds interactive git operation tools to the MCP server
func registerGitTools(s *server.MCPServer) error {
	// git_add_patch tool
	gitAddPatchTool := mcp.NewTool("git_add_patch",
		mcp.WithDescription("Start interactive git staging (git add -p) and return operation ID"),
		mcp.WithString("working_directory",
			mcp.Description("Working directory for the git operation (default: current directory)"),
		),
		mcp.WithArray("args",
			mcp.Description("Additional arguments for git add -p (e.g., [\"file1.txt\", \"*.js\"])"),
		),
	)
	s.AddTool(gitAddPatchTool, gitAddPatchHandler)

	// git_add_patch_respond tool
	gitAddPatchRespondTool := mcp.NewTool("git_add_patch_respond",
		mcp.WithDescription("Send response to interactive git add -p operation"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("Operation ID returned from git_add_patch"),
		),
		mcp.WithString("response",
			mcp.Required(),
			mcp.Description("Response to send: 'y' (yes), 'n' (no), 's' (split), 'q' (quit), 'a' (all), 'd' (done), '?' (help)"),
		),
	)
	s.AddTool(gitAddPatchRespondTool, gitAddPatchRespondHandler)



	return nil
}

func generateSessionID() string {
	return fmt.Sprintf("git-add-patch-%d", time.Now().UnixNano())
}

func gitAddPatchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workingDir := request.GetString("working_directory", "")
	argsSlice := request.GetStringSlice("args", []string{})

	// Generate unique session ID
	sessionID := generateSessionID()

	// Build command
	command := "git add -p"
	if len(argsSlice) > 0 {
		command = fmt.Sprintf("git add -p %s", strings.Join(argsSlice, " "))
	}
	
	// Add exit status tracking
	commandWithStatus := fmt.Sprintf("%s; echo \"EXIT_STATUS:$?\"", command)

	// Start tmux session
	if err := tmux.StartSession(sessionID, "", workingDir); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to start session: %v", err)), nil
	}

	// Track the operation
	gitOpsMutex.Lock()
	gitOperations[sessionID] = &gitOperation{
		SessionName: sessionID,
		Command:     command,
		StartTime:   time.Now(),
		WorkingDir:  workingDir,
		Status:      "active",
	}
	gitOpsMutex.Unlock()

	// Send the git command with explicit ENTER
	if err := tmux.SendKeys(sessionID, commandWithStatus); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send git command: %v", err)), nil
	}
	
	if err := tmux.SendKeys(sessionID, "Enter"); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send ENTER: %v", err)), nil
	}

	// Wait a moment then capture screen
	time.Sleep(300 * time.Millisecond)
	result, err := tmux.CapturePane(sessionID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to capture screen: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Git add -p started with session ID: %s\n\nCurrent screen:\n%s", sessionID, result)), nil
}

func gitAddPatchRespondHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, err := request.RequireString("session_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	response, err := request.RequireString("response")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Check if operation exists
	gitOpsMutex.RLock()
	op, exists := gitOperations[sessionID]
	gitOpsMutex.RUnlock()

	if !exists {
		return mcp.NewToolResultError(fmt.Sprintf("Git operation with session ID '%s' not found", sessionID)), nil
	}

	if op.Status != "active" {
		return mcp.NewToolResultError(fmt.Sprintf("Git operation '%s' is not active (status: %s)", sessionID, op.Status)), nil
	}

	// Send the response followed by ENTER
	if err := tmux.SendKeys(sessionID, response); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send response: %v", err)), nil
	}
	
	if err := tmux.SendKeys(sessionID, "Enter"); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send ENTER: %v", err)), nil
	}

	// Wait a moment then capture screen
	time.Sleep(200 * time.Millisecond)
	content, err := tmux.CapturePane(sessionID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to capture screen: %v", err)), nil
	}

	// Check if operation finished (look for EXIT_STATUS)
	if strings.Contains(content, "EXIT_STATUS:") {
		gitOpsMutex.Lock()
		if strings.Contains(content, "EXIT_STATUS:0") {
			op.Status = "finished"
		} else {
			op.Status = "error"
		}
		gitOpsMutex.Unlock()
	}

	return mcp.NewToolResultText(fmt.Sprintf("Response '%s' sent to git add -p operation.\n\nCurrent screen:\n%s", response, content)), nil
}


