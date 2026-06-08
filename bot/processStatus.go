package bot

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func readProcMemInfo() (uint64, uint64) {
	content, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	var total uint64
	var available uint64
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, _ := strconv.ParseUint(fields[1], 10, 64)
		value *= 1024
		switch fields[0] {
		case "MemTotal:":
			total = value
		case "MemAvailable:":
			available = value
		}
	}
	return total, available
}

func cpuModelText() string {
	content, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "未知型号"
	}
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.SplitN(line, ":", 2)
		if len(fields) != 2 {
			continue
		}
		key := strings.TrimSpace(fields[0])
		if key == "model name" || key == "Hardware" || key == "Processor" {
			value := strings.TrimSpace(fields[1])
			if value != "" {
				return value
			}
		}
	}
	return "未知型号"
}

func isAudioFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp3", ".flac", ".m4a", ".aac", ".ogg", ".wav":
		return true
	default:
		return false
	}
}

func sendChunkedMessage(chatID int64, replyTo int, header string, lines []string, bot *tgbotapi.BotAPI) error {
	const maxMessageLen = 3500
	chunk := header
	first := true
	for _, line := range lines {
		next := line + "\n"
		if len(chunk)+len(next) > maxMessageLen {
			msg := tgbotapi.NewMessage(chatID, strings.TrimSpace(chunk))
			if first {
				msg.ReplyToMessageID = replyTo
				first = false
			}
			if _, err := bot.Send(msg); err != nil {
				return err
			}
			chunk = ""
		}
		chunk += next
	}
	if strings.TrimSpace(chunk) == "" {
		return nil
	}
	msg := tgbotapi.NewMessage(chatID, strings.TrimSpace(chunk))
	if first {
		msg.ReplyToMessageID = replyTo
	}
	_, err := bot.Send(msg)
	return err
}
