package notifier

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

type EmailNotifier struct {
	Host     string   // smtp.yandex.ru
	Port     int
	Username string   // логин для аутентификации на SMTP / SMTP auth login
	Password string   // пароль или пароль приложения / password or app password
	From     string   // адрес отправителя / sender address
	To       []string // список получателей / list of recipients
}

func NewEmailNotifier(host string, port int, username, password, from string, to []string) *EmailNotifier {
	return &EmailNotifier{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
		To:       to,
	}
}

func (e *EmailNotifier) Name() string { return "email" }

// html строит HTML-тело письма. / html builds the HTML body of the email.
func (e *EmailNotifier) html(msg Message) string {
	tm := msg.Time
	if tm.IsZero() {
		tm = time.Now()
	}

	accent, tint := "#6c757d", "#f1f3f5"
	switch msg.Level {
	case LevelBroken:
		accent, tint = "#dc3545", "#fdecea"
	case LevelFixed:
		accent, tint = "#28a745", "#eaf7ed"
	}

	details := ""
	if msg.Details != "" {
		details = fmt.Sprintf(`
      <tr><td style="padding:16px 24px 0 24px;">
        <div style="font-size:12px;font-weight:600;color:#6c757d;text-transform:uppercase;letter-spacing:.04em;margin-bottom:6px;">Подробности</div>
        <pre style="background:#f5f5f5;color:#212529;padding:12px 14px;border-radius:8px;white-space:pre-wrap;word-break:break-word;font-size:13px;line-height:1.5;margin:0;">%s</pre>
      </td></tr>`, htmlEscape(msg.Details))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="margin:0;padding:24px;background:#f0f2f5;font-family:-apple-system,Segoe UI,Roboto,Arial,sans-serif;">
  <table role="presentation" width="100%%" style="max-width:520px;margin:0 auto;background:#ffffff;border-radius:12px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,.08);">
    <tr>
      <td style="background:%s;padding:20px 24px;">
        <div style="font-size:20px;font-weight:700;color:#ffffff;">%s %s</div>
      </td>
    </tr>
    <tr>
      <td style="padding:20px 24px 4px 24px;">
        <table role="presentation" width="100%%" style="border-collapse:collapse;">
          <tr>
            <td style="padding:6px 0;color:#6c757d;font-size:13px;width:110px;vertical-align:top;">Сервис</td>
            <td style="padding:6px 0;color:#212529;font-size:14px;font-weight:600;">%s</td>
          </tr>
          <tr>
            <td style="padding:6px 0;color:#6c757d;font-size:13px;vertical-align:top;">Событие</td>
            <td style="padding:6px 0;color:#212529;font-size:14px;">%s</td>
          </tr>
        </table>
      </td>
    </tr>
    %s
    <tr>
      <td style="padding:18px 24px;background:%s;">
        <span style="color:#6c757d;font-size:12px;">🕒 %s &nbsp;·&nbsp; система мониторинга</span>
      </td>
    </tr>
  </table>
</body>
</html>`,
		accent, msg.Level.Emoji(), htmlEscape(msg.Level.RuTitle()),
		htmlEscape(msg.Service), htmlEscape(msg.Title),
		details,
		tint, tm.Format("02.01.2006 15:04:05"))
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

func (e *EmailNotifier) Send(ctx context.Context, msg Message) error {
	if e.Host == "" || e.From == "" || len(e.To) == 0 {
		return fmt.Errorf("email: не задан host, from или список получателей")
	}

	subject := fmt.Sprintf("[%s] %s: %s", msg.Level.RuTitle(), msg.Service, msg.Title)
	body := e.html(msg)

	headers := map[string]string{
		"From":         e.From,
		"To":           strings.Join(e.To, ", "),
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=\"UTF-8\"",
	}
	var msgBuf strings.Builder
	for k, v := range headers {
		fmt.Fprintf(&msgBuf, "%s: %s\r\n", k, v)
	}
	msgBuf.WriteString("\r\n")
	msgBuf.WriteString(body)

	addr := fmt.Sprintf("%s:%d", e.Host, e.Port)

	if deadline, ok := ctx.Deadline(); ok {
		_ = deadline
	}

	var auth smtp.Auth
	if e.Username != "" {
		auth = smtp.PlainAuth("", e.Username, e.Password, e.Host)
	}

	if e.Port == 465 {
		return e.sendImplicitTLS(addr, auth, msgBuf.String())
	}
	return smtp.SendMail(addr, auth, e.From, e.To, []byte(msgBuf.String()))
}

func (e *EmailNotifier) sendImplicitTLS(addr string, auth smtp.Auth, message string) error {
	tlsConfig := &tls.Config{ServerName: e.Host}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("email: tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, e.Host)
	if err != nil {
		return fmt.Errorf("email: smtp client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("email: auth: %w", err)
		}
	}
	if err := client.Mail(e.From); err != nil {
		return fmt.Errorf("email: mail from: %w", err)
	}
	for _, to := range e.To {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("email: rcpt to %s: %w", to, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("email: data: %w", err)
	}
	if _, err := w.Write([]byte(message)); err != nil {
		return fmt.Errorf("email: write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("email: close writer: %w", err)
	}
	return client.Quit()
}
