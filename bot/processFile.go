package bot

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func processFileCommand(message tgbotapi.Message, bot *tgbotapi.BotAPI) error {
	if filePassword == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "文件管理未启用，请先在 config.ini 设置 FilePassword")
		msg.ReplyToMessageID = message.MessageID
		_, err := bot.Send(msg)
		return err
	}
	setFileSession(message, fileSession{
		Authed: false,
		Cwd:    fileRoot,
	})
	msg := tgbotapi.NewMessage(message.Chat.ID, inputFilePassword)
	msg.ReplyToMessageID = message.MessageID
	_, err := bot.Send(msg)
	return err
}

func processFileInput(message tgbotapi.Message, bot *tgbotapi.BotAPI) error {
	session, ok := getFileSession(message)
	if !ok {
		return nil
	}
	input := strings.TrimSpace(message.Text)
	if !session.Authed {
		if input != filePassword {
			clearFileSession(message)
			msg := tgbotapi.NewMessage(message.Chat.ID, "密码错误，已退出文件管理")
			msg.ReplyToMessageID = message.MessageID
			_, err := bot.Send(msg)
			return err
		}
		root, err := safeFilePath(fileRoot, "")
		if err != nil {
			clearFileSession(message)
			msg := tgbotapi.NewMessage(message.Chat.ID, "文件管理根目录不可用\n"+err.Error())
			msg.ReplyToMessageID = message.MessageID
			_, sendErr := bot.Send(msg)
			return sendErr
		}
		session.Authed = true
		session.Cwd = root
		setFileSession(message, session)
		return sendFileListing(message.Chat.ID, message.MessageID, session.Cwd, bot)
	}

	if input == "" || input == "ls" {
		return sendFileListing(message.Chat.ID, message.MessageID, session.Cwd, bot)
	}
	if input == "pwd" {
		msg := tgbotapi.NewMessage(message.Chat.ID, session.Cwd)
		msg.ReplyToMessageID = message.MessageID
		_, err := bot.Send(msg)
		return err
	}
	if input == "exit" || input == "quit" {
		clearFileSession(message)
		msg := tgbotapi.NewMessage(message.Chat.ID, "已退出文件管理")
		msg.ReplyToMessageID = message.MessageID
		_, err := bot.Send(msg)
		return err
	}

	cmd, arg, _ := strings.Cut(input, " ")
	arg = strings.TrimSpace(arg)
	switch cmd {
	case "cd":
		if arg == "" {
			arg = "."
		}
		next, err := safeFilePath(session.Cwd, arg)
		if err != nil {
			return sendFileError(message, bot, err)
		}
		info, err := os.Stat(next)
		if err != nil {
			return sendFileError(message, bot, err)
		}
		if !info.IsDir() {
			return sendFileError(message, bot, fmt.Errorf("不是文件夹: %s", arg))
		}
		session.Cwd = next
		setFileSession(message, session)
		return sendFileListing(message.Chat.ID, message.MessageID, session.Cwd, bot)
	case "mkdir":
		if arg == "" {
			return sendFileError(message, bot, fmt.Errorf("用法: mkdir 文件夹名"))
		}
		target, err := safeFilePath(session.Cwd, arg)
		if err != nil {
			return sendFileError(message, bot, err)
		}
		if err := os.MkdirAll(target, 0755); err != nil {
			return sendFileError(message, bot, err)
		}
		msg := tgbotapi.NewMessage(message.Chat.ID, "已创建: "+target)
		msg.ReplyToMessageID = message.MessageID
		_, err = bot.Send(msg)
		return err
	case "rm":
		if arg == "" {
			return sendFileError(message, bot, fmt.Errorf("用法: rm 文件或文件夹名"))
		}
		target, err := safeFilePath(session.Cwd, arg)
		if err != nil {
			return sendFileError(message, bot, err)
		}
		if target == fileRoot {
			return sendFileError(message, bot, fmt.Errorf("不允许删除文件管理根目录"))
		}
		if err := os.RemoveAll(target); err != nil {
			return sendFileError(message, bot, err)
		}
		msg := tgbotapi.NewMessage(message.Chat.ID, "已删除: "+target)
		msg.ReplyToMessageID = message.MessageID
		_, err = bot.Send(msg)
		return err
	case "zip":
		return processFileZip(message, session, arg, bot)
	case "unzip":
		return processFileUnzip(message, session, arg, bot)
	default:
		msg := tgbotapi.NewMessage(message.Chat.ID, "支持命令: ls, pwd, cd xxx, mkdir xxx, rm xxx, zip 压缩包名.zip 文件或文件夹, unzip 压缩包.zip [目录], exit")
		msg.ReplyToMessageID = message.MessageID
		_, err := bot.Send(msg)
		return err
	}
}

func processFileZip(message tgbotapi.Message, session fileSession, arg string, bot *tgbotapi.BotAPI) error {
	parts := strings.Fields(arg)
	if len(parts) < 2 {
		return sendFileError(message, bot, fmt.Errorf("用法: zip 压缩包名.zip 文件或文件夹"))
	}
	archivePath, err := safeFilePath(session.Cwd, parts[0])
	if err != nil {
		return sendFileError(message, bot, err)
	}
	targetPath, err := safeFilePath(session.Cwd, parts[1])
	if err != nil {
		return sendFileError(message, bot, err)
	}
	if err := createFileZip(archivePath, targetPath); err != nil {
		return sendFileError(message, bot, err)
	}
	msg := tgbotapi.NewMessage(message.Chat.ID, "已压缩: "+archivePath)
	msg.ReplyToMessageID = message.MessageID
	_, err = bot.Send(msg)
	return err
}

func processFileUnzip(message tgbotapi.Message, session fileSession, arg string, bot *tgbotapi.BotAPI) error {
	parts := strings.Fields(arg)
	if len(parts) < 1 {
		return sendFileError(message, bot, fmt.Errorf("用法: unzip 压缩包.zip [目录]"))
	}
	archivePath, err := safeFilePath(session.Cwd, parts[0])
	if err != nil {
		return sendFileError(message, bot, err)
	}
	targetDir := session.Cwd
	if len(parts) >= 2 {
		targetDir, err = safeFilePath(session.Cwd, parts[1])
		if err != nil {
			return sendFileError(message, bot, err)
		}
	}
	if err := unzipFileToDir(archivePath, targetDir); err != nil {
		return sendFileError(message, bot, err)
	}
	msg := tgbotapi.NewMessage(message.Chat.ID, "已解压到: "+targetDir)
	msg.ReplyToMessageID = message.MessageID
	_, err = bot.Send(msg)
	return err
}

func sendFileListing(chatID int64, replyTo int, dir string, bot *tgbotapi.BotAPI) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "读取目录失败\n"+err.Error())
		msg.ReplyToMessageID = replyTo
		_, sendErr := bot.Send(msg)
		return sendErr
	}
	lines := make([]string, 0, len(entries)+1)
	for _, entry := range entries {
		name := entry.Name()
		info, err := entry.Info()
		if err != nil {
			lines = append(lines, name+" (读取失败)")
			continue
		}
		if entry.IsDir() {
			lines = append(lines, "[D] "+name+"/")
			continue
		}
		lines = append(lines, fmt.Sprintf("[F] %s (%s)", name, formatBytes(uint64(info.Size()))))
	}
	sort.Strings(lines)
	header := fmt.Sprintf("当前目录: %s\n命令: ls, pwd, cd xxx, mkdir xxx, rm xxx, zip 压缩包.zip 文件或文件夹, unzip 压缩包.zip [目录], exit\n", dir)
	if len(lines) == 0 {
		lines = append(lines, "空目录")
	}
	return sendChunkedMessage(chatID, replyTo, header, lines, bot)
}

func createFileZip(archivePath, targetPath string) error {
	archive, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	baseDir := filepath.Dir(targetPath)
	return filepath.WalkDir(targetPath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		return addPathToZip(zipWriter, path, rel)
	})
}

func addPathToZip(zipWriter *zip.Writer, filePath, name string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = filepath.ToSlash(name)
	header.Method = zip.Deflate
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, file)
	return err
}

func unzipFileToDir(archivePath, targetDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		targetPath, err := safeFilePath(targetDir, file.Name)
		if err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		dst, err := os.Create(targetPath)
		if err != nil {
			_ = src.Close()
			return err
		}
		_, copyErr := io.Copy(dst, src)
		closeErr := src.Close()
		dstCloseErr := dst.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if dstCloseErr != nil {
			return dstCloseErr
		}
	}
	return nil
}

func sendFileError(message tgbotapi.Message, bot *tgbotapi.BotAPI, err error) error {
	msg := tgbotapi.NewMessage(message.Chat.ID, "操作失败\n"+err.Error())
	msg.ReplyToMessageID = message.MessageID
	_, sendErr := bot.Send(msg)
	return sendErr
}

func safeFilePath(base, input string) (string, error) {
	root, err := filepath.Abs(fileRoot)
	if err != nil {
		return "", err
	}
	fileRoot = root
	current := base
	if current == "" {
		current = root
	}
	if !filepath.IsAbs(current) {
		current = filepath.Join(root, current)
	}
	target := current
	if input != "" {
		if filepath.IsAbs(input) {
			target = input
		} else {
			target = filepath.Join(current, input)
		}
	}
	target, err = filepath.Abs(filepath.Clean(target))
	if err != nil {
		return "", err
	}
	if target != root && !strings.HasPrefix(target, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("不允许访问 %s 之外的路径", root)
	}
	return target, nil
}

func fileSessionKey(message tgbotapi.Message) string {
	return searchWaitKey(message)
}

func setFileSession(message tgbotapi.Message, session fileSession) {
	key := fileSessionKey(message)
	if key == "" {
		return
	}
	fileSessions.Lock()
	defer fileSessions.Unlock()
	fileSessions.users[key] = session
}

func getFileSession(message tgbotapi.Message) (fileSession, bool) {
	key := fileSessionKey(message)
	if key == "" {
		return fileSession{}, false
	}
	fileSessions.Lock()
	defer fileSessions.Unlock()
	session, ok := fileSessions.users[key]
	return session, ok
}

func clearFileSession(message tgbotapi.Message) {
	key := fileSessionKey(message)
	if key == "" {
		return
	}
	fileSessions.Lock()
	defer fileSessions.Unlock()
	delete(fileSessions.users, key)
}

func isFileSession(message tgbotapi.Message) bool {
	_, ok := getFileSession(message)
	return ok
}
