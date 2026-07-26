package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/nag763/dbaradar/services/webhooks/internal/config"
	"github.com/nag763/dbaradar/services/webhooks/internal/model"
)

func newTestHandler() *Handler {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cfg := &config.Config{
		TrustedChatIDs: map[string]bool{
			"456": true,
			"789": true,
		},
	}
	return NewHandler(logger, cfg)
}

func TestHandleWebhookRequest_ValidCommand(t *testing.T) {
	h := newTestHandler()

	update := model.Update{
		UpdateID: 123,
		Message: &model.Message{
			MessageID: 1,
			From: &model.User{
				ID:        456,
				FirstName: "Test",
			},
			Chat: &model.Chat{
				ID:   789,
				Type: "private",
			},
			Date: 1000,
			Text: "/add iphone 15",
		},
	}

	body, _ := json.Marshal(update)
	respBody, statusCode := h.HandleWebhookRequest(context.Background(), string(body))

	if statusCode != 200 {
		t.Errorf("want status 200, got %d", statusCode)
	}

	var resp model.CommandResponse
	json.Unmarshal([]byte(respBody), &resp)
	if resp.Method != "sendMessage" {
		t.Errorf("want method sendMessage, got %s", resp.Method)
	}
	if resp.ChatID != 789 {
		t.Errorf("want chat_id 789, got %d", resp.ChatID)
	}
}

func TestHandleWebhookRequest_InvalidJSON(t *testing.T) {
	h := newTestHandler()

	respBody, statusCode := h.HandleWebhookRequest(context.Background(), "invalid json")

	if statusCode != 400 {
		t.Errorf("want status 400, got %d", statusCode)
	}
	if respBody == "" {
		t.Error("expected error response")
	}
}

func TestHandleWebhookRequest_NonCommandText(t *testing.T) {
	h := newTestHandler()

	update := model.Update{
		UpdateID: 123,
		Message: &model.Message{
			MessageID: 1,
			Chat: &model.Chat{
				ID:   789,
				Type: "private",
			},
			Date: 1000,
			Text: "hello world",
		},
	}

	body, _ := json.Marshal(update)
	respBody, statusCode := h.HandleWebhookRequest(context.Background(), string(body))

	if statusCode != 200 {
		t.Errorf("want status 200, got %d", statusCode)
	}
	if respBody != `{}` {
		t.Errorf("want empty response for non-command, got %s", respBody)
	}
}

func TestHandleWebhookRequest_AllCommands(t *testing.T) {
	h := newTestHandler()

	commands := []string{"/add test", "/remove test", "/clear", "/list"}

	for _, cmd := range commands {
		update := model.Update{
			UpdateID: 123,
			Message: &model.Message{
				MessageID: 1,
				From: &model.User{
					ID: 456,
				},
				Chat: &model.Chat{
					ID:   789,
					Type: "private",
				},
				Date: 1000,
				Text: cmd,
			},
		}

		body, _ := json.Marshal(update)
		respBody, statusCode := h.HandleWebhookRequest(context.Background(), string(body))

		if statusCode != 200 {
			t.Errorf("command %s: want status 200, got %d", cmd, statusCode)
		}

		var resp model.CommandResponse
		json.Unmarshal([]byte(respBody), &resp)
		if resp.Method != "sendMessage" {
			t.Errorf("command %s: want method sendMessage, got %s", cmd, resp.Method)
		}
	}
}

func TestHandleWebhookRequest_UntrustedChatID(t *testing.T) {
	h := newTestHandler()

	update := model.Update{
		UpdateID: 123,
		Message: &model.Message{
			MessageID: 1,
			From: &model.User{
				ID: 999,
			},
			Chat: &model.Chat{
				ID:   888,
				Type: "group",
			},
			Date: 1000,
			Text: "/add iphone 15",
		},
	}

	body, _ := json.Marshal(update)
	respBody, statusCode := h.HandleWebhookRequest(context.Background(), string(body))

	if statusCode != 403 {
		t.Errorf("want status 403, got %d", statusCode)
	}
	if respBody != `{"error":"forbidden"}` {
		t.Errorf("want forbidden error, got %s", respBody)
	}
}

func TestHandleWebhookRequest_TrustedUserIDInGroup(t *testing.T) {
	h := newTestHandler()

	update := model.Update{
		UpdateID: 123,
		Message: &model.Message{
			MessageID: 1,
			From: &model.User{
				ID: 456,
			},
			Chat: &model.Chat{
				ID:   888,
				Type: "group",
			},
			Date: 1000,
			Text: "/add iphone 15",
		},
	}

	body, _ := json.Marshal(update)
	respBody, statusCode := h.HandleWebhookRequest(context.Background(), string(body))

	if statusCode != 200 {
		t.Errorf("want status 200, got %d", statusCode)
	}

	var resp model.CommandResponse
	json.Unmarshal([]byte(respBody), &resp)
	if resp.Method != "sendMessage" {
		t.Errorf("want method sendMessage, got %s", resp.Method)
	}
}
