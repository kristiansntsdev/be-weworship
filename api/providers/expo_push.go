package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const expoAPIURL = "https://exp.host/--/api/v2/push/send"

// ExpoPushProvider sends push notifications via the Expo Push Notifications API.
// It accepts ExponentPushToken[...] tokens directly, so no Firebase credentials are needed.
type ExpoPushProvider struct {
	client *http.Client
}

// NewExpoPushProvider creates a new ExpoPushProvider.
func NewExpoPushProvider() *ExpoPushProvider {
	return &ExpoPushProvider{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// expoMessage is the payload for a single Expo push notification.
type expoMessage struct {
	To       string            `json:"to"`
	Title    string            `json:"title"`
	Body     string            `json:"body"`
	Data     map[string]string `json:"data,omitempty"`
	Sound    string            `json:"sound"`
	Priority string            `json:"priority"`
}

// expoResponse is the outer envelope returned by the Expo API.
type expoResponse struct {
	Data []struct {
		Status  string `json:"status"`
		ID      string `json:"id"`
		Message string `json:"message"`
		Details struct {
			Error string `json:"error"`
		} `json:"details"`
	} `json:"data"`
}

// Send sends notifications to a batch of Expo push tokens (max 100 per call).
// Tokens that are not valid Expo tokens are filtered out with a warning.
func (p *ExpoPushProvider) Send(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	if len(tokens) == 0 {
		return nil
	}

	// Filter to valid Expo tokens only
	var valid []string
	for _, t := range tokens {
		if strings.HasPrefix(t, "ExponentPushToken[") || strings.HasPrefix(t, "ExpoPushToken[") {
			valid = append(valid, t)
		} else {
			log.Printf("[expo] skipping non-Expo token: %.30s...", t)
		}
	}
	if len(valid) == 0 {
		log.Printf("[expo] no valid Expo tokens in batch of %d", len(tokens))
		return nil
	}

	// Build message batch
	msgs := make([]expoMessage, len(valid))
	for i, tok := range valid {
		msgs[i] = expoMessage{
			To:       tok,
			Title:    title,
			Body:     body,
			Data:     data,
			Sound:    "default",
			Priority: "high",
		}
	}

	payload, err := json.Marshal(msgs)
	if err != nil {
		return fmt.Errorf("expo: marshal failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, expoAPIURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("expo: create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("expo: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	var result expoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("expo: decode response failed: %w", err)
	}

	okCount := 0
	for i, item := range result.Data {
		if item.Status == "ok" {
			okCount++
		} else {
			tok := ""
			if i < len(valid) {
				tok = valid[i][:min(20, len(valid[i]))]
			}
			log.Printf("[expo] delivery error token=%.20s... status=%s error=%s msg=%s",
				tok, item.Status, item.Details.Error, item.Message)
		}
	}
	log.Printf("[expo] Send: %d/%d delivered OK", okCount, len(valid))
	return nil
}

// SendToTokens is an alias for Send that takes variadic tokens for single sends.
func (p *ExpoPushProvider) SendToTokens(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	return p.Send(ctx, tokens, title, body, data)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
