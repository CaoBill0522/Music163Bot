package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/XiaoMengXinX/Music163Api-Go/api"
	"github.com/XiaoMengXinX/Music163Api-Go/types"
	downloader "github.com/XiaoMengXinX/SimpleDownloader"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type downloadSongURL struct {
	Url  string
	Md5  string
	Size int
}

type songInfo struct {
	MusicID     int
	SongName    string
	SongArtists string
	SongAlbum   string
	PicURL      string
	FileExt     string
	MusicSize   int
	BitRate     int
	Duration    int
}

func downloadMusicToServerPath(musicID int, message tgbotapi.Message, bot *tgbotapi.BotAPI) (filePath string, err error) {
	return downloadMusicToServerPathWithContext(taskContext(message.Chat.ID), musicID, message, bot)
}

func downloadMusicToServerPathWithContext(ctx context.Context, musicID int, message tgbotapi.Message, bot *tgbotapi.BotAPI) (filePath string, err error) {
	d := downloader.NewDownloader().SetSavePath(downloadDir).SetBreakPoint(true)
	if downloaderTimeout > 0 {
		d.SetTimeOut(time.Duration(downloaderTimeout) * time.Second)
	} else {
		d.SetTimeOut(60 * time.Second)
	}

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()

	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, fetchInfo)
	err = sendNonCritical(bot, editMsg)
	if err != nil {
		return "", err
	}

	songInfo, songURL, err := getDownloadSongInfo(musicID)
	if err != nil {
		editMsg = tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, err.Error())
		_ = sendNonCritical(bot, editMsg)
		return "", err
	}

	err = os.MkdirAll(downloadDir, 0755)
	if err != nil {
		editMsg = tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, fmt.Sprintf("创建下载目录失败\n%v", err))
		_ = sendNonCritical(bot, editMsg)
		return "", err
	}

	fileName := safeMusicFileName(songInfo)
	tmpFileName := fmt.Sprintf("%d-%s", time.Now().UnixMicro(), cleanURLPathBase(songURL.Url))
	task, err := d.NewDownloadTask(songURL.Url)
	if err != nil {
		editMsg = tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, fmt.Sprintf("创建下载任务失败\n%v", err))
		_ = sendNonCritical(bot, editMsg)
		return "", err
	}
	hostReplacer := strings.NewReplacer("m8.", "m7.", "m801.", "m701.", "m804.", "m701.", "m704.", "m701.")
	task.ReplaceHostName(hostReplacer.Replace(task.GetHostName())).ForceHttps().ForceMultiThread()

	editMsg = tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, fmt.Sprintf(musicInfoMsg+downloading, songInfo.SongName, songInfo.SongAlbum, songInfo.FileExt, float64(songInfo.MusicSize)/1024/1024))
	err = sendNonCritical(bot, editMsg)
	if err != nil {
		return "", err
	}

	errCh := task.SetFileName(tmpFileName).DownloadWithChannel()
	err = updateDownloadMessage(ctx, task, errCh, message, songInfo, downloading, bot)
	if err != nil && config["ReverseProxy"] != "" {
		ch := task.WithResolvedIpOnHost(config["ReverseProxy"]).DownloadWithChannel()
		err = updateDownloadMessage(ctx, task, ch, message, songInfo, redownloading, bot)
	}
	if err != nil {
		task.CleanTempFiles()
		editMsg = tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, fmt.Sprintf("下载失败\n%v", err))
		_ = sendNonCritical(bot, editMsg)
		return "", err
	}

	tmpPath := filepath.Join(downloadDir, tmpFileName)
	if songURL.Md5 != "" {
		verified, verifyErr := verifyMD5(tmpPath, songURL.Md5)
		if verifyErr != nil || !verified {
			_ = os.Remove(tmpPath)
			err = fmt.Errorf("%s\n%s", md5VerFailed, retryLater)
			editMsg = tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, err.Error())
			_ = sendNonCritical(bot, editMsg)
			return "", err
		}
	}

	finalPath := filepath.Join(downloadDir, fileName)
	finalPath = uniqueFilePath(finalPath)
	err = os.Rename(tmpPath, finalPath)
	if err != nil {
		editMsg = tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, fmt.Sprintf("保存文件失败\n%v", err))
		_ = sendNonCritical(bot, editMsg)
		return "", err
	}

	editMsg = tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, fmt.Sprintf(musicInfoMsg+downloadDone, songInfo.SongName, songInfo.SongAlbum, songInfo.FileExt, float64(songInfo.MusicSize)/1024/1024, finalPath))
	return finalPath, sendNonCritical(bot, editMsg)
}

func getDownloadSongInfo(musicID int) (songInfo, downloadSongURL, error) {
	b := api.NewBatch(
		api.BatchAPI{
			Key:  api.SongDetailAPI,
			Json: api.CreateSongDetailReqJson([]int{musicID}),
		},
		api.BatchAPI{
			Key:  api.SongUrlAPI,
			Json: api.CreateSongURLJson(api.SongURLConfig{Ids: []int{musicID}}),
		},
	)
	if doErr := b.Do(data).Error; doErr != nil {
		return songInfo{}, downloadSongURL{}, doErr
	}
	_, result := b.Parse()

	var songDetail types.SongsDetailData
	_ = json.Unmarshal([]byte(result[api.SongDetailAPI]), &songDetail)

	var songURL types.SongsURLData
	_ = json.Unmarshal([]byte(result[api.SongUrlAPI]), &songURL)

	if len(songDetail.Songs) == 0 || len(songURL.Data) == 0 {
		return songInfo{}, downloadSongURL{}, errors.New(fetchInfoFailed)
	}
	if songURL.Data[0].Url == "" {
		return songInfo{}, downloadSongURL{}, errors.New(getUrlFailed)
	}

	info := songInfo{
		MusicID:     musicID,
		Duration:    songDetail.Songs[0].Dt / 1000,
		SongName:    songDetail.Songs[0].Name,
		SongArtists: parseArtist(songDetail.Songs[0]),
		SongAlbum:   songDetail.Songs[0].Al.Name,
		PicURL:      songDetail.Songs[0].Al.PicUrl,
		MusicSize:   songURL.Data[0].Size,
	}
	baseURL := songURL.Data[0].Url
	if queryIndex := strings.Index(baseURL, "?"); queryIndex != -1 {
		baseURL = baseURL[:queryIndex]
	}
	switch path.Ext(path.Base(baseURL)) {
	case ".mp3":
		info.FileExt = "mp3"
	case ".flac":
		info.FileExt = "flac"
	default:
		info.FileExt = "mp3"
	}
	if info.Duration > 0 {
		info.BitRate = 8 * songURL.Data[0].Size / info.Duration
	}

	urlInfo := downloadSongURL{
		Url:  songURL.Data[0].Url,
		Md5:  songURL.Data[0].Md5,
		Size: songURL.Data[0].Size,
	}
	return info, urlInfo, nil
}

func updateDownloadMessage(ctx context.Context, task *downloader.DownloadTask, ch chan error, message tgbotapi.Message, songInfo songInfo, statusText string, bot *tgbotapi.BotAPI) (err error) {
	var lastUpdateTime int64
	lastProgressTime := time.Now()
	lastWrittenBytes := task.GetWrittenBytes()
	stallTimeout := downloadStallTimeout()
	for {
		select {
		case <-ctx.Done():
			cancelDownloadTask(task)
			task.CleanTempFiles()
			logrus.Warnf("download canceled: chat=%d music=%d title=%s err=%v", message.Chat.ID, songInfo.MusicID, songInfo.SongName, ctx.Err())
			return fmt.Errorf("任务已停止")
		case err, ok := <-ch:
			if !ok {
				return nil
			}
			if err != nil {
				logrus.Warnf("download finished with error: chat=%d music=%d title=%s err=%v", message.Chat.ID, songInfo.MusicID, songInfo.SongName, err)
			}
			return err
		case <-time.After(200 * time.Millisecond):
			writtenBytes := task.GetWrittenBytes()
			fileSize := task.GetFileSize()
			if writtenBytes > lastWrittenBytes {
				lastProgressTime = time.Now()
				lastWrittenBytes = writtenBytes
			}
			if time.Since(lastProgressTime) > stallTimeout {
				cancelDownloadTask(task)
				task.CleanTempFiles()
				logrus.Warnf("download stalled: chat=%d music=%d title=%s written=%d size=%d timeout=%s", message.Chat.ID, songInfo.MusicID, songInfo.SongName, writtenBytes, fileSize, stallTimeout)
				return fmt.Errorf("下载超过 %s 没有进度，已停止当前歌曲", formatDuration(stallTimeout))
			}
			if fileSize == 0 || writtenBytes == 0 || time.Now().Unix()-lastUpdateTime < 2 {
				continue
			}
			editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, fmt.Sprintf(musicInfoMsg+statusText+downloadStatus, songInfo.SongName, songInfo.SongAlbum, songInfo.FileExt, float64(songInfo.MusicSize)/1024/1024, task.CalculateSpeed(time.Millisecond*500), float64(writtenBytes)/1024/1024, float64(fileSize)/1024/1024, (writtenBytes*100)/fileSize))
			_ = sendNonCritical(bot, editMsg)
			lastUpdateTime = time.Now().Unix()
		}
	}
}

func cancelDownloadTask(task *downloader.DownloadTask) {
	defer func() {
		_ = recover()
	}()
	task.Cancel()
}

func downloadStallTimeout() time.Duration {
	if downloaderTimeout > 0 {
		return time.Duration(downloaderTimeout) * time.Second
	}
	return 60 * time.Second
}

func safeMusicFileName(songInfo songInfo) string {
	replacer := strings.NewReplacer("/", " ", "?", " ", "*", " ", ":", " ", "|", " ", "\\", " ", "<", " ", ">", " ", "\"", " ")
	return replacer.Replace(fmt.Sprintf("%s - %s.%s", strings.ReplaceAll(songInfo.SongArtists, "/", ","), songInfo.SongName, songInfo.FileExt))
}

func localSongExists(musicID int) (bool, string, error) {
	info, _, err := getDownloadSongInfo(musicID)
	if err != nil {
		return false, "", err
	}
	return localSongInfoExists(info)
}

func localSongInfoExists(info songInfo) (bool, string, error) {
	return localSongInfoExistsInDir(info, downloadDir)
}

func localSongInfoExistsInDir(info songInfo, dir string) (bool, string, error) {
	fileName := safeMusicFileName(info)
	fullPath := filepath.Join(dir, fileName)
	if _, err := os.Stat(fullPath); err == nil {
		return true, fullPath, nil
	} else if !os.IsNotExist(err) {
		return false, "", err
	}

	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	for _, ext := range []string{".mp3", ".flac", ".m4a", ".aac", ".ogg", ".wav"} {
		candidate := filepath.Join(dir, base+ext)
		if _, err := os.Stat(candidate); err == nil {
			return true, candidate, nil
		} else if !os.IsNotExist(err) {
			return false, "", err
		}
	}
	return false, "", nil
}

func cleanURLPathBase(rawURL string) string {
	if queryIndex := strings.Index(rawURL, "?"); queryIndex != -1 {
		rawURL = rawURL[:queryIndex]
	}
	name := path.Base(rawURL)
	if name == "." || name == "/" || name == "" {
		return fmt.Sprintf("%d.download", time.Now().UnixMicro())
	}
	return name
}

func uniqueFilePath(filePath string) string {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return filePath
	}
	ext := filepath.Ext(filePath)
	base := strings.TrimSuffix(filePath, ext)
	for i := 1; ; i++ {
		candidate := base + " (" + strconv.Itoa(i) + ")" + ext
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}
