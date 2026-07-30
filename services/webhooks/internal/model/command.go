package model

import (
	"fmt"
	"strings"
)

// Command represents a parsed bot command.
type Command struct {
	Name      string   `validate:"required,oneof=add remove clear list"`
	Arguments []string `validate:"required_if=Name add,required_if=Name remove"`
	ChatID    uint64   `validate:"required"`
	UserID    uint64   `validate:"required,eqfield=ChatID"`
	MessageID int
}

// CommandResponse represents a response to send back to Telegram.
type CommandResponse struct {
	Method string `json:"method"`
	ChatID uint64 `json:"chat_id"`
	Text   string `json:"text"`
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

	args := []string{}
	if len(parts) > 1 {
		args = parts[1:]
	}

	return &Command{
		Name:      cmdName,
		Arguments: args,
	}, nil
}

func (c *Command) GetArgumentsJoined() string {
	if len(c.Arguments) == 0 {
		return ""
	}
	return strings.Join(c.Arguments, " ")
}

func (c *Command) HasArguments() bool {
	return len(c.GetArgumentsJoined()) != 0
}
