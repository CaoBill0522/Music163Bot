package bot

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/XiaoMengXinX/Music163Api-Go/utils"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// Start bot entry
func Start(conf map[string]string) (actionCode int) {
	config = conf
	defer func() {
		e := recover()
		if e != nil {
			logrus.Errorln(e)
			actionCode = 1
		}
	}()
	// 创建缓存文件夹
	dirExists(cacheDir)

	if config["MUSIC_U"] != "" {
		data = utils.RequestData{
			Cookies: []*http.Cookie{
				{
					Name:  "MUSIC_U",
					Value: config["MUSIC_U"],
				},
			},
		}
	}
	if config["BotAPI"] != "" {
		botAPI = config["BotAPI"]
	}
	if config["DownloadPath"] != "" {
		downloadDir = config["DownloadPath"]
	}
	if config["Mp3Path"] != "" {
		mp3Dir = config["Mp3Path"]
	}
	if config["FileURLBase"] != "" {
		fileURLBase = strings.TrimRight(config["FileURLBase"], "/")
	}
	if config["FileRoot"] != "" {
		fileRoot = config["FileRoot"]
	}
	if config["FilePassword"] != "" {
		filePassword = config["FilePassword"]
	}
	if config["VipProbeKeyword"] != "" {
		vipProbeKeyword = config["VipProbeKeyword"]
	}

	if downloaderTimeout, _ = strconv.Atoi(config["DownloadTimeout"]); downloaderTimeout <= 0 {
		downloaderTimeout = 60
	}

	// 设置 bot 日志接口
	err := tgbotapi.SetLogger(logrus.StandardLogger())
	if err != nil {
		logrus.Errorln(err)
		return 1
	}
	// 配置 token、api、debug
	bot, err = tgbotapi.NewBotAPIWithAPIEndpoint(config["BOT_TOKEN"], botAPI+"/bot%s/%s")
	if err != nil {
		logrus.Errorln(err)
		return 1
	}
	if config["BotDebug"] == "true" {
		bot.Debug = true
	}

	logrus.Printf("%s 验证成功", bot.Self.UserName)
	botName = bot.Self.UserName
	err = setBotCommands(bot)
	if err != nil {
		logrus.Errorln(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	defer bot.StopReceivingUpdates()

	for update := range updates {
		if update.Message == nil && update.CallbackQuery == nil {
			continue
		}
		switch {
		case update.Message != nil:
			updateMsg := *update.Message
			if atStr := strings.ReplaceAll(update.Message.CommandWithAt(), update.Message.Command(), ""); update.Message.Command() != "" && (atStr == "" || atStr == "@"+botName) {
				switch update.Message.Command() {
				case "start":
					go func() {
						err := processStartCommand(updateMsg, bot)
						if err != nil {
							logrus.Errorln(err)
						}
					}()
				case "stop":
					go func() {
						err := processStopCommand(updateMsg, bot)
						if err != nil {
							logrus.Errorln(err)
						}
					}()
				case "status":
					go func() {
						err := processStatusCommand(updateMsg, bot)
						if err != nil {
							logrus.Errorln(err)
						}
					}()
				case "tasks":
					go func() {
						err := processTasksCommand(updateMsg, bot)
						if err != nil {
							logrus.Errorln(err)
						}
					}()
				case "download":
					go func() {
						err := processDownloadArchiveCommand(updateMsg, bot)
						if err != nil {
							logrus.Errorln(err)
						}
					}()
				case "file":
					go func() {
						err := processFileCommand(updateMsg, bot)
						if err != nil {
							logrus.Errorln(err)
						}
					}()
				case "playlist":
					go func() {
						err := processPlaylistCommand(updateMsg, bot)
						if err != nil {
							logrus.Errorln(err)
						}
					}()
				case "playlistmp3":
					go func() {
						err := processPlaylistMP3Command(updateMsg, bot)
						if err != nil {
							logrus.Errorln(err)
						}
					}()
				case "search":
					go func() {
						err := processSearchCommand(updateMsg, bot)
						if err != nil {
							logrus.Errorln(err)
						}
					}()
				case "searchmp3":
					go func() {
						err := processSearchMP3Command(updateMsg, bot)
						if err != nil {
							logrus.Errorln(err)
						}
					}()
				}
			} else if consumeDownloadArchiveWait(updateMsg) {
				go func() {
					err := processDownloadArchiveInput(updateMsg, updateMsg.Text, bot)
					if err != nil {
						logrus.Errorln(err)
					}
				}()
			} else if isFileSession(updateMsg) {
				go func() {
					err := processFileInput(updateMsg, bot)
					if err != nil {
						logrus.Errorln(err)
					}
				}()
			} else if consumePlaylistSelectionWait(updateMsg) {
				go func() {
					err := processPlaylistSelection(updateMsg, updateMsg.Text, bot)
					if err != nil {
						logrus.Errorln(err)
					}
				}()
			} else if consumePlaylistMP3Wait(updateMsg) {
				go func() {
					err := processPlaylistInput(updateMsg, updateMsg.Text, true, bot)
					if err != nil {
						logrus.Errorln(err)
					}
				}()
			} else if consumePlaylistWait(updateMsg) {
				go func() {
					err := processPlaylistInput(updateMsg, updateMsg.Text, false, bot)
					if err != nil {
						logrus.Errorln(err)
					}
				}()
			} else if consumeSearchMP3Wait(updateMsg) {
				go func() {
					err := processSearchMP3(updateMsg, updateMsg.Text, bot)
					if err != nil {
						logrus.Errorln(err)
					}
				}()
			} else if consumeSearchWait(updateMsg) {
				go func() {
					err := processSearch(updateMsg, updateMsg.Text, bot)
					if err != nil {
						logrus.Errorln(err)
					}
				}()
			}
		case update.CallbackQuery != nil:
			updateQuery := *update.CallbackQuery
			args := strings.Split(updateQuery.Data, " ")
			if len(args) < 2 {
				continue
			}
			switch args[0] {
			case "download":
				go func() {
					err := processCallbackDownload(args, updateQuery, bot)
					if err != nil {
						logrus.Errorln(err)
					}
				}()
			case "downloadmp3":
				go func() {
					err := processCallbackDownloadMP3(args, updateQuery, bot)
					if err != nil {
						logrus.Errorln(err)
					}
				}()
			case "playlist_retry":
				go func() {
					err := retryPlaylistFailed(args, updateQuery, bot)
					if err != nil {
						logrus.Errorln(err)
					}
				}()
			}
		}
	}
	return 0
}

func setBotCommands(bot *tgbotapi.BotAPI) error {
	commands := tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{
			Command:     "start",
			Description: "展示功能菜单",
		},
		tgbotapi.BotCommand{
			Command:     "search",
			Description: "搜索并下载歌曲，自动嵌入信息和歌词",
		},
		tgbotapi.BotCommand{
			Command:     "searchmp3",
			Description: "搜索并下载歌曲为 MP3，自动嵌入信息和歌词",
		},
		tgbotapi.BotCommand{
			Command:     "playlist",
			Description: "按歌单链接批量下载并嵌入信息和歌词",
		},
		tgbotapi.BotCommand{
			Command:     "playlistmp3",
			Description: "按歌单链接批量下载为 MP3 并嵌入信息和歌词",
		},
		tgbotapi.BotCommand{
			Command:     "stop",
			Description: "停止当前任务",
		},
		tgbotapi.BotCommand{
			Command:     "status",
			Description: "查看服务器和 Bot 运行状态",
		},
		tgbotapi.BotCommand{
			Command:     "tasks",
			Description: "查看当前任务",
		},
		tgbotapi.BotCommand{
			Command:     "download",
			Description: "打包 music 或 musicmp3 并生成下载直链",
		},
		tgbotapi.BotCommand{
			Command:     "file",
			Description: "密码验证后管理服务器文件",
		},
	)
	_, err := bot.Request(commands)
	return err
}
