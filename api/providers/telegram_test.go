package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testBotToken = "123456:SECRET-token"

func newTestBot(baseURL string) *TelegramBot {
	return &TelegramBot{baseURL: baseURL, token: testBotToken, chatID: "-1001", client: &http.Client{Timeout: 2 * time.Second}}
}

func TestNewTelegramFromEnv_DisabledWithoutChatID(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", testBotToken)
	t.Setenv("TELEGRAM_CHAT_ID", "")
	assert.Nil(t, NewTelegramFromEnv())
}

func TestTelegramBot_SendMessage_PostsToChat(t *testing.T) {
	var got telegramSendRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/bot"+testBotToken+"/sendMessage", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer srv.Close()

	require.NoError(t, newTestBot(srv.URL).SendMessage(context.Background(), "<b>hi</b>"))
	assert.Equal(t, "-1001", got.ChatID)
	assert.Equal(t, "<b>hi</b>", got.Text)
	assert.Equal(t, "HTML", got.ParseMode)
}

func TestTelegramBot_SendMessage_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request: chat not found"}`))
	}))
	defer srv.Close()

	err := newTestBot(srv.URL).SendMessage(context.Background(), "hi")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chat not found")
}

func TestTelegramBot_SendMessage_NetworkErrorHidesToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // connection refused

	err := newTestBot(url).SendMessage(context.Background(), "hi")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), testBotToken)
}
