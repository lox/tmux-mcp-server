package main

import (
	"fmt"
	"os"

	"github.com/lox/tmux-mcp-server/internal/server"
)

func main() {
	// Parse command line arguments
	config := server.ParseArgs(os.Args[1:])

	// Create and configure the server
	s, err := server.NewServer(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create server: %v\n", err)
		os.Exit(1)
	}

	// Start the server
	if err := server.Serve(s); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
