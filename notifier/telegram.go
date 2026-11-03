package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type TelegramNotifier struct {
	BotToken string // токен / bot token
	ChatID   string // ID чата/канала(число или @username) / chat or channel ID (number or @username)
	Client   *http.Client
}

// NewTelegramNotifier создаёт нотификатор с таймаутом HTTP-клиента по умолчанию.
// NewTelegramNotifier creates a notifier with a default HTTP client timeout.
func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		BotToken: botToken,
		ChatID:   chatID,
		Client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (t *TelegramNotifier) Name() string { return "telegram" }

func (t *TelegramNotifier) buildHTML(msg Message) string {
	esc := telegramEscapeHTML
	tm := msg.Time
	if tm.IsZero() {
		tm = time.Now()
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s <b>%s</b>\n", msg.Level.Emoji(), esc(strings.ToUpper(msg.Level.RuTitle())))
	b.WriteString(" \n")
	fmt.Fprintf(&b, "🖥 <b>Сервис:</b> %s\n", esc(msg.Service))
	fmt.Fprintf(&b, "📋 <b>Событие:</b> %s\n", esc(msg.Title))

	if msg.Details != "" {
		b.WriteString("\n<b>Подробности:</b>\n")
		fmt.Fprintf(&b, "<pre>%s</pre>\n", esc(msg.Details))
	}

	fmt.Fprintf(&b, "\n🕒 <i>%s</i>", esc(tm.Format("02.01.2006 15:04:05")))
	return b.String()
}

func telegramEscapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

func (t *TelegramNotifier) Send(ctx context.Context, msg Message) error {
	if t.BotToken == "" || t.ChatID == "" {
		return fmt.Errorf("telegram: не задан bot_token или chat_id")
	}

	payload := map[string]any{
		"chat_id":    t.ChatID,
		"text":       t.buildHTML(msg),
		"parse_mode": "HTML",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: marshal payload: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.BotToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.Client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
