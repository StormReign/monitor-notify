# monitor-notify

Go-модуль для отправки уведомлений мониторинга («сломалось» / «починилось» / «инфо») в Telegram, MAX и на email (SMTP).

## Установка

```bash
go get github.com/StormReign/monitor-notify
```

## Использование как библиотеки

```go
import (
	"context"
	"time"

	"github.com/StormReign/monitor-notify/config"
	"github.com/StormReign/monitor-notify/notifier"
)

cfg, err := config.Load("config.env")
if err != nil {
	// ...
}

var notifiers []notifier.Notifier
if cfg.Telegram.Enabled {
	notifiers = append(notifiers, notifier.NewTelegramNotifier(cfg.Telegram.BotToken, cfg.Telegram.ChatID))
}

msg := notifier.Message{
	Level:   notifier.LevelBroken,
	Service: "api",
	Title:   "Сервис недоступен",
	Details: "timeout 30s",
	Time:    time.Now(),
}

ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()
errs := notifier.SendAll(ctx, notifiers, msg)
```

Полный пример — в [examples/library_usage.go](examples/library_usage.go).

## Консольная утилита

```bash
go install github.com/StormReign/monitor-notify/cmd/notify@latest
notify -level broken -service api -title "API не отвечает" -details "timeout 30s"
```

Флаги: `-config` (по умолчанию `config.env`), `-level` (`broken` | `fixed` | `info`), `-service`, `-title` (обязательный), `-details`, `-channels` (через запятую: `telegram,max,email`; пусто — все включённые), `-timeout`.

## Настройка

Скопируйте `config.example.env` в `config.env` и заполните нужные каналы. Переменные окружения с теми же именами перекрывают значения из файла. `config.env` в репозиторий не попадает (см. `.gitignore`).
