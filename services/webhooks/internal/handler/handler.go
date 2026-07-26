package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/nag763/dbaradar/services/webhooks/internal/config"
	"github.com/nag763/dbaradar/services/webhooks/internal/model"
)

// Handler processes webhook requests.
type Handler struct {
	logger *slog.Logger
	cfg    *config.Config
}

// NewHandler creates a new Handler.
func NewHandler(logger *slog.Logger, cfg *config.Config) *Handler {
	return &Handler{
		logger: logger,
		cfg:    cfg,
	}
}

// HandleWebhookRequest processes a Telegram webhook request and returns JSON response + status code.
func (h *Handler) HandleWebhookRequest(ctx context.Context, body string) (string, int) {
	var update model.Update
	if err := json.Unmarshal([]byte(body), &update); err != nil {
		h.logger.Error("failed to decode webhook request",
			"error", err,
			"body_length", len(body),
		)
		return `{"error":"invalid request"}`, 400
	}

	// Extract message and text
	if update.Message == nil || update.Message.Text == "" {
		h.logger.Debug("webhook received message without text",
			"update_id", update.UpdateID,
			"has_message", update.Message != nil,
		)
		return `{}`, 200
	}

	// Extract chat and user IDs for validation
	var chatID int64
	var userID int64
	var chatType string
	if update.Message.Chat != nil {
		chatID = update.Message.Chat.ID
		chatType = update.Message.Chat.Type
	}
	if update.Message.From != nil {
		userID = update.Message.From.ID
	}

	// Validate chat ID and user ID are trusted
	chatIDStr := strconv.FormatInt(chatID, 10)
	userIDStr := strconv.FormatInt(userID, 10)
	isChatTrusted := h.cfg.TrustedChatIDs[chatIDStr]
	isUserTrusted := h.cfg.TrustedChatIDs[userIDStr]

	if !isChatTrusted && !isUserTrusted {
		h.logger.Warn("access denied: untrusted chat or user",
			"update_id", update.UpdateID,
			"chat_id", chatID,
			"chat_type", chatType,
			"user_id", userID,
			"chat_trusted", isChatTrusted,
			"user_trusted", isUserTrusted,
		)
		return `{"error":"forbidden"}`, 403
	}

	h.logger.Debug("webhook request accepted",
		"update_id", update.UpdateID,
		"chat_id", chatID,
		"chat_type", chatType,
		"user_id", userID,
	)

	// Parse command
	cmd, err := model.ParseCommand(update.Message.Text)
	if err != nil {
		h.logger.Debug("non-command text received",
			"update_id", update.UpdateID,
			"text", update.Message.Text,
		)
		return `{}`, 200
	}

	// Set command metadata
	cmd.ChatID = chatID
	cmd.UserID = userID
	cmd.MessageID = update.Message.MessageID

	// Log command
	h.logger.Info("command received",
		"command", cmd.Name,
		"args", cmd.Arguments,
		"chat_id", cmd.ChatID,
		"user_id", cmd.UserID,
		"message_id", cmd.MessageID,
	)

	// Route to command handler
	var responseText string
	switch cmd.Name {
	case "add":
		responseText = fmt.Sprintf("Echo /%s with args: %v", cmd.Name, cmd.Arguments)
	case "remove":
		responseText = fmt.Sprintf("Echo /%s with args: %v", cmd.Name, cmd.Arguments)
	case "clear":
		responseText = fmt.Sprintf("Echo /%s with args: %v", cmd.Name, cmd.Arguments)
	case "list":
		responseText = fmt.Sprintf("Echo /%s with args: %v", cmd.Name, cmd.Arguments)
	default:
		responseText = "Unknown command"
	}

	// Build response
	response := model.CommandResponse{
		Method: "sendMessage",
		ChatID: cmd.ChatID,
		Text:   responseText,
	}

	respBody, err := json.Marshal(response)
	if err != nil {
		h.logger.Error("failed to marshal command response",
			"error", err,
			"command", cmd.Name,
		)
		return `{"error":"internal error"}`, 500
	}
	return string(respBody), 200
}
