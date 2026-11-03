package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/StormReign/monitor-notify/config"
	"github.com/StormReign/monitor-notify/notifier"
)

type scenario struct {
	Level   notifier.Level
	Service string
	Title   string
	Details string
}

func main() {
	configPath := "config.env"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("ошибка загрузки конфига %s: %v\n", configPath, err)
		os.Exit(1)
	}

	var notifiers []notifier.Notifier
	if cfg.Telegram.Enabled {
		notifiers = append(notifiers, notifier.NewTelegramNotifier(cfg.Telegram.BotToken, cfg.Telegram.ChatID))
	}
	if cfg.Max.Enabled {
		notifiers = append(notifiers, notifier.NewMaxNotifier(cfg.Max.BotToken, cfg.Max.ChatID))
	}
	if cfg.Email.Enabled {
		notifiers = append(notifiers, notifier.NewEmailNotifier(
			cfg.Email.Host, cfg.Email.Port, cfg.Email.Username, cfg.Email.Password,
			cfg.Email.From, cfg.Email.To,
		))
	}
	if len(notifiers) == 0 {
		fmt.Println("нет ни одного активного канала — проверьте config.env")
		os.Exit(1)
	}

	scenarios := []scenario{
		{
			Level:   notifier.LevelBroken,
			Service: "postgres-primary",
			Title:   "Недоступен по TCP",
			Details: "connection refused, 10 попыток за 30с",
		},
		{
			Level:   notifier.LevelFixed,
			Service: "postgres-primary",
			Title:   "Снова доступен",
			Details: "",
		},
		{
			Level:   notifier.LevelBroken,
			Service: "web-frontend",
			Title:   "HTTP 502 от балансировщика",
			Details: "5 из 5 последних проверок вернули 502 Bad Gateway",
		},
		{
			Level:   notifier.LevelFixed,
			Service: "web-frontend",
			Title:   "Отвечает 200 OK",
			Details: "",
		},
	}

	fmt.Printf("запускаю тестовый прогон: %d сценариев, каналы: ", len(scenarios))
	for i, n := range notifiers {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Print(n.Name())
	}
	fmt.Println()

	for i, sc := range scenarios {
		msg := notifier.Message{
			Level:   sc.Level,
			Service: sc.Service,
			Title:   sc.Title,
			Details: sc.Details,
			Time:    time.Now(),
		}

		fmt.Printf("\n[%d/%d] %s — %s: %s\n", i+1, len(scenarios), sc.Level.RuTitle(), sc.Service, sc.Title)

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		errs := notifier.SendAll(ctx, notifiers, msg)
		cancel()

		if len(errs) == 0 {
			fmt.Println("  успешно отправлено во все каналы")
		} else {
			for _, e := range errs {
				fmt.Printf("  ошибка: %v\n", e)
			}
		}

		if i < len(scenarios)-1 {
			time.Sleep(3 * time.Second)
		}
	}

	fmt.Println("\nготово: тестовый прогон завершён")
}
