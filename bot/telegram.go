package bot

import (
	"errors"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func sendNonCritical(bot *tgbotapi.BotAPI, c tgbotapi.Chattable) error {
	_, err := bot.Send(c)
	if isIgnorableTelegramError(err) {
		return nil
	}
	return err
}

func requestNonCritical(bot *tgbotapi.BotAPI, c tgbotapi.Chattable) error {
	_, err := bot.Request(c)
	if isIgnorableTelegramError(err) {
		return nil
	}
	return err
}

func isIgnorableTelegramError(err error) bool {
	if err == nil {
		return false
	}
	var tgErr tgbotapi.Error
	if errors.As(err, &tgErr) {
		if tgErr.RetryAfter > 0 {
			return true
		}
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "too many requests") ||
		strings.Contains(text, "message is not modified")
}
