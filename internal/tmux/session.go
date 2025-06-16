package tmux

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CheckTmuxAvailable verifies tmux is installed and available
func CheckTmuxAvailable() error {
	if _, err := exec.LookPath("tmux"); err != nil {
		return fmt.Errorf("tmux is required but not found in PATH")
	}
	return nil
}

// StartSession creates a new session with the given name
func StartSession(sessionName, command, workingDir string) error {
	// Use tmux directly to match the expected sessionName exactly
	args := []string{"new-session", "-d", "-s", sessionName, "-x", "80", "-y", "24"}

	if workingDir != "" {
		args = append(args, "-c", workingDir)
	}

	if command != "" {
		args = append(args, command)
	}

	cmd := exec.Command("tmux", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %v", err)
	}

	// Give the command time to start
	time.Sleep(200 * time.Millisecond)

	return nil
}

// SendKeys sends keystrokes to a session by name
func SendKeys(sessionName, keys string) error {
	// Use direct exec.Command to avoid shell injection
	cmd := exec.Command("tmux", "send-keys", "-t", sessionName, keys)
	return cmd.Run()
}

// SendCommands sends a sequence of commands to a session with enhanced features
func SendCommands(sessionName string, commands []string, defaultDelayMs int, captureScreen bool) (string, error) {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("Executing %d commands on session '%s':\n", len(commands), sessionName))

	for i, command := range commands {
		// Check if it's a special command
		if strings.HasPrefix(command, "<") && strings.HasSuffix(command, ">") {
			err := executeSpecialCommand(sessionName, command)
			if err != nil {
				return "", fmt.Errorf("failed to execute command %d ('%s'): %v", i+1, command, err)
			}
		} else {
			// It's literal text - use -l flag for literal UTF-8
			err := sendLiteralText(sessionName, command)
			if err != nil {
				return "", fmt.Errorf("failed to send literal text %d ('%s'): %v", i+1, command, err)
			}
		}

		// Apply default delay between commands (except for sleep commands)
		if defaultDelayMs > 0 && !strings.HasPrefix(command, "<SLEEP") {
			time.Sleep(time.Duration(defaultDelayMs) * time.Millisecond)
		}
	}

	result.WriteString("Commands executed successfully.\n")

	// Capture screen if requested
	if captureScreen {
		content, err := CapturePane(sessionName)
		if err != nil {
			result.WriteString(fmt.Sprintf("Warning: Failed to capture screen: %v\n", err))
		} else {
			result.WriteString("\nScreen content:\n")
			result.WriteString(content)
		}
	}

	return result.String(), nil
}

// executeSpecialCommand handles <COMMAND> format commands
func executeSpecialCommand(sessionName, command string) error {
	// Remove < and > brackets
	cmd := strings.TrimPrefix(strings.TrimSuffix(command, ">"), "<")

	// Handle sleep commands
	if strings.HasPrefix(cmd, "SLEEP ") {
		return handleSleepCommand(cmd)
	}

	// Map special commands to tmux key names
	tmuxKey := mapToTmuxKey(cmd)
	if tmuxKey == "" {
		return fmt.Errorf("unknown special command: %s", command)
	}

	// Send the special key using tmux send-keys
	tmuxCmd := exec.Command("tmux", "send-keys", "-t", sessionName, tmuxKey)
	return tmuxCmd.Run()
}

// sendLiteralText sends text literally using tmux -l flag
func sendLiteralText(sessionName, text string) error {
	tmuxCmd := exec.Command("tmux", "send-keys", "-l", "-t", sessionName, text)
	return tmuxCmd.Run()
}

// handleSleepCommand processes <SLEEP Xms> or <SLEEP Xs> commands
func handleSleepCommand(cmd string) error {
	// Parse "SLEEP 500ms" or "SLEEP 2s"
	parts := strings.Split(cmd, " ")
	if len(parts) != 2 {
		return fmt.Errorf("invalid sleep command format: %s", cmd)
	}

	timeStr := parts[1]
	var duration time.Duration
	var err error

	if strings.HasSuffix(timeStr, "ms") {
		ms := strings.TrimSuffix(timeStr, "ms")
		var msInt int
		if msInt, err = strconv.Atoi(ms); err != nil {
			return fmt.Errorf("invalid milliseconds value: %s", ms)
		}
		duration = time.Duration(msInt) * time.Millisecond
	} else if strings.HasSuffix(timeStr, "s") {
		s := strings.TrimSuffix(timeStr, "s")
		var seconds float64
		if seconds, err = strconv.ParseFloat(s, 64); err != nil {
			return fmt.Errorf("invalid seconds value: %s", s)
		}
		duration = time.Duration(seconds * float64(time.Second))
	} else {
		return fmt.Errorf("sleep time must end with 'ms' or 's': %s", timeStr)
	}

	time.Sleep(duration)
	return nil
}

// mapToTmuxKey maps our special commands to tmux key names
func mapToTmuxKey(cmd string) string {
	keyMap := map[string]string{
		"ENTER":     "Enter",
		"ESC":       "Escape",
		"TAB":       "Tab",
		"BACKSPACE": "BSpace",
		"DELETE":    "Delete",
		"UP":        "Up",
		"DOWN":      "Down",
		"LEFT":      "Left",
		"RIGHT":     "Right",
		"HOME":      "Home",
		"END":       "End",
		"PAGEUP":    "PPage",
		"PAGEDOWN":  "NPage",
		"SPACE":     "Space",
	}

	// Handle CTRL+ combinations
	if strings.HasPrefix(cmd, "CTRL+") {
		key := strings.TrimPrefix(cmd, "CTRL+")
		return "C-" + strings.ToLower(key)
	}

	// Handle ALT+ combinations
	if strings.HasPrefix(cmd, "ALT+") {
		key := strings.TrimPrefix(cmd, "ALT+")
		return "M-" + strings.ToLower(key)
	}

	return keyMap[cmd]
}

// stripAnsiCodes removes ANSI escape sequences from text
func stripAnsiCodes(text string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(text, "")
}

// CapturePane captures the current screen content of a session by name
func CapturePane(sessionName string) (string, error) {
	return CapturePaneRaw(sessionName, true)
}

// CapturePaneRaw captures screen content with optional ANSI stripping
func CapturePaneRaw(sessionName string, stripAnsi bool) (string, error) {
	cmd := exec.Command("tmux", "capture-pane", "-t", sessionName, "-e", "-p")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to capture screen: %v", err)
	}

	result := string(output)
	if stripAnsi {
		result = stripAnsiCodes(result)
	}

	return result, nil
}

// ListSessions returns list of active tmux sessions
func ListSessions() (string, error) {
	cmd := exec.Command("tmux", "list-sessions")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list sessions: %v", err)
	}

	return string(output), nil
}

// JoinSession joins an existing session, optionally with a new name
func JoinSession(sessionName, newSessionName string) error {
	// First check if the session exists
	cmd := exec.Command("tmux", "has-session", "-t", sessionName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("session '%s' does not exist", sessionName)
	}

	// If a new session name is provided, create a new session that shares windows with the target
	if newSessionName != "" {
		// Create a new session sharing the same session group as the target
		cmd := exec.Command("tmux", "new-session", "-d", "-s", newSessionName, "-t", sessionName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create shared session: %v", err)
		}
	}

	// Give the command time to start
	time.Sleep(200 * time.Millisecond)

	return nil
}

// KillSession closes a session by name
func KillSession(sessionName string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", sessionName)
	return cmd.Run()
}
