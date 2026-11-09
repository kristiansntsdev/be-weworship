package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// TelegramBot posts admin alerts to one chat through the Telegram Bot API
// (TELEGRAM_BOT_TOKEN + TELEGRAM_CHAT_ID).
type TelegramBot struct {
	baseURL string
	token   string
	chatID  string
	client  *http.Client
}

// NewTelegramFromEnv returns nil (Telegram alerts disabled) when TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID is unset.
func NewTelegramFromEnv() TelegramIface {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	chatID := strings.TrimSpace(os.Getenv("TELEGRAM_CHAT_ID"))
	if token == "" || chatID == "" {
		log.Printf("[telegram] TELEGRAM_BOT_TOKEN/TELEGRAM_CHAT_ID not set — Telegram alerts disabled")
		return nil
	}
	return &TelegramBot{
		baseURL: "https://api.telegram.org",
		token:   token,
		chatID:  chatID,
		// Callers pass their own deadline via ctx; this is only a ceiling.
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type telegramSendRequest struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

type telegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

// SendMessage posts an HTML-formatted message to the configured chat.
// Callers must escape user text with html.EscapeString.
func (t *TelegramBot) SendMessage(ctx context.Context, htmlText string) error {
	body, err := json.Marshal(telegramSendRequest{
		ChatID:                t.chatID,
		Text:                  htmlText,
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		t.baseURL+"/bot"+t.token+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return errors.New("telegram: could not build request")
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := t.client.Do(req)
	if err != nil {
		// *url.Error prints the request URL, which contains the bot token; keep only the cause.
		var uerr *url.Error
		if errors.As(err, &uerr) {
			return fmt.Errorf("telegram request failed: %w", uerr.Err)
		}
		return errors.New("telegram request failed")
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	var out telegramResponse
	_ = json.Unmarshal(raw, &out)
	if res.StatusCode != http.StatusOK || !out.OK {
		return fmt.Errorf("telegram sendMessage: status %d: %s", res.StatusCode, out.Description)
	}
	return nil
}
