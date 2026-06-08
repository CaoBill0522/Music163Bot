package bot

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/XiaoMengXinX/Music163Api-Go/api"
	"github.com/XiaoMengXinX/Music163Api-Go/types"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type playlistRetryTask struct {
	PlaylistID   int
	PlaylistName string
	MusicIDs     []int
	ToMP3        bool
}

type playlistArtist struct {
	Name string `json:"name"`
}

type playlistDetailData struct {
	Playlist struct {
		Name     string `json:"name"`
		TrackIDs []struct {
			ID int `json:"id"`
		} `json:"trackIds"`
		Tracks []struct {
			ID      int              `json:"id"`
			Name    string           `json:"name"`
			Ar      []playlistArtist `json:"ar"`
			Artists []playlistArtist `json:"artists"`
		} `json:"tracks"`
	} `json:"playlist"`
}

type playlistMusicItem struct {
	ID   int
	Name string
}

const (
	playlistStageChoice    = "choice"
	playlistStageSelection = "selection"
)

var playlistRetries = struct {
	sync.Mutex
	tasks map[string]playlistRetryTask
}{
	tasks: make(map[string]playlistRetryTask),
}

func processPlaylistCommand(message tgbotapi.Message, bot *tgbotapi.BotAPI) (err error) {
	input := message.CommandArguments()
	if input == "" {
		setPlaylistWait(message)
		msg := tgbotapi.NewMessage(message.Chat.ID, inputNextPlaylist)
		msg.ReplyToMessageID = message.MessageID
		_, err = bot.Send(msg)
		return err
	}
	return processPlaylistInput(message, input, false, bot)
}

func processPlaylistMP3Command(message tgbotapi.Message, bot *tgbotapi.BotAPI) (err error) {
	input := message.CommandArguments()
	if input == "" {
		setPlaylistMP3Wait(message)
		msg := tgbotapi.NewMessage(message.Chat.ID, inputNextPlaylist)
		msg.ReplyToMessageID = message.MessageID
		_, err = bot.Send(msg)
		return err
	}
	return processPlaylistInput(message, input, true, bot)
}

func processPlaylistInput(message tgbotapi.Message, input string, toMP3 bool, bot *tgbotapi.BotAPI) error {
	if strings.Contains(input, "163cn.tv") || strings.Contains(input, "163cn.link") {
		input = getRedirectUrl(input)
	}
	playlistID := parsePlaylistID(input)
	if playlistID == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "未识别到网易云歌单链接")
		msg.ReplyToMessageID = message.MessageID
		_, err := bot.Send(msg)
		return err
	}
	return downloadPlaylistFromMessageMode(playlistID, message, toMP3, bot)
}

func downloadPlaylistFromMessage(playlistID int, message tgbotapi.Message, bot *tgbotapi.BotAPI) (err error) {
	return downloadPlaylistFromMessageMode(playlistID, message, false, bot)
}

func downloadPlaylistFromMessageMode(playlistID int, message tgbotapi.Message, toMP3 bool, bot *tgbotapi.BotAPI) (err error) {
	msg := tgbotapi.NewMessage(message.Chat.ID, fetchingPlaylist)
	msg.ReplyToMessageID = message.MessageID
	msgResult, err := bot.Send(msg)
	if err != nil {
		return err
	}
	return askPlaylistDownloadMode(playlistID, message, msgResult, toMP3, bot)
}

func downloadPlaylistToServer(playlistID int, message tgbotapi.Message, bot *tgbotapi.BotAPI) error {
	return askPlaylistDownloadMode(playlistID, message, message, false, bot)
}

func askPlaylistDownloadMode(playlistID int, userMessage tgbotapi.Message, statusMessage tgbotapi.Message, toMP3 bool, bot *tgbotapi.BotAPI) error {
	playlistName, items, err := getPlaylistMusicItems(playlistID)
	if err != nil {
		editPlaylistMessage(statusMessage, bot, fmt.Sprintf("获取歌单失败\n%v", err))
		return err
	}
	if len(items) == 0 {
		err = fmt.Errorf(playlistEmpty)
		editPlaylistMessage(statusMessage, bot, err.Error())
		return err
	}

	musicIDs := make([]int, 0, len(items))
	for _, item := range items {
		musicIDs = append(musicIDs, item.ID)
	}

	setPlaylistSelectionWait(userMessage, playlistSelectionTask{
		PlaylistID:   playlistID,
		PlaylistName: playlistName,
		MusicIDs:     musicIDs,
		Items:        items,
		Stage:        playlistStageChoice,
		ToMP3:        toMP3,
	})

	title := playlistTitle(playlistName, playlistID)
	err = editPlaylistMessage(statusMessage, bot, fmt.Sprintf("%s\n共 %d 首歌曲", title, len(items)))
	if err != nil {
		return err
	}
	err = showPlaylistSongList(statusMessage, playlistSelectionTask{PlaylistID: playlistID, PlaylistName: playlistName, Items: items}, bot)
	if err != nil {
		return err
	}
	prompt := tgbotapi.NewMessage(statusMessage.Chat.ID, inputPlaylistMode)
	_, err = bot.Send(prompt)
	return err
}

func processPlaylistSelection(message tgbotapi.Message, input string, bot *tgbotapi.BotAPI) error {
	task, ok := takePlaylistSelectionTask(message)
	if !ok {
		msg := tgbotapi.NewMessage(message.Chat.ID, "当前没有待选择的歌单")
		msg.ReplyToMessageID = message.MessageID
		_, err := bot.Send(msg)
		return err
	}

	if task.Stage == playlistStageChoice {
		return processPlaylistModeChoice(message, input, task, bot)
	}

	selectedIndexes, err := parsePlaylistSelection(input, len(task.MusicIDs))
	if err != nil {
		setPlaylistSelectionWait(message, task)
		msg := tgbotapi.NewMessage(message.Chat.ID, err.Error()+"\n"+inputPlaylistPick)
		msg.ReplyToMessageID = message.MessageID
		_, sendErr := bot.Send(msg)
		return sendErr
	}

	musicIDs := make([]int, 0, len(selectedIndexes))
	for _, index := range selectedIndexes {
		musicIDs = append(musicIDs, task.MusicIDs[index-1])
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("已选择 %d 首歌曲，开始下载", len(musicIDs)))
	msg.ReplyToMessageID = message.MessageID
	msgResult, err := bot.Send(msg)
	if err != nil {
		return err
	}
	return downloadPlaylistMusicIDsToServer(task.PlaylistID, task.PlaylistName, musicIDs, task.ToMP3, msgResult, bot)
}

func processPlaylistModeChoice(message tgbotapi.Message, input string, task playlistSelectionTask, bot *tgbotapi.BotAPI) error {
	choice := strings.TrimSpace(strings.ToLower(input))
	switch choice {
	case "all", "*", "全部", "全选", "全部下载":
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("已选择全部 %d 首歌曲，开始下载", len(task.MusicIDs)))
		msg.ReplyToMessageID = message.MessageID
		msgResult, err := bot.Send(msg)
		if err != nil {
			return err
		}
		return downloadPlaylistMusicIDsToServer(task.PlaylistID, task.PlaylistName, task.MusicIDs, task.ToMP3, msgResult, bot)
	case "part", "partial", "部分", "下载部分", "选择部分":
		task.Stage = playlistStageSelection
		setPlaylistSelectionWait(message, task)
		msg := tgbotapi.NewMessage(message.Chat.ID, inputPlaylistPick)
		msg.ReplyToMessageID = message.MessageID
		_, err := bot.Send(msg)
		return err
	default:
		setPlaylistSelectionWait(message, task)
		msg := tgbotapi.NewMessage(message.Chat.ID, "未识别选择\n"+inputPlaylistMode)
		msg.ReplyToMessageID = message.MessageID
		_, err := bot.Send(msg)
		return err
	}
}

func showPlaylistSongList(message tgbotapi.Message, task playlistSelectionTask, bot *tgbotapi.BotAPI) error {
	listLines := make([]string, 0, len(task.Items))
	for i, item := range task.Items {
		listLines = append(listLines, fmt.Sprintf("%d. %s", i+1, item.Name))
	}
	title := playlistTitle(task.PlaylistName, task.PlaylistID)
	return sendChunkedMessage(message.Chat.ID, message.MessageID, title+"\n", listLines, bot)
}

func downloadPlaylistMusicIDsToServer(playlistID int, playlistName string, musicIDs []int, toMP3 bool, message tgbotapi.Message, bot *tgbotapi.BotAPI) error {
	chatID := message.Chat.ID
	taskName := playlistTitle(playlistName, playlistID)
	if toMP3 {
		taskName += " MP3"
	}
	startTask(chatID, taskName)
	defer finishTask(chatID)

	successCount := 0
	failedCount := 0
	skippedCount := 0
	failedItems := make([]failedPlaylistItem, 0)
	totalCount := len(musicIDs)
	title := playlistTitle(playlistName, playlistID)
	var err error

	for i, musicID := range musicIDs {
		if isTaskStopped(chatID) {
			return editPlaylistMessage(message, bot, fmt.Sprintf("%s\n%s\n进度: %d/%d\n成功: %d 跳过: %d 失败: %d", title, playlistCanceled, i, totalCount, successCount, skippedCount, failedCount))
		}
		if i > 0 {
			time.Sleep(2 * time.Second)
			if isTaskStopped(chatID) {
				return editPlaylistMessage(message, bot, fmt.Sprintf("%s\n%s\n进度: %d/%d\n成功: %d 跳过: %d 失败: %d", title, playlistCanceled, i, totalCount, successCount, skippedCount, failedCount))
			}
		}
		songName := getSongDisplayName(musicID)
		editPlaylistMessage(message, bot, fmt.Sprintf("%s\n进度: %d/%d\n成功: %d 跳过: %d 失败: %d\n正在处理: %s", title, i+1, totalCount, successCount, skippedCount, failedCount, songName))
		exists, existingPath, infoErr := playlistSongExists(musicID, toMP3)
		if infoErr == nil && exists {
			skippedCount++
			editPlaylistMessage(message, bot, fmt.Sprintf("%s\n进度: %d/%d\n成功: %d 跳过: %d 失败: %d\n已存在，跳过: %s\n%s", title, i+1, totalCount, successCount, skippedCount, failedCount, songName, existingPath))
			continue
		}
		if toMP3 {
			err = downloadAllMP3ToServer(musicID, message, bot)
		} else {
			err = downloadAllToServer(musicID, message, bot)
		}
		if err != nil {
			failedCount++
			failedItems = append(failedItems, failedPlaylistItem{
				ID:    musicID,
				Name:  songName,
				Error: err.Error(),
			})
			editPlaylistMessage(message, bot, fmt.Sprintf("%s\n进度: %d/%d\n成功: %d 跳过: %d 失败: %d\n刚才失败: %s", title, i+1, totalCount, successCount, skippedCount, failedCount, songName))
			continue
		}
		successCount++
		editPlaylistMessage(message, bot, fmt.Sprintf("%s\n进度: %d/%d\n成功: %d 跳过: %d 失败: %d\n刚才完成: %s", title, i+1, totalCount, successCount, skippedCount, failedCount, songName))
		if isTaskStopped(chatID) {
			return editPlaylistMessage(message, bot, fmt.Sprintf("%s\n%s\n进度: %d/%d\n成功: %d 跳过: %d 失败: %d", title, playlistCanceled, i+1, totalCount, successCount, skippedCount, failedCount))
		}
	}

	status := fmt.Sprintf("%s\n歌单处理完成\n总数: %d\n成功: %d\n跳过: %d\n失败: %d", title, totalCount, successCount, skippedCount, failedCount)
	if len(failedItems) > 0 {
		status = fmt.Sprintf("%s\n失败歌曲（最多显示10首）:\n%s\n是否需要重新下载失败歌曲？", status, formatFailedPlaylistItems(failedItems, 10))
		retryKey := savePlaylistRetryTask(playlistID, playlistName, failedItems, toMP3)
		return editPlaylistMessageWithRetry(message, bot, status, retryKey)
	}
	return editPlaylistMessage(message, bot, status)
}

func retryPlaylistFailed(args []string, updateQuery tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) error {
	if len(args) < 2 || updateQuery.Message == nil {
		return nil
	}
	callback := tgbotapi.NewCallback(updateQuery.ID, callbackText)
	_, err := bot.Request(callback)
	if err != nil {
		return err
	}

	task, ok := takePlaylistRetryTask(args[1])
	if !ok || len(task.MusicIDs) == 0 {
		return editPlaylistMessage(*updateQuery.Message, bot, "没有可重试的失败歌曲，可能已经重试过或服务已重启")
	}
	editPlaylistMessage(*updateQuery.Message, bot, "开始重新下载失败歌曲...")
	return downloadPlaylistMusicIDsToServer(task.PlaylistID, task.PlaylistName, task.MusicIDs, task.ToMP3, *updateQuery.Message, bot)
}

func getPlaylistMusicIDs(playlistID int) (string, []int, error) {
	playlistName, items, err := getPlaylistMusicItems(playlistID)
	if err != nil {
		return "", nil, err
	}
	musicIDs := make([]int, 0, len(items))
	for _, item := range items {
		musicIDs = append(musicIDs, item.ID)
	}
	return playlistName, musicIDs, nil
}

func playlistSongExists(musicID int, toMP3 bool) (bool, string, error) {
	if !toMP3 {
		return localSongExists(musicID)
	}
	info, _, err := getDownloadSongInfo(musicID)
	if err != nil {
		return false, "", err
	}
	info.FileExt = "mp3"
	return localSongInfoExistsInDir(info, mp3Dir)
}

func getPlaylistMusicItems(playlistID int) (string, []playlistMusicItem, error) {
	b := api.NewBatch(
		api.BatchAPI{
			Key:  api.PlaylistDetailAPI,
			Json: api.CreatePlaylistDetailReqJson(playlistID),
		},
	)
	if doErr := b.Do(data).Error; doErr != nil {
		return "", nil, doErr
	}
	_, result := b.Parse()

	var detail playlistDetailData
	if err := json.Unmarshal([]byte(result[api.PlaylistDetailAPI]), &detail); err != nil {
		return "", nil, err
	}

	items := make([]playlistMusicItem, 0, len(detail.Playlist.TrackIDs))
	seen := make(map[int]bool)
	for _, track := range detail.Playlist.Tracks {
		if track.ID == 0 || seen[track.ID] {
			continue
		}
		seen[track.ID] = true
		items = append(items, playlistMusicItem{
			ID:   track.ID,
			Name: playlistTrackName(track.ID, track.Name, track.Ar, track.Artists),
		})
	}
	for _, track := range detail.Playlist.TrackIDs {
		if track.ID == 0 || seen[track.ID] {
			continue
		}
		seen[track.ID] = true
		items = append(items, playlistMusicItem{
			ID:   track.ID,
			Name: getSongDisplayName(track.ID),
		})
	}
	return detail.Playlist.Name, items, nil
}

func playlistTrackName(musicID int, name string, ar []playlistArtist, artists []playlistArtist) string {
	if strings.TrimSpace(name) == "" {
		return getSongDisplayName(musicID)
	}
	useArtists := ar
	if len(useArtists) == 0 {
		useArtists = artists
	}
	artistNames := make([]string, 0, len(useArtists))
	for _, artist := range useArtists {
		if strings.TrimSpace(artist.Name) != "" {
			artistNames = append(artistNames, artist.Name)
		}
	}
	if len(artistNames) == 0 {
		return name
	}
	return strings.Join(artistNames, "/") + " - " + name
}

func parsePlaylistSelection(input string, total int) ([]int, error) {
	normalized := strings.TrimSpace(strings.ToLower(input))
	if normalized == "" {
		return nil, fmt.Errorf("请输入要下载的歌曲序号")
	}
	if normalized == "all" || normalized == "*" || normalized == "全部" || normalized == "全选" || normalized == "全部下载" {
		indexes := make([]int, 0, total)
		for i := 1; i <= total; i++ {
			indexes = append(indexes, i)
		}
		return indexes, nil
	}

	selected := make(map[int]bool)
	parts := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == ',' || r == '，' || r == ' ' || r == '\n' || r == '\t'
	})
	for _, part := range parts {
		if part == "" {
			continue
		}
		bounds := strings.Split(part, "-")
		if len(bounds) == 1 {
			index, err := strconv.Atoi(bounds[0])
			if err != nil || index < 1 || index > total {
				return nil, fmt.Errorf("无效序号: %s", part)
			}
			selected[index] = true
			continue
		}
		if len(bounds) != 2 {
			return nil, fmt.Errorf("无效区间: %s", part)
		}
		start, startErr := strconv.Atoi(bounds[0])
		end, endErr := strconv.Atoi(bounds[1])
		if startErr != nil || endErr != nil || start < 1 || end < 1 || start > total || end > total || start > end {
			return nil, fmt.Errorf("无效区间: %s", part)
		}
		for i := start; i <= end; i++ {
			selected[i] = true
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("未识别到要下载的歌曲序号")
	}
	indexes := make([]int, 0, len(selected))
	for index := range selected {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	return indexes, nil
}

func playlistTitle(name string, playlistID int) string {
	if strings.TrimSpace(name) == "" {
		return fmt.Sprintf("歌单: %d", playlistID)
	}
	return fmt.Sprintf("歌单: %s", name)
}

func editPlaylistMessage(message tgbotapi.Message, bot *tgbotapi.BotAPI, text string) error {
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, text)
	_, err := bot.Send(editMsg)
	return err
}

func editPlaylistMessageWithRetry(message tgbotapi.Message, bot *tgbotapi.BotAPI, text, retryKey string) error {
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, text)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("重新下载失败歌曲", fmt.Sprintf("playlist_retry %s", retryKey)),
		),
	)
	editMsg.ReplyMarkup = &keyboard
	_, err := bot.Send(editMsg)
	return err
}

type failedPlaylistItem struct {
	ID    int
	Name  string
	Error string
}

func formatFailedPlaylistItems(items []failedPlaylistItem, limit int) string {
	lines := make([]string, 0, limit)
	for i, item := range items {
		if i >= limit {
			break
		}
		lines = append(lines, fmt.Sprintf("%d. %s\n原因: %s", i+1, item.Name, item.Error))
	}
	if len(items) > limit {
		lines = append(lines, fmt.Sprintf("还有 %d 首未显示", len(items)-limit))
	}
	return strings.Join(lines, "\n")
}

func savePlaylistRetryTask(playlistID int, playlistName string, failedItems []failedPlaylistItem, toMP3 bool) string {
	musicIDs := make([]int, 0, len(failedItems))
	for _, item := range failedItems {
		musicIDs = append(musicIDs, item.ID)
	}
	key := fmt.Sprintf("%x", time.Now().UnixNano())
	playlistRetries.Lock()
	defer playlistRetries.Unlock()
	playlistRetries.tasks[key] = playlistRetryTask{
		PlaylistID:   playlistID,
		PlaylistName: playlistName,
		MusicIDs:     musicIDs,
		ToMP3:        toMP3,
	}
	return key
}

func takePlaylistRetryTask(key string) (playlistRetryTask, bool) {
	playlistRetries.Lock()
	defer playlistRetries.Unlock()
	task, ok := playlistRetries.tasks[key]
	if ok {
		delete(playlistRetries.tasks, key)
	}
	return task, ok
}

func getSongDisplayName(musicID int) string {
	info, err := getSongBasicInfo(musicID)
	if err != nil {
		return fmt.Sprintf("歌曲ID %d", musicID)
	}
	return fmt.Sprintf("%s - %s", info.SongArtists, info.SongName)
}

func getSongBasicInfo(musicID int) (songInfo, error) {
	b := api.NewBatch(
		api.BatchAPI{
			Key:  api.SongDetailAPI,
			Json: api.CreateSongDetailReqJson([]int{musicID}),
		},
	)
	if doErr := b.Do(data).Error; doErr != nil {
		return songInfo{}, doErr
	}
	_, result := b.Parse()

	var detail types.SongsDetailData
	if err := json.Unmarshal([]byte(result[api.SongDetailAPI]), &detail); err != nil {
		return songInfo{}, err
	}
	if len(detail.Songs) == 0 {
		return songInfo{}, fmt.Errorf(fetchInfoFailed)
	}
	return songInfo{
		MusicID:     musicID,
		SongName:    detail.Songs[0].Name,
		SongArtists: parseArtist(detail.Songs[0]),
		SongAlbum:   detail.Songs[0].Al.Name,
		PicURL:      detail.Songs[0].Al.PicUrl,
	}, nil
}

func setPlaylistWait(message tgbotapi.Message) {
	key := searchWaitKey(message)
	if key == "" {
		return
	}
	playlistWait.Lock()
	defer playlistWait.Unlock()
	playlistWait.users[key] = true
}

func clearPlaylistWait(message tgbotapi.Message) {
	key := searchWaitKey(message)
	if key == "" {
		return
	}
	playlistWait.Lock()
	defer playlistWait.Unlock()
	delete(playlistWait.users, key)
}

func setPlaylistMP3Wait(message tgbotapi.Message) {
	key := searchWaitKey(message)
	if key == "" {
		return
	}
	playlistMP3Wait.Lock()
	defer playlistMP3Wait.Unlock()
	playlistMP3Wait.users[key] = true
}

func clearPlaylistMP3Wait(message tgbotapi.Message) {
	key := searchWaitKey(message)
	if key == "" {
		return
	}
	playlistMP3Wait.Lock()
	defer playlistMP3Wait.Unlock()
	delete(playlistMP3Wait.users, key)
}

func consumePlaylistMP3Wait(message tgbotapi.Message) bool {
	if message.Text == "" {
		return false
	}
	key := searchWaitKey(message)
	if key == "" {
		return false
	}
	playlistMP3Wait.Lock()
	defer playlistMP3Wait.Unlock()
	if !playlistMP3Wait.users[key] {
		return false
	}
	delete(playlistMP3Wait.users, key)
	return true
}

func setPlaylistSelectionWait(message tgbotapi.Message, task playlistSelectionTask) {
	key := searchWaitKey(message)
	if key == "" {
		return
	}
	playlistSelectionWait.Lock()
	defer playlistSelectionWait.Unlock()
	playlistSelectionWait.users[key] = task
}

func clearPlaylistSelectionWait(message tgbotapi.Message) {
	key := searchWaitKey(message)
	if key == "" {
		return
	}
	playlistSelectionWait.Lock()
	defer playlistSelectionWait.Unlock()
	delete(playlistSelectionWait.users, key)
}

func consumePlaylistSelectionWait(message tgbotapi.Message) bool {
	if message.Text == "" {
		return false
	}
	key := searchWaitKey(message)
	if key == "" {
		return false
	}
	playlistSelectionWait.Lock()
	defer playlistSelectionWait.Unlock()
	_, ok := playlistSelectionWait.users[key]
	return ok
}

func takePlaylistSelectionTask(message tgbotapi.Message) (playlistSelectionTask, bool) {
	key := searchWaitKey(message)
	if key == "" {
		return playlistSelectionTask{}, false
	}
	playlistSelectionWait.Lock()
	defer playlistSelectionWait.Unlock()
	task, ok := playlistSelectionWait.users[key]
	if ok {
		delete(playlistSelectionWait.users, key)
	}
	return task, ok
}

func consumePlaylistWait(message tgbotapi.Message) bool {
	if message.Text == "" {
		return false
	}
	key := searchWaitKey(message)
	if key == "" {
		return false
	}
	playlistWait.Lock()
	defer playlistWait.Unlock()
	if !playlistWait.users[key] {
		return false
	}
	delete(playlistWait.users, key)
	return true
}
