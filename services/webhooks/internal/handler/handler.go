package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/nag763/dbaradar/services/webhooks/internal/model"
	"github.com/nag763/dbaradar/services/webhooks/internal/query"
	"github.com/nag763/dbaradar/services/webhooks/internal/trusted"
)

// Handler processes webhook requests.
type Handler struct {
	logger   *slog.Logger
	cfg      *trusted.Trusted
	querySvc *query.QueryService
	validate *validator.Validate
}

// NewHandler creates a new Handler.
func NewHandler(logger *slog.Logger, cfg *trusted.Trusted, querySvc *query.QueryService) *Handler {
	return &Handler{
		logger:   logger,
		cfg:      cfg,
		querySvc: querySvc,
		validate: validator.New(),
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
	if err := h.validate.Struct(update); err != nil {
		h.logger.Debug("webhook received message has an error",
			"err", err,
		)
		return `{"error": "invalid request"}`, 200
	}

	// Extract chat and user IDs for validation
	var chatID uint64
	var userID uint64
	var chatType string
	if update.Message.Chat != nil {
		chatID = update.Message.Chat.ID
		chatType = update.Message.Chat.Type
	}
	if update.Message.From != nil {
		userID = update.Message.From.ID
	}

	// Validate chat ID and user ID are trusted
	chatIDStr := strconv.FormatUint(chatID, 10)
	userIDStr := strconv.FormatUint(userID, 10)
	isChatTrusted := h.cfg.ChatIDs[chatIDStr]
	isUserTrusted := h.cfg.ChatIDs[userIDStr]

	if !isChatTrusted && !isUserTrusted {
		h.logger.Warn("access denied: untrusted chat or user",
			"update_id", update.UpdateID,
			"chat_id", chatID,
			"chat_type", chatType,
			"user_id", userID,
			"chat_trusted", isChatTrusted,
			"user_trusted", isUserTrusted,
		)
		return `{}`, 200
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

	if err := h.validate.Struct(cmd); err != nil {
		h.logger.Warn("the received command is not legit", "err", err)
		return `{}`, 200
	}

	// Route to command handler
	var responseText string
	switch cmd.Name {
	case "add":
		itemNameToAdd := cmd.GetArgumentsJoined()
		h.logger.Debug("trying to insert new query", "chatId", chatIDStr, "item", itemNameToAdd)
		if err = h.querySvc.AddItem(ctx, query.ItemCommand{
			ClientId: chatIDStr,
			ItemName: itemNameToAdd,
		}); err != nil {
			h.logger.Error("an error happened while trying to insert the item", "err", err)
			responseText = fmt.Sprintf("An error happened while trying to add `%s`", itemNameToAdd)
			break
		}
		responseText = fmt.Sprintf("The item %s has been added to your watchlist", itemNameToAdd)
	case "remove":
		itemNameToRemove := cmd.GetArgumentsJoined()
		h.logger.Debug("trying to remove query", "chatId", chatIDStr, "item", itemNameToRemove)
		if err = h.querySvc.RemoveItem(ctx, query.ItemCommand{
			ClientId: chatIDStr,
			ItemName: itemNameToRemove,
		}); err != nil {
			h.logger.Error("an error happened while trying to remove the item", "err", err)
			responseText = fmt.Sprintf("An error happened while trying to remove `%s`", itemNameToRemove)
			break
		}
		responseText = fmt.Sprintf("The item %s has been removed from your watchlist", itemNameToRemove)
	case "list":
		h.logger.Info("trying to access all the queries for chatId", "chatId", chatID)
		items, err := h.querySvc.ListItems(ctx, chatIDStr)
		if err != nil {
			h.logger.Error("an error happened while trying to query items", "err", err)
			responseText = "An error happened while trying to retrieve your queries"
		}
		var sb strings.Builder
		sb.WriteString("Your current watchlist :")
		sb.WriteRune('\n')
		for _, item := range items {
			sb.WriteString(fmt.Sprintf("- %s\n", item.ItemName))
		}
		responseText = sb.String()
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
