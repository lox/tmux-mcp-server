# Tmux MCP Server

An MCP server that lets AI agents interact with terminal sessions through tmux.

## Running the Server

```bash
go run ./cmd/tmux-mcp-server
```

The server communicates via stdio and provides tools for managing tmux sessions.

## Usage

The server provides these tools:

- `start_session` - Create a new tmux session
- `send_commands` - Send commands and keystrokes to a session
- `view_session` - Capture the current screen content
- `list_sessions` - Show all active sessions
- `join_session` - Join an existing session
- `close_session` - End a session
- `git_add_patch` - Start interactive git staging (git add -p)
- `git_add_patch_respond` - Send responses to git add -p prompts

### Example: Editing a file with vim

```json
{
  "name": "start_session",
  "arguments": {
    "session_name": "edit_work",
    "command": "vim README.md"
  }
}
```

```json
{
  "name": "send_commands",
  "arguments": {
    "session_name": "edit_work",
    "commands": [
      "i",
      "Hello world!",
      "<ESC>",
      ":wq",
      "<ENTER>"
    ]
  }
}
```

The `send_commands` tool takes an array where plain strings are typed literally and `<COMMAND>` format handles special keys like `<ENTER>`, `<ESC>`, `<TAB>`, etc.

### Example: Interactive git staging

```json
{
  "name": "git_add_patch",
  "arguments": {
    "working_directory": "/path/to/repo",
    "args": ["*.js", "README.md"]
  }
}
```

```json
{
  "name": "git_add_patch_respond",
  "arguments": {
    "session_id": "git-add-patch-1234567890",
    "response": "y"
  }
}
```

To quit early, send 'q':
```json
{
  "name": "git_add_patch_respond", 
  "arguments": {
    "session_id": "git-add-patch-1234567890",
    "response": "q"
  }
}
```

## Development

This project uses [Hermit](https://cashapp.github.io/hermit/) for managing development dependencies. Hermit ensures consistent development environments across different machines.

```
. bin/activate-hermit
```

## Requirements

- Go 1.24.2+
- tmux
