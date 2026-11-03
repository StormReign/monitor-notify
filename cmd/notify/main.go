package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/StormReign/monitor-notify/config"
	"github.com/StormReign/monitor-notify/notifier"
)

func main() {
	configPath := flag.String("config", "config.env", "путь к файлу конфигурации")
	level := flag.String("level", "info", "уровень события: broken | fixed | info")
	service := flag.String("service", "", "название сервиса/компонента")
	title := flag.String("title", "", "короткий заголовок события")
	details := flag.String("details", "", "подробности (текст ошибки, лог и т.п.)")
	channels := flag.String("channels", "", "через запятую: telegram,max,email (пусто = все включённые в конфиге)")
	timeout := flag.Duration("timeout", 15*time.Second, "таймаут на отправку во все каналы")
	flag.Parse()

	if *title == "" {
		fmt.Fprintln(os.Stderr, "ошибка: не указан -title")
		flag.Usage()
		os.Exit(2)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ошибка загрузки конфига: %v\n", err)
		os.Exit(1)
	}

	var lvl notifier.Level
	switch strings.ToLower(*level) {
	case "broken":
		lvl = notifier.LevelBroken
	case "fixed":
		lvl = notifier.LevelFixed
	default:
		lvl = notifier.LevelInfo
	}

	msg := notifier.Message{
		Level:   lvl,
		Service: *service,
		Title:   *title,
		Details: *details,
		Time:    time.Now(),
	}

	wanted := map[string]bool{}
	if *channels != "" {
		for _, c := range strings.Split(*channels, ",") {
			wanted[strings.TrimSpace(strings.ToLower(c))] = true
		}
	}
	want := func(name string, enabledInConfig bool) bool {
		if len(wanted) == 0 {
			return enabledInConfig
		}
		return wanted[name]
	}

	var notifiers []notifier.Notifier

	if want("telegram", cfg.Telegram.Enabled) {
		notifiers = append(notifiers, notifier.NewTelegramNotifier(cfg.Telegram.BotToken, cfg.Telegram.ChatID))
	}
	if want("max", cfg.Max.Enabled) {
		notifiers = append(notifiers, notifier.NewMaxNotifier(cfg.Max.BotToken, cfg.Max.ChatID))
	}
	if want("email", cfg.Email.Enabled) {
		notifiers = append(notifiers, notifier.NewEmailNotifier(
			cfg.Email.Host, cfg.Email.Port, cfg.Email.Username, cfg.Email.Password,
			cfg.Email.From, cfg.Email.To,
		))
	}

	if len(notifiers) == 0 {
		fmt.Fprintln(os.Stderr, "нет ни одного активного канала — проверьте конфиг или -channels")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	errs := notifier.SendAll(ctx, notifiers, msg)
	for _, n := range notifiers {
		status := "OK"
		for _, e := range errs {
			if strings.HasPrefix(e.Error(), n.Name()+":") {
				status = "FAIL"
			}
		}
		fmt.Printf("[%s] %s\n", n.Name(), status)
	}
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "ошибка: %v\n", e)
		}
		os.Exit(1)
	}
	fmt.Println("готово: уведомление отправлено во все выбранные каналы")
}
