// Package telegram provides a client for sending messages via the Telegram Bot API.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nag763/dbaradar/services/telegram-notifier/internal/model"
)

const (
	apiBaseURL     = "https://api.telegram.org"
	maxReasonLen   = 200
	requestTimeout = 10 * time.Second
)

// Client sends messages through the Telegram Bot API.
type Client struct {
	httpClient *http.Client
	botToken   string
	chatID     string
}

// NewClient creates a Telegram client with the given bot token and target chat.
func NewClient(botToken, chatID string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: requestTimeout},
		botToken:   botToken,
		chatID:     chatID,
	}
}

// sendMessageRequest is the payload for the Telegram sendMessage API.
type sendMessageRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// apiResponse represents a Telegram Bot API response.
type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// SendDealNotification formats and sends a deal notification to the configured chat.
func (c *Client) SendDealNotification(ctx context.Context, deal model.DealNotification) error {
	message := FormatMessage(deal)

	payload := sendMessageRequest{
		ChatID:    c.chatID,
		Text:      message,
		ParseMode: "HTML",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal telegram payload: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", apiBaseURL, c.botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram API request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var apiResp apiResponse
		json.Unmarshal(respBody, &apiResp)
		return fmt.Errorf("telegram API error (status %d): %s", resp.StatusCode, apiResp.Description)
	}

	return nil
}

// FormatMessage builds an HTML-formatted Telegram message for a deal notification.
func FormatMessage(deal model.DealNotification) string {
	reason := deal.Reason
	if len(reason) > maxReasonLen {
		reason = reason[:maxReasonLen] + "..."
	}

	var b strings.Builder
	// Header contains item text and price + currency
	fmt.Fprintf(&b, "<b>%s — %d %s</b>\n\n", escapeHTML(deal.Heading), deal.Price.Amount, escapeHTML(deal.Price.CurrencyCode))
	if deal.Location != "" {
		fmt.Fprintf(&b, "Location: %s\n", escapeHTML(deal.Location))
	}
	// Do not include the original query in notifications
	if reason != "" {
		fmt.Fprintf(&b, "%s\n\n", escapeHTML(reason))
	}
	fmt.Fprintf(&b, "<a href=\"%s\">View on DBA.dk</a>", deal.CanonicalURL)

	return b.String()
}

// escapeHTML escapes special characters for Telegram HTML parse mode.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
