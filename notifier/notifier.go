package notifier

import (
	"context"
	"fmt"
	"time"
)

// Level — статус события мониторинга.
// Level — monitoring event status.
type Level string

const (
	LevelBroken Level = "broken" // что-то сломалось / something broke
	LevelFixed  Level = "fixed"  // что-то починилось / something got fixed
	LevelInfo   Level = "info"   // нейтральное сообщение (например, ручной тест) / neutral message (e.g. a manual test)
)

func (l Level) Emoji() string {
	switch l {
	case LevelBroken:
		return "🔴"
	case LevelFixed:
		return "🟢"
	default:
		return "ℹ️"
	}
}

// RuTitle возвращает человекочитаемое название уровня на русском.
// RuTitle returns a human-readable level name in Russian.
func (l Level) RuTitle() string {
	switch l {
	case LevelBroken:
		return "Сломалось"
	case LevelFixed:
		return "Починилось"
	default:
		return "Информация"
	}
}

type Message struct {
	Level   Level     // сломалось / починилось / инфа — broken / fixed / info
	Service string    // что за сервис/компонент / which service or component
	Title   string    // короткий заголовок / short title
	Details string    // подробности (текст ошибки, метрики и т.п.), может быть пустым / details (error text, metrics, etc.), may be empty
	Time    time.Time // когда произошло событие / when the event happened
}

// PlainText — простое текстовое представление без разметки
// PlainText — a plain text representation without markup
func (m Message) PlainText() string {
	t := m.Time
	if t.IsZero() {
		t = time.Now()
	}
	s := fmt.Sprintf("%s [%s] %s: %s", m.Level.Emoji(), m.Level.RuTitle(), m.Service, m.Title)
	if m.Details != "" {
		s += "\n" + m.Details
	}
	s += "\n" + t.Format("2006-01-02 15:04:05")
	return s
}

type Notifier interface {
	// Name — короткое имя канала, для логов и отчётов об ошибках
	// Name — short channel name, used for logs and error reports
	Name() string
	// Send отправляет сообщение. Возвращает ошибку, если отправка не удалась
	// Send sends the message. Returns an error if sending failed
	Send(ctx context.Context, msg Message) error
}

// SendAll отправляет сообщение во все переданные каналы и собирает ошибки
// SendAll sends the message to all given channels and collects errors
func SendAll(ctx context.Context, notifiers []Notifier, msg Message) []error {
	var errs []error
	for _, n := range notifiers {
		if err := n.Send(ctx, msg); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
