package bot

import (
	"context"
	"regexp"
	"sync"
	"time"

	"github.com/XiaoMengXinX/Music163Api-Go/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// config 配置文件数据
var config map[string]string

// data 网易云 cookie
var data utils.RequestData

var bot *tgbotapi.BotAPI
var botName string
var cacheDir = "./cache"
var botAPI = "https://api.telegram.org"
var downloadDir = "/home/music"
var mp3Dir = "/home/music_mp3"
var fileURLBase string
var fileRoot = "/home"
var filePassword string
var vipProbeKeyword = "徐良 飞机场"

// downloaderTimeout 下载超时时间
var downloaderTimeout int

var botStartedAt = time.Now()

var searchWait = struct {
	sync.Mutex
	users map[string]bool
}{
	users: make(map[string]bool),
}

var searchMP3Wait = struct {
	sync.Mutex
	users map[string]bool
}{
	users: make(map[string]bool),
}

var playlistWait = struct {
	sync.Mutex
	users map[string]bool
}{
	users: make(map[string]bool),
}

var playlistMP3Wait = struct {
	sync.Mutex
	users map[string]bool
}{
	users: make(map[string]bool),
}

var downloadArchiveWait = struct {
	sync.Mutex
	users map[string]bool
}{
	users: make(map[string]bool),
}

type playlistSelectionTask struct {
	PlaylistID   int
	PlaylistName string
	MusicIDs     []int
	Items        []playlistMusicItem
	Stage        string
	ToMP3        bool
}

var playlistSelectionWait = struct {
	sync.Mutex
	users map[string]playlistSelectionTask
}{
	users: make(map[string]playlistSelectionTask),
}

var taskControl = struct {
	sync.Mutex
	stopped map[int64]bool
	active  map[int64]string
	ctx     map[int64]context.Context
	cancel  map[int64]context.CancelFunc
}{
	stopped: make(map[int64]bool),
	active:  make(map[int64]string),
	ctx:     make(map[int64]context.Context),
	cancel:  make(map[int64]context.CancelFunc),
}

type fileSession struct {
	Authed bool
	Cwd    string
}

var fileSessions = struct {
	sync.Mutex
	users map[string]fileSession
}{
	users: make(map[string]fileSession),
}

var (
	reg1   = regexp.MustCompile(`(.*)song\?id=`)
	reg2   = regexp.MustCompile("(.*)song/")
	reg5   = regexp.MustCompile("/(.*)")
	reg4   = regexp.MustCompile("&(.*)")
	reg3   = regexp.MustCompile(`\?(.*)`)
	regInt = regexp.MustCompile(`\d+`)
	regUrl = regexp.MustCompile("(http|https)://[\\w\\-_]+(\\.[\\w\\-_]+)+([\\w\\-.,@?^=%&:/~+#]*[\\w\\-@?^=%&/~+#])?")
)

var (
	musicInfoMsg = `%s
专辑: %s
%s %.2fMB
`
	inputKeyword      = "请输入搜索关键词"
	inputNextKeyword  = "请发送要搜索的歌名"
	inputNextMP3      = "请发送要搜索并转换为 MP3 的歌名"
	inputNextPlaylist = "请发送网易云歌单链接"
	inputPlaylistMode = "请回复 全部/all 下载全部歌曲，或回复 部分/part 选择要下载的歌曲；发送 /stop 取消当前任务"
	inputPlaylistPick = "请回复要下载的歌曲序号，例如 1,3-5,8；发送 /stop 取消当前任务"
	inputDownloadType = "请回复 music 或 musicmp3，Bot 会压缩对应目录并生成下载直链；发送 /stop 取消"
	inputFilePassword = "请输入文件管理密码；发送 /stop 取消"
	taskStopped       = "已请求停止当前任务"
	noActiveTask      = "当前没有正在运行的任务"
	playlistCanceled  = "当前歌单任务已停止"
	searching         = `搜索中...`
	fetchingPlaylist  = `正在获取歌单中...`
	playlistEmpty     = `歌单中没有可处理的歌曲`
	fetchingLyric     = `正在获取歌词中...`
	noResults         = `未找到结果`
	getLrcFailed      = `获取歌词失败，歌曲可能不存在或为纯音乐`
	getUrlFailed      = `获取歌曲下载链接失败`
	fetchInfo         = `正在获取歌曲信息...`
	fetchInfoFailed   = `获取歌曲信息失败`
	downloading       = `下载中...`
	downloadStatus    = " %s\n%.2fMB/%.2fMB %d%%"
	redownloading     = `下载失败，尝试重新下载中...`
	downloadDone      = "下载完成\n%s"
	md5VerFailed      = "MD5校验失败"
	retryLater        = "请稍后重试"

	callbackText = "Success"
)
