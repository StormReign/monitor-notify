package examples

import (
	"context"
	"time"

	"github.com/StormReign/monitor-notify/config"
	"github.com/StormReign/monitor-notify/notifier"
)

// шлём уведомление во все включённые каналы.
// sends a notification to all enabled channels.
func CheckAndNotify(cfg *config.Config, serviceName string, isUp bool, errText string) error {
	var notifiers []notifier.Notifier
	if cfg.Telegram.Enabled {
		notifiers = append(notifiers, notifier.NewTelegramNotifier(cfg.Telegram.BotToken, cfg.Telegram.ChatID))
	}
	if cfg.Email.Enabled {
		notifiers = append(notifiers, notifier.NewEmailNotifier(
			cfg.Email.Host, cfg.Email.Port, cfg.Email.Username, cfg.Email.Password,
			cfg.Email.From, cfg.Email.To,
		))
	}

	level := notifier.LevelFixed
	title := "Сервис снова в норме"
	if !isUp {
		level = notifier.LevelBroken
		title = "Сервис недоступен"
	}

	msg := notifier.Message{
		Level:   level,
		Service: serviceName,
		Title:   title,
		Details: errText,
		Time:    time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	errs := notifier.SendAll(ctx, notifiers, msg)
	if len(errs) > 0 {
		return errs[0] // или залогировать все errs, как вам удобнее / or log all errs, whichever you prefer
	}
	return nil
}
