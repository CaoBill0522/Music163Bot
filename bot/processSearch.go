package bot

import (
	"fmt"
	"strconv"

	"github.com/XiaoMengXinX/Music163Api-Go/api"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func processSearchCommand(message tgbotapi.Message, bot *tgbotapi.BotAPI) (err error) {
	keyword := message.CommandArguments()
	if keyword == "" {
		setSearchWait(message)
		msg := tgbotapi.NewMessage(message.Chat.ID, inputNextKeyword)
		msg.ReplyToMessageID = message.MessageID
		_, err = bot.Send(msg)
		return err
	}
	return processSearch(message, keyword, bot)
}

func processSearchMP3Command(message tgbotapi.Message, bot *tgbotapi.BotAPI) (err error) {
	keyword := message.CommandArguments()
	if keyword == "" {
		setSearchMP3Wait(message)
		msg := tgbotapi.NewMessage(message.Chat.ID, inputNextMP3)
		msg.ReplyToMessageID = message.MessageID
		_, err = bot.Send(msg)
		return err
	}
	return processSearchMP3(message, keyword, bot)
}

func processSearch(message tgbotapi.Message, keyword string, bot *tgbotapi.BotAPI) (err error) {
	return processSearchWithCallback(message, keyword, "download", bot)
}

func processSearchMP3(message tgbotapi.Message, keyword string, bot *tgbotapi.BotAPI) (err error) {
	return processSearchWithCallback(message, keyword, "downloadmp3", bot)
}

func processSearchWithCallback(message tgbotapi.Message, keyword, callbackPrefix string, bot *tgbotapi.BotAPI) (err error) {
	var msgResult tgbotapi.Message
	if keyword == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, inputKeyword)
		msg.ReplyToMessageID = message.MessageID
		msgResult, err = bot.Send(msg)
		return err
	}
	msg := tgbotapi.NewMessage(message.Chat.ID, searching)
	msg.ReplyToMessageID = message.MessageID
	msgResult, err = bot.Send(msg)
	if err != nil {
		return err
	}
	searchResult, _ := api.SearchSong(data, api.SearchSongConfig{
		Keyword: keyword,
		Limit:   10,
	})
	if len(searchResult.Result.Songs) == 0 {
		newEditMsg := tgbotapi.NewEditMessageText(message.Chat.ID, msgResult.MessageID, noResults)
		msgResult, err = bot.Send(newEditMsg)
		return err
	}
	var inlineButton []tgbotapi.InlineKeyboardButton
	var textMessage string
	for i := 0; i < len(searchResult.Result.Songs) && i < 8; i++ {
		var songArtists string
		for i, artist := range searchResult.Result.Songs[i].Artists {
			if i == 0 {
				songArtists = artist.Name
			} else {
				songArtists = fmt.Sprintf("%s/%s", songArtists, artist.Name)
			}
		}
		inlineButton = append(inlineButton, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d", i+1), fmt.Sprintf("%s %d", callbackPrefix, searchResult.Result.Songs[i].Id)))
		textMessage = fmt.Sprintf("%s%d.「%s」 - %s\n", textMessage, i+1, searchResult.Result.Songs[i].Name, songArtists)
	}
	var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(inlineButton)
	newEditMsg := tgbotapi.NewEditMessageText(message.Chat.ID, msgResult.MessageID, textMessage)
	newEditMsg.ReplyMarkup = &numericKeyboard
	message, err = bot.Send(newEditMsg)
	if err != nil {
		return err
	}
	return err
}

func searchWaitKey(message tgbotapi.Message) string {
	if message.From == nil {
		return ""
	}
	return fmt.Sprintf("%d:%d", message.Chat.ID, message.From.ID)
}

func setSearchWait(message tgbotapi.Message) {
	key := searchWaitKey(message)
	if key == "" {
		return
	}
	searchWait.Lock()
	defer searchWait.Unlock()
	searchWait.users[key] = true
}

func clearSearchWait(message tgbotapi.Message) {
	key := searchWaitKey(message)
	if key == "" {
		return
	}
	searchWait.Lock()
	defer searchWait.Unlock()
	delete(searchWait.users, key)
}

func setSearchMP3Wait(message tgbotapi.Message) {
	key := searchWaitKey(message)
	if key == "" {
		return
	}
	searchMP3Wait.Lock()
	defer searchMP3Wait.Unlock()
	searchMP3Wait.users[key] = true
}

func clearSearchMP3Wait(message tgbotapi.Message) {
	key := searchWaitKey(message)
	if key == "" {
		return
	}
	searchMP3Wait.Lock()
	defer searchMP3Wait.Unlock()
	delete(searchMP3Wait.users, key)
}

func consumeSearchMP3Wait(message tgbotapi.Message) bool {
	if message.Text == "" {
		return false
	}
	key := searchWaitKey(message)
	if key == "" {
		return false
	}
	searchMP3Wait.Lock()
	defer searchMP3Wait.Unlock()
	if !searchMP3Wait.users[key] {
		return false
	}
	delete(searchMP3Wait.users, key)
	return true
}

func consumeSearchWait(message tgbotapi.Message) bool {
	if message.Text == "" {
		return false
	}
	key := searchWaitKey(message)
	if key == "" {
		return false
	}
	searchWait.Lock()
	defer searchWait.Unlock()
	if !searchWait.users[key] {
		return false
	}
	delete(searchWait.users, key)
	return true
}

func processCallbackDownload(args []string, updateQuery tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) (err error) {
	musicID, _ := strconv.Atoi(args[1])
	if musicID == 0 || updateQuery.Message == nil {
		return nil
	}
	callback := tgbotapi.NewCallback(updateQuery.ID, callbackText)
	_, err = bot.Request(callback)
	if err != nil {
		return err
	}
	startTask(updateQuery.Message.Chat.ID, fmt.Sprintf("下载歌曲 %d", musicID))
	defer finishTask(updateQuery.Message.Chat.ID)
	return downloadAllToServer(musicID, *updateQuery.Message, bot)
}

func processCallbackDownloadMP3(args []string, updateQuery tgbotapi.CallbackQuery, bot *tgbotapi.BotAPI) (err error) {
	musicID, _ := strconv.Atoi(args[1])
	if musicID == 0 || updateQuery.Message == nil {
		return nil
	}
	callback := tgbotapi.NewCallback(updateQuery.ID, callbackText)
	_, err = bot.Request(callback)
	if err != nil {
		return err
	}
	startTask(updateQuery.Message.Chat.ID, fmt.Sprintf("下载 MP3 歌曲 %d", musicID))
	defer finishTask(updateQuery.Message.Chat.ID)
	return downloadAllMP3ToServer(musicID, *updateQuery.Message, bot)
}
