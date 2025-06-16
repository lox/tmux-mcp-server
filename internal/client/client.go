package client

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// Client provides a high-level interface for interacting with the TTY MCP server
type Client struct {
	mcpClient *client.Client
}

// NewStdioClient creates a new stdio client for the TTY MCP server
func NewStdioClient(command string, args ...string) (*Client, error) {
	mcpClient, err := client.NewStdioMCPClient(command, nil, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create stdio client: %v", err)
	}

	return &Client{
		mcpClient: mcpClient,
	}, nil
}

// Initialize initializes the MCP client
func (c *Client) Initialize(ctx context.Context) error {
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = "2025-03-26"
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "tty-test-client",
		Version: "1.0.0",
	}

	err := c.mcpClient.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start client: %v", err)
	}

	_, err = c.mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		return fmt.Errorf("failed to initialize: %v", err)
	}

	return nil
}

// ListTools lists available tools
func (c *Client) ListTools(ctx context.Context) (*mcp.ListToolsResult, error) {
	request := mcp.ListToolsRequest{}
	return c.mcpClient.ListTools(ctx, request)
}

// StartSession starts a new tmux session
func (c *Client) StartSession(ctx context.Context, sessionName, command, workingDir string) (*mcp.CallToolResult, error) {
	request := mcp.CallToolRequest{}
	request.Params.Name = "start_session"
	request.Params.Arguments = map[string]interface{}{
		"session_name": sessionName,
	}

	args := request.Params.Arguments.(map[string]interface{})
	if command != "" {
		args["command"] = command
	}

	if workingDir != "" {
		args["working_directory"] = workingDir
	}

	return c.mcpClient.CallTool(ctx, request)
}

// SendKeys sends keystrokes to a session
func (c *Client) SendKeys(ctx context.Context, sessionName, keys string) (*mcp.CallToolResult, error) {
	request := mcp.CallToolRequest{}
	request.Params.Name = "send_keys"
	request.Params.Arguments = map[string]interface{}{
		"session_name": sessionName,
		"keys":         keys,
	}

	return c.mcpClient.CallTool(ctx, request)
}

// ViewSession captures the current screen of a session
func (c *Client) ViewSession(ctx context.Context, sessionName string) (*mcp.CallToolResult, error) {
	request := mcp.CallToolRequest{}
	request.Params.Name = "view_session"
	request.Params.Arguments = map[string]interface{}{
		"session_name": sessionName,
	}

	return c.mcpClient.CallTool(ctx, request)
}

// ListSessions lists all active sessions
func (c *Client) ListSessions(ctx context.Context) (*mcp.CallToolResult, error) {
	request := mcp.CallToolRequest{}
	request.Params.Name = "list_sessions"
	request.Params.Arguments = map[string]interface{}{}

	return c.mcpClient.CallTool(ctx, request)
}

// CloseSession closes a session
func (c *Client) CloseSession(ctx context.Context, sessionName string) (*mcp.CallToolResult, error) {
	request := mcp.CallToolRequest{}
	request.Params.Name = "close_session"
	request.Params.Arguments = map[string]interface{}{
		"session_name": sessionName,
	}

	return c.mcpClient.CallTool(ctx, request)
}

// Close closes the client
func (c *Client) Close() error {
	return c.mcpClient.Close()
}

// GetToolResultText extracts text content from a tool result
func GetToolResultText(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}

	for _, content := range result.Content {
		if textContent, ok := mcp.AsTextContent(content); ok {
			return textContent.Text
		}
	}

	return ""
}
