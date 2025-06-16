package testing

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/lox/tmux-mcp-server/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTTYServerIntegration(t *testing.T) {
	// Skip if tmux is not available
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available, skipping integration test")
	}

	// Build the server binary for testing
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err, "Failed to get absolute project root")
	
	serverBinary := filepath.Join(projectRoot, "test-tmux-mcp-server")

	// Build the binary
	cmd := exec.Command("go", "build", "-o", serverBinary, "./cmd/tmux-mcp-server")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to build server: %s", string(output))
	defer func() { _ = os.Remove(serverBinary) }()
	
	// Verify binary exists and is executable
	info, err := os.Stat(serverBinary)
	require.NoError(t, err, "Binary should exist at %s", serverBinary)
	require.False(t, info.IsDir(), "Binary path should not be a directory")
	t.Logf("Built server binary at: %s", serverBinary)

	t.Run("TestToolsList", func(t *testing.T) {
		// Create client that spawns the server
		mcpClient, err := client.NewStdioClient(serverBinary)
		require.NoError(t, err, "Failed to create client")
		defer func() { _ = mcpClient.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Initialize the client
		err = mcpClient.Initialize(ctx)
		require.NoError(t, err, "Failed to initialize client")

		// List tools
		tools, err := mcpClient.ListTools(ctx)
		require.NoError(t, err, "Failed to list tools")
		require.NotNil(t, tools, "Tools list should not be nil")

		// Expected tools
		expectedTools := []string{
			"start_session",
			"send_keys",
			"send_commands",
			"view_session",
			"list_sessions",
			"join_session",
			"close_session",
		}

		toolNames := make([]string, len(tools.Tools))
		for i, tool := range tools.Tools {
			toolNames[i] = tool.Name
		}

		for _, expectedTool := range expectedTools {
			assert.Contains(t, toolNames, expectedTool, "Expected tool %s to be available", expectedTool)
		}
	})

	t.Run("TestSessionLifecycle", func(t *testing.T) {
		// Create client that spawns the server
		mcpClient, err := client.NewStdioClient(serverBinary)
		require.NoError(t, err, "Failed to create client")
		defer func() { _ = mcpClient.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Initialize the client
		err = mcpClient.Initialize(ctx)
		require.NoError(t, err, "Failed to initialize client")

		// Start a session with a shell (no command so it stays alive)
		sessionName := "test_session_lifecycle"
		startResult, err := mcpClient.StartSession(ctx, sessionName, "", "")
		require.NoError(t, err, "Failed to start session")
		require.NotNil(t, startResult, "Start session result should not be nil")

		startText := client.GetToolResultText(startResult)
		assert.Contains(t, startText, "started successfully", "Expected session start confirmation")

		// Give the command a moment to run
		time.Sleep(500 * time.Millisecond)

		// View the session
		viewResult, err := mcpClient.ViewSession(ctx, sessionName)
		require.NoError(t, err, "Failed to view session")
		require.NotNil(t, viewResult, "View session result should not be nil")

		viewText := client.GetToolResultText(viewResult)
		t.Logf("Session content: %s", viewText)

		// List sessions
		listResult, err := mcpClient.ListSessions(ctx)
		require.NoError(t, err, "Failed to list sessions")
		require.NotNil(t, listResult, "List sessions result should not be nil")

		listText := client.GetToolResultText(listResult)
		assert.Contains(t, listText, sessionName, "Expected session name in list")

		// Close the session
		closeResult, err := mcpClient.CloseSession(ctx, sessionName)
		require.NoError(t, err, "Failed to close session")
		require.NotNil(t, closeResult, "Close session result should not be nil")

		closeText := client.GetToolResultText(closeResult)
		assert.Contains(t, closeText, "closed successfully", "Expected session close confirmation")
	})

	t.Run("TestSendCommands", func(t *testing.T) {
		// Create client that spawns the server
		mcpClient, err := client.NewStdioClient(serverBinary)
		require.NoError(t, err, "Failed to create client")
		defer func() { _ = mcpClient.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Initialize the client
		err = mcpClient.Initialize(ctx)
		require.NoError(t, err, "Failed to initialize client")

		// Start a shell session
		sessionName := "test_send_commands"
		startResult, err := mcpClient.StartSession(ctx, sessionName, "", "")
		require.NoError(t, err, "Failed to start session")
		require.NotNil(t, startResult, "Start session result should not be nil")

		startText := client.GetToolResultText(startResult)
		assert.Contains(t, startText, "started successfully", "Expected session to start")

		// Test sending keys to the session
		result, err := mcpClient.SendKeys(ctx, sessionName, "echo 'Hello from test'")
		require.NoError(t, err, "Failed to send keys")
		require.NotNil(t, result, "Send keys result should not be nil")

		// Send Enter to execute the command
		_, err = mcpClient.SendKeys(ctx, sessionName, "Enter")
		require.NoError(t, err, "Failed to send Enter")

		// Wait for command to execute
		time.Sleep(500 * time.Millisecond)

		// View the session to see the output
		viewResult, err := mcpClient.ViewSession(ctx, sessionName)
		require.NoError(t, err, "Failed to view session")
		viewText := client.GetToolResultText(viewResult)
		t.Logf("Session after echo: %s", viewText)

		// Close the session
		closeResult, err := mcpClient.CloseSession(ctx, sessionName)
		require.NoError(t, err, "Failed to close session")
		require.NotNil(t, closeResult, "Close session result should not be nil")

		closeText := client.GetToolResultText(closeResult)
		assert.Contains(t, closeText, "closed successfully", "Expected session close confirmation")
	})
}
