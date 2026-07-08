package bot

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/XiaoMengXinX/Music163Api-Go/api"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func taskChatKey(message tgbotapi.Message) int64 {
	return message.Chat.ID
}

func startTask(chatID int64, name string) context.Context {
	taskControl.Lock()
	defer taskControl.Unlock()
	if cancel := taskControl.cancel[chatID]; cancel != nil {
		cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	taskControl.stopped[chatID] = false
	taskControl.active[chatID] = name
	taskControl.ctx[chatID] = ctx
	taskControl.cancel[chatID] = cancel
	return ctx
}

func finishTask(chatID int64, ctx context.Context) {
	taskControl.Lock()
	defer taskControl.Unlock()
	if taskControl.ctx[chatID] != ctx {
		return
	}
	if cancel := taskControl.cancel[chatID]; cancel != nil {
		cancel()
	}
	delete(taskControl.active, chatID)
	delete(taskControl.stopped, chatID)
	delete(taskControl.ctx, chatID)
	delete(taskControl.cancel, chatID)
}

func requestStop(chatID int64) bool {
	taskControl.Lock()
	defer taskControl.Unlock()
	_, active := taskControl.active[chatID]
	taskControl.stopped[chatID] = true
	if cancel := taskControl.cancel[chatID]; cancel != nil {
		cancel()
	}
	return active
}

func isTaskStopped(chatID int64) bool {
	taskControl.Lock()
	defer taskControl.Unlock()
	return taskControl.stopped[chatID]
}

func activeTaskSnapshot() map[int64]string {
	taskControl.Lock()
	defer taskControl.Unlock()
	result := make(map[int64]string, len(taskControl.active))
	for chatID, name := range taskControl.active {
		result[chatID] = name
	}
	return result
}

func taskContext(chatID int64) context.Context {
	taskControl.Lock()
	defer taskControl.Unlock()
	if ctx := taskControl.ctx[chatID]; ctx != nil {
		return ctx
	}
	return context.Background()
}

func processStopCommand(message tgbotapi.Message, bot *tgbotapi.BotAPI) error {
	pending := hasPendingInteraction(message)
	clearSearchWait(message)
	clearSearchMP3Wait(message)
	clearPlaylistWait(message)
	clearPlaylistMP3Wait(message)
	clearPlaylistSelectionWait(message)
	clearDownloadArchiveWait(message)
	clearFileSession(message)

	active := requestStop(taskChatKey(message))
	text := taskStopped
	if !active && !pending {
		text = noActiveTask
	}
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyToMessageID = message.MessageID
	_, err := bot.Send(msg)
	return err
}

func hasPendingInteraction(message tgbotapi.Message) bool {
	key := searchWaitKey(message)
	if key == "" {
		return false
	}

	searchWait.Lock()
	searching := searchWait.users[key]
	searchWait.Unlock()
	if searching {
		return true
	}

	searchMP3Wait.Lock()
	searchMP3 := searchMP3Wait.users[key]
	searchMP3Wait.Unlock()
	if searchMP3 {
		return true
	}

	playlistWait.Lock()
	playlist := playlistWait.users[key]
	playlistWait.Unlock()
	if playlist {
		return true
	}

	playlistMP3Wait.Lock()
	playlistMP3 := playlistMP3Wait.users[key]
	playlistMP3Wait.Unlock()
	if playlistMP3 {
		return true
	}

	playlistSelectionWait.Lock()
	_, selecting := playlistSelectionWait.users[key]
	playlistSelectionWait.Unlock()
	if selecting {
		return true
	}

	downloadArchiveWait.Lock()
	downloadingArchive := downloadArchiveWait.users[key]
	downloadArchiveWait.Unlock()
	if downloadingArchive {
		return true
	}

	if isFileSession(message) {
		return true
	}

	return false
}

func processTasksCommand(message tgbotapi.Message, bot *tgbotapi.BotAPI) error {
	active := activeTaskSnapshot()
	if len(active) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "当前没有正在进行的任务")
		msg.ReplyToMessageID = message.MessageID
		_, err := bot.Send(msg)
		return err
	}
	lines := make([]string, 0, len(active))
	for chatID, name := range active {
		lines = append(lines, fmt.Sprintf("%d: %s", chatID, name))
	}
	sort.Strings(lines)
	msg := tgbotapi.NewMessage(message.Chat.ID, "当前任务:\n"+strings.Join(lines, "\n"))
	msg.ReplyToMessageID = message.MessageID
	_, err := bot.Send(msg)
	return err
}

func processStartCommand(message tgbotapi.Message, bot *tgbotapi.BotAPI) error {
	text := `功能菜单
/search - 搜索并下载歌曲，自动嵌入信息和歌词
/searchmp3 - 搜索并下载为 MP3，自动嵌入信息和歌词
/playlist - 下载歌单，支持全部或区间
/playlistmp3 - 下载歌单为 MP3，支持全部或区间，自动嵌入信息和歌词
/tasks - 查看当前任务
/stop - 停止当前任务或退出当前交互
/file - 密码验证后管理文件
/status - 查看服务器和 Bot 状态
/download - 打包 music 或 musicmp3 并生成下载直链`
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyToMessageID = message.MessageID
	_, err := bot.Send(msg)
	return err
}

func processStatusCommand(message tgbotapi.Message, bot *tgbotapi.BotAPI) error {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	active := activeTaskSnapshot()
	activeLines := make([]string, 0, len(active))
	for chatID, name := range active {
		activeLines = append(activeLines, fmt.Sprintf("%d: %s", chatID, name))
	}
	sort.Strings(activeLines)
	activeText := "无"
	if len(activeLines) > 0 {
		activeText = strings.Join(activeLines, "\n")
	}
	hostname, _ := os.Hostname()

	status := fmt.Sprintf(`服务器状态
主机: %s
系统: %s/%s
CPU: %s (%d 核)
内存: %s
磁盘: %s

Bot 状态
实例: @%s
运行时间: %s
Goroutine: %d
下载目录: %s
活跃任务: %d
%s

网易云账号
%s

进程内存
Alloc: %s
Sys: %s`,
		hostname,
		runtime.GOOS,
		runtime.GOARCH,
		cpuModelText(),
		runtime.NumCPU(),
		systemMemoryText(),
		diskUsageText(downloadDir),
		botName,
		formatDuration(time.Since(botStartedAt)),
		runtime.NumGoroutine(),
		downloadDir,
		len(activeLines),
		activeText,
		musicUStatusText(),
		formatBytes(mem.Alloc),
		formatBytes(mem.Sys),
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, status)
	msg.ReplyToMessageID = message.MessageID
	_, err := bot.Send(msg)
	return err
}

func musicUStatusText() string {
	if config["MUSIC_U"] == "" {
		return "MUSIC_U: 未配置"
	}
	loginStatus, err := api.GetLoginStatus(data)
	if err != nil {
		return "MUSIC_U: 探测失败（" + err.Error() + "）"
	}
	userID := loginStatus.Profile.UserId
	nickname := loginStatus.Profile.Nickname
	if userID == 0 {
		userID = loginStatus.Account.Id
	}
	if nickname == "" {
		nickname = loginStatus.Account.UserName
	}
	if userID == 0 {
		return fmt.Sprintf("MUSIC_U: 可能无效（code=%d）", loginStatus.Code)
	}

	return fmt.Sprintf("MUSIC_U: 有效\n账号: %s (%d)\n%s", nickname, userID, probeVipByPaidSong())
}

func diskUsageText(dir string) string {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return fmt.Sprintf("读取失败: %v", err)
	}
	total := uint64(stat.Blocks) * uint64(stat.Bsize)
	free := uint64(stat.Bavail) * uint64(stat.Bsize)
	used := total - free
	percent := 0.0
	if total > 0 {
		percent = float64(used) * 100 / float64(total)
	}
	return fmt.Sprintf("%s / %s (%.1f%% 已用)", formatBytes(used), formatBytes(total), percent)
}

func systemMemoryText() string {
	total, available := readProcMemInfo()
	if total == 0 {
		return "当前系统不支持读取"
	}
	used := total - available
	percent := 0.0
	if total > 0 {
		percent = float64(used) * 100 / float64(total)
	}
	return fmt.Sprintf("%s / %s (%.1f%% 已用)", formatBytes(used), formatBytes(total), percent)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d秒", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d分%d秒", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d小时%d分", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%d天%d小时", int(d.Hours())/24, int(d.Hours())%24)
}

func formatBytes[T ~uint64 | ~int64](bytes T) string {
	value := float64(bytes)
	units := []string{"B", "KB", "MB", "GB", "TB"}
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%.0f%s", value, units[unit])
	}
	return fmt.Sprintf("%.2f%s", value, units[unit])
}
