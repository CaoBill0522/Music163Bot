package bot

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func downloadAllToServer(musicID int, message tgbotapi.Message, bot *tgbotapi.BotAPI) (err error) {
	if exists, existingPath, existsErr := localSongExists(musicID); existsErr == nil && exists {
		editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, fmt.Sprintf("歌曲已存在，已跳过\n%s", existingPath))
		_, err = bot.Send(editMsg)
		return err
	}
	musicPath, err := downloadMusicToServerPath(musicID, message, bot)
	if err != nil {
		return err
	}

	lyricPath, lyricErr := downloadLyricToServer(musicID)
	status := fmt.Sprintf("全部下载完成\n音频: %s", musicPath)
	if lyricErr != nil {
		status = fmt.Sprintf("%s\n歌词: 下载失败（%v）", status, lyricErr)
	} else {
		status = fmt.Sprintf("%s\n歌词: %s", status, lyricPath)
		editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, fmt.Sprintf("%s\n正在补齐封面和歌曲信息并嵌入歌词...", status))
		_, _ = bot.Send(editMsg)
		info, _, infoErr := getDownloadSongInfo(musicID)
		if infoErr != nil {
			status = fmt.Sprintf("%s\n嵌入元数据: 失败（%v）", status, infoErr)
		} else {
			embedErr := embedMetadataIntoAudio(musicPath, lyricPath, info)
			if embedErr != nil {
				status = fmt.Sprintf("%s\n嵌入元数据: 失败（%v）", status, embedErr)
			} else {
				_ = os.Remove(lyricPath)
				status = fmt.Sprintf("%s\n嵌入元数据: 完成\n歌词文件: 已删除", status)
			}
		}
	}
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, status)
	_, err = bot.Send(editMsg)
	return err
}

func downloadAllMP3ToServer(musicID int, message tgbotapi.Message, bot *tgbotapi.BotAPI) (err error) {
	info, _, infoErr := getDownloadSongInfo(musicID)
	if infoErr != nil {
		editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, infoErr.Error())
		_, _ = bot.Send(editMsg)
		return infoErr
	}
	mp3Info := info
	mp3Info.FileExt = "mp3"
	if exists, existingPath, existsErr := localSongInfoExistsInDir(mp3Info, mp3Dir); existsErr == nil && exists {
		editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, fmt.Sprintf("歌曲已存在，已跳过\n%s", existingPath))
		_, err = bot.Send(editMsg)
		return err
	}

	musicPath, err := downloadMusicToServerPath(musicID, message, bot)
	if err != nil {
		return err
	}

	err = os.MkdirAll(mp3Dir, 0755)
	if err != nil {
		editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, fmt.Sprintf("创建 MP3 目录失败\n%v", err))
		_, _ = bot.Send(editMsg)
		return err
	}
	mp3Path := uniqueFilePath(filepath.Join(mp3Dir, safeMusicFileName(mp3Info)))
	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, fmt.Sprintf("正在转换 MP3...\n%s", filepath.Base(mp3Path)))
	_, _ = bot.Send(editMsg)
	if strings.EqualFold(filepath.Ext(musicPath), ".mp3") {
		err = copyFile(musicPath, mp3Path)
	} else {
		err = convertAudioToMP3(musicPath, mp3Path)
	}
	if err != nil {
		editMsg = tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, fmt.Sprintf("MP3 转换失败\n%v", err))
		_, _ = bot.Send(editMsg)
		return err
	}

	lyricPath, lyricErr := downloadLyricToServer(musicID)
	status := fmt.Sprintf("MP3 下载完成\n音频: %s", mp3Path)
	if lyricErr != nil {
		status = fmt.Sprintf("%s\n歌词: 下载失败（%v）", status, lyricErr)
	} else {
		status = fmt.Sprintf("%s\n歌词: %s", status, lyricPath)
		editMsg = tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, fmt.Sprintf("%s\n正在嵌入歌曲信息和歌词...", status))
		_, _ = bot.Send(editMsg)
		embedErr := embedMetadataIntoAudio(mp3Path, lyricPath, mp3Info)
		if embedErr != nil {
			status = fmt.Sprintf("%s\n嵌入元数据: 失败（%v）", status, embedErr)
		} else {
			_ = os.Remove(lyricPath)
			status = fmt.Sprintf("%s\n嵌入元数据: 完成\n歌词文件: 已删除", status)
		}
	}
	editMsg = tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, status)
	_, err = bot.Send(editMsg)
	return err
}
