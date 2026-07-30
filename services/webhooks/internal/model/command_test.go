package model

import (
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		wantCmd   string
		wantArgs  []string
		wantError bool
	}{
		{
			name:     "valid add command",
			text:     "/add iphone 15",
			wantCmd:  "add",
			wantArgs: []string{"iphone", "15"},
		},
		{
			name:     "valid remove command",
			text:     "/remove query",
			wantCmd:  "remove",
			wantArgs: []string{"query"},
		},
		{
			name:     "valid clear command",
			text:     "/clear",
			wantCmd:  "clear",
			wantArgs: []string{},
		},
		{
			name:     "valid list command",
			text:     "/list",
			wantCmd:  "list",
			wantArgs: []string{},
		},
		{
			name:      "command with spaces",
			text:      "/add  iphone   15",
			wantCmd:   "add",
			wantArgs:  []string{"iphone", "15"},
			wantError: false,
		},
		{
			name:      "leading/trailing spaces",
			text:      "  /add test  ",
			wantCmd:   "add",
			wantArgs:  []string{"test"},
			wantError: false,
		},
		{
			name:      "unknown command",
			text:      "/unknown test",
			wantError: true,
		},
		{
			name:      "non-command text",
			text:      "hello world",
			wantError: true,
		},
		{
			name:      "empty text",
			text:      "",
			wantError: true,
		},
		{
			name:      "only spaces",
			text:      "   ",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := ParseCommand(tt.text)
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if cmd.Name != tt.wantCmd {
				t.Errorf("want command %q, got %q", tt.wantCmd, cmd.Name)
			}
			if len(cmd.Arguments) != len(tt.wantArgs) {
				t.Errorf("want %d args, got %d", len(tt.wantArgs), len(cmd.Arguments))
			}
			for i, arg := range tt.wantArgs {
				if cmd.Arguments[i] != arg {
					t.Errorf("arg %d: want %q, got %q", i, arg, cmd.Arguments[i])
				}
			}
		})
	}
}
