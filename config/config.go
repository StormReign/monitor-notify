package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Telegram struct {
		Enabled  bool
		BotToken string
		ChatID   string
	}
	Max struct {
		Enabled  bool
		BotToken string
		ChatID   string
	}
	Email struct {
		Enabled  bool
		Host     string
		Port     int
		Username string
		Password string
		From     string
		To       []string
	}
}

func Load(path string) (*Config, error) {
	raw, err := readKeyValueFile(path)
	if err != nil {
		return nil, err
	}

	// Переменные окружения перекрывают значения из файла.
	// Environment variables override values from the file.
	for _, key := range []string{
		"TELEGRAM_ENABLED", "TELEGRAM_BOT_TOKEN", "TELEGRAM_CHAT_ID",
		"MAX_ENABLED", "MAX_BOT_TOKEN", "MAX_CHAT_ID",
		"EMAIL_ENABLED", "EMAIL_HOST", "EMAIL_PORT", "EMAIL_USERNAME",
		"EMAIL_PASSWORD", "EMAIL_FROM", "EMAIL_TO",
	} {
		if v, ok := os.LookupEnv(key); ok {
			raw[key] = v
		}
	}

	cfg := &Config{}

	cfg.Telegram.Enabled = parseBool(raw["TELEGRAM_ENABLED"])
	cfg.Telegram.BotToken = raw["TELEGRAM_BOT_TOKEN"]
	cfg.Telegram.ChatID = raw["TELEGRAM_CHAT_ID"]

	cfg.Max.Enabled = parseBool(raw["MAX_ENABLED"])
	cfg.Max.BotToken = raw["MAX_BOT_TOKEN"]
	cfg.Max.ChatID = raw["MAX_CHAT_ID"]

	cfg.Email.Enabled = parseBool(raw["EMAIL_ENABLED"])
	cfg.Email.Host = raw["EMAIL_HOST"]
	cfg.Email.Port = parseIntDefault(raw["EMAIL_PORT"], 587)
	cfg.Email.Username = raw["EMAIL_USERNAME"]
	cfg.Email.Password = raw["EMAIL_PASSWORD"]
	cfg.Email.From = raw["EMAIL_FROM"]
	if raw["EMAIL_TO"] != "" {
		for _, addr := range strings.Split(raw["EMAIL_TO"], ",") {
			addr = strings.TrimSpace(addr)
			if addr != "" {
				cfg.Email.To = append(cfg.Email.To, addr)
			}
		}
	}

	return cfg, nil
}

// readKeyValueFile читает файл вида KEY=VALUE, пропуская пустые строки
// readKeyValueFile reads a KEY=VALUE file, skipping empty lines

func readKeyValueFile(path string) (map[string]string, error) {
	result := map[string]string{}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, fmt.Errorf("config: открытие файла %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			return nil, fmt.Errorf("config: %s:%d: строка без '=': %q", path, lineNum, line)
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		// Снимаем обрамляющие кавычки, если есть: KEY="value"
		// Strip surrounding quotes if present: KEY="value"
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1]
		}
		result[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("config: чтение файла %s: %w", path, err)
	}
	return result, nil
}

func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "1" || s == "true" || s == "yes" || s == "on"
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}
