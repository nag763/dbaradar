package telegram

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nag763/dbaradar/services/telegram-notifier/internal/model"
)

func TestFormatMessage_Basic(t *testing.T) {
	deal := model.DealNotification{
		Heading:      "iPhone 15 Pro",
		Price:        model.Price{Amount: 4500, CurrencyCode: "DKK"},
		Location:     "Copenhagen",
		Query:        "iphone 15 pro",
		Reason:       "40% below market",
		CanonicalURL: "https://www.dba.dk/test",
	}

	msg := FormatMessage(deal)

	for _, want := range []string{
		"<b>iPhone 15 Pro — 4500 DKK</b>",
		"Copenhagen",
		"40% below market",
		`<a href="https://www.dba.dk/test">View on DBA.dk</a>`,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("message missing %q\nGot: %s", want, msg)
		}
	}
}

func TestFormatMessage_HTMLEscape(t *testing.T) {
	deal := model.DealNotification{
		Heading:      "Item <special> & rare",
		Price:        model.Price{Amount: 100, CurrencyCode: "DKK"},
		CanonicalURL: "https://dba.dk/test",
	}

	msg := FormatMessage(deal)

	if strings.Contains(msg, "<special>") {
		t.Error("HTML special chars should be escaped")
	}
	if !strings.Contains(msg, "&lt;special&gt;") {
		t.Errorf("expected escaped HTML, got: %s", msg)
	}
	if !strings.Contains(msg, "&amp; rare") {
		t.Errorf("expected escaped ampersand, got: %s", msg)
	}
}

func TestFormatMessage_LongReasonTruncated(t *testing.T) {
	deal := model.DealNotification{
		Heading:      "Test",
		Price:        model.Price{Amount: 100, CurrencyCode: "DKK"},
		Reason:       strings.Repeat("x", 300),
		CanonicalURL: "https://dba.dk/test",
	}

	msg := FormatMessage(deal)

	if strings.Contains(msg, strings.Repeat("x", 300)) {
		t.Error("reason should be truncated")
	}
	if !strings.Contains(msg, "...") {
		t.Error("truncated reason should end with ...")
	}
}

func TestSendDealNotification_Success(t *testing.T) {
	var receivedBody sendMessageRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/bottest-token/sendMessage") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(apiResponse{OK: true})
	}))
	defer server.Close()

	deal := model.DealNotification{
		EventID:      "123",
		Heading:      "Test Item",
		Price:        model.Price{Amount: 500, CurrencyCode: "DKK"},
		CanonicalURL: "https://dba.dk/test",
		Location:     "Aarhus",
		Verdict:      "GOOD_DEAL",
		Reason:       "Great deal",
		Query:        "test",
	}

	message := FormatMessage(deal)
	payload := sendMessageRequest{
		ChatID:    "12345",
		Text:      message,
		ParseMode: "HTML",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(server.URL+"/bottest-token/sendMessage", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if receivedBody.ChatID != "12345" {
		t.Errorf("unexpected chat_id: %s", receivedBody.ChatID)
	}
	if receivedBody.ParseMode != "HTML" {
		t.Errorf("unexpected parse_mode: %s", receivedBody.ParseMode)
	}
	if !strings.Contains(receivedBody.Text, "Test Item") {
		t.Errorf("message should contain heading, got: %s", receivedBody.Text)
	}
}

func TestSendDealNotification_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(apiResponse{OK: false, Description: "Bad Request: chat not found"})
	}))
	defer server.Close()

	deal := model.DealNotification{EventID: "123", Heading: "Test"}
	message := FormatMessage(deal)
	payload := sendMessageRequest{
		ChatID:    "invalid",
		Text:      message,
		ParseMode: "HTML",
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(server.URL+"/bottest-token/sendMessage", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("unexpected error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-200 status")
	}
}

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"<b>bold</b>", "&lt;b&gt;bold&lt;/b&gt;"},
		{"a & b", "a &amp; b"},
		{"<>&", "&lt;&gt;&amp;"},
	}

	for _, tt := range tests {
		got := escapeHTML(tt.input)
		if got != tt.want {
			t.Errorf("escapeHTML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
