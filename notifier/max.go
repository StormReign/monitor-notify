package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MaxNotifier отправляет сообщения через MAX Bot API.
// MaxNotifier sends messages via the MAX Bot API.
// Документация: https://dev.max.ru/docs-api/methods/POST/messages
// Docs: https://dev.max.ru/docs-api/methods/POST/messages
type MaxNotifier struct {
	BotToken string
	ChatID   string
	BaseURL  string // по умолчанию https://platform-api2.max.ru / defaults to https://platform-api2.max.ru
	Client   *http.Client
}

func NewMaxNotifier(botToken, chatID string) *MaxNotifier {
	return &MaxNotifier{
		BotToken: botToken,
		ChatID:   chatID,
		BaseURL:  "https://platform-api2.max.ru",
		Client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (m *MaxNotifier) Name() string { return "max" }

// buildHTML строит текст сообщения в HTML-разметке MAX (format=html).
// buildHTML builds the message text in MAX's HTML markup (format=html).
func (m *MaxNotifier) buildHTML(msg Message) string {
	esc := telegramEscapeHTML // экранирование <, >, & — одинаковое для HTML / escaping <, >, & — same as for HTML
	tm := msg.Time
	if tm.IsZero() {
		tm = time.Now()
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s <b>%s</b>\n", msg.Level.Emoji(), esc(strings.ToUpper(msg.Level.RuTitle())))
	b.WriteString("\n")
	fmt.Fprintf(&b, "🖥 <b>Сервис:</b> %s\n", esc(msg.Service))
	fmt.Fprintf(&b, "📋 <b>Событие:</b> %s\n", esc(msg.Title))

	if msg.Details != "" {
		b.WriteString("\n<b>Подробности:</b>\n")
		fmt.Fprintf(&b, "<pre>%s</pre>\n", esc(msg.Details))
	}

	fmt.Fprintf(&b, "\n🕒 <i>%s</i>", esc(tm.Format("02.01.2006 15:04:05")))
	return b.String()
}

func (m *MaxNotifier) Send(ctx context.Context, msg Message) error {
	if m.BotToken == "" || m.ChatID == "" {
		return fmt.Errorf("max: не задан bot_token или chat_id")
	}

	payload := map[string]any{
		"text":   m.buildHTML(msg),
		"format": "html",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("max: marshal payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/messages?chat_id=%s", m.BaseURL, url.QueryEscape(m.ChatID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("max: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", m.BotToken)

	resp, err := m.Client.Do(req)
	if err != nil {
		return fmt.Errorf("max: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("max: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
