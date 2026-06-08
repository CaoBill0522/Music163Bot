package bot

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func processDownloadArchiveCommand(message tgbotapi.Message, bot *tgbotapi.BotAPI) error {
	setDownloadArchiveWait(message)
	msg := tgbotapi.NewMessage(message.Chat.ID, inputDownloadType)
	msg.ReplyToMessageID = message.MessageID
	_, err := bot.Send(msg)
	return err
}

func processDownloadArchiveInput(message tgbotapi.Message, input string, bot *tgbotapi.BotAPI) error {
	kind := strings.TrimSpace(strings.ToLower(input))
	var dir string
	var name string
	switch kind {
	case "music":
		dir = downloadDir
		name = "music"
	case "musicmp3", "mp3", "music_mp3":
		dir = mp3Dir
		name = "musicmp3"
	default:
		setDownloadArchiveWait(message)
		msg := tgbotapi.NewMessage(message.Chat.ID, "未识别目录\n"+inputDownloadType)
		msg.ReplyToMessageID = message.MessageID
		_, err := bot.Send(msg)
		return err
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "正在压缩 "+name+" 目录...")
	msg.ReplyToMessageID = message.MessageID
	msgResult, err := bot.Send(msg)
	if err != nil {
		return err
	}

	startTask(message.Chat.ID, "打包下载 "+name)
	defer finishTask(message.Chat.ID)

	archivePath, fileCount, err := createMusicArchive(dir, name)
	if err != nil {
		editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, msgResult.MessageID, "压缩失败\n"+err.Error())
		_, _ = bot.Send(editMsg)
		return err
	}

	link := fileDownloadLink(archivePath)
	status := fmt.Sprintf("压缩完成\n目录: %s\n歌曲数: %d\n压缩包: %s", dir, fileCount, archivePath)
	if link == "" {
		status += "\n未配置 FileURLBase，无法生成下载直链"
	} else {
		status += "\n下载直链: " + link
	}
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, msgResult.MessageID, status)
	_, err = bot.Send(editMsg)
	return err
}

func createMusicArchive(dir, name string) (string, int, error) {
	if _, err := os.Stat(dir); err != nil {
		return "", 0, err
	}
	files := make([]string, 0)
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !isAudioFile(entry.Name()) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return "", 0, err
	}
	if len(files) == 0 {
		return "", 0, fmt.Errorf("目录中没有可打包的歌曲")
	}
	archivePath := filepath.Join(dir, fmt.Sprintf("%s_%s.zip", name, time.Now().Format("20060102_150405")))
	if err := createZipArchive(archivePath, files); err != nil {
		return "", 0, err
	}
	return archivePath, len(files), nil
}

func fileDownloadLink(filePath string) string {
	if fileURLBase == "" {
		return ""
	}
	root, err := filepath.Abs(fileRoot)
	if err != nil {
		return ""
	}
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(root, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return fileURLBase + "/" + strings.Join(parts, "/")
}

func setDownloadArchiveWait(message tgbotapi.Message) {
	key := searchWaitKey(message)
	if key == "" {
		return
	}
	downloadArchiveWait.Lock()
	defer downloadArchiveWait.Unlock()
	downloadArchiveWait.users[key] = true
}

func clearDownloadArchiveWait(message tgbotapi.Message) {
	key := searchWaitKey(message)
	if key == "" {
		return
	}
	downloadArchiveWait.Lock()
	defer downloadArchiveWait.Unlock()
	delete(downloadArchiveWait.users, key)
}

func consumeDownloadArchiveWait(message tgbotapi.Message) bool {
	if message.Text == "" {
		return false
	}
	key := searchWaitKey(message)
	if key == "" {
		return false
	}
	downloadArchiveWait.Lock()
	defer downloadArchiveWait.Unlock()
	if !downloadArchiveWait.users[key] {
		return false
	}
	delete(downloadArchiveWait.users, key)
	return true
}
