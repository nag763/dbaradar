package model

import (
	"fmt"
	"strings"
)

// Command represents a parsed bot command.
type Command struct {
	Name      string
	Arguments []string
	ChatID    int64
	UserID    int64
	MessageID int
}

// CommandResponse represents a response to send back to Telegram.
type CommandResponse struct {
	Method string `json:"method"`
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

var validCommands = map[string]bool{
	"add":    true,
	"remove": true,
	"clear":  true,
	"list":   true,
}

// ParseCommand extracts a command from message text.
// Returns the parsed command or an error if the text is not a valid command.
func ParseCommand(text string) (*Command, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}

	parts := strings.Fields(text)
	if len(parts) == 0 {
		return nil, fmt.Errorf("no command found")
	}

	cmdPart := parts[0]
	if !strings.HasPrefix(cmdPart, "/") {
		return nil, fmt.Errorf("not a command")
	}

	cmdName := strings.TrimPrefix(cmdPart, "/")
	if !validCommands[cmdName] {
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}

	args := []string{}
	if len(parts) > 1 {
		args = parts[1:]
	}

	return &Command{
		Name:      cmdName,
		Arguments: args,
	}, nil
}
