# Music163bot-Go

一个自用向 Telegram 网易云音乐下载 Bot。Bot 可以搜索单曲、下载歌单、转换 MP3、嵌入歌曲信息和歌词，也可以打包服务器里的音乐目录生成下载直链。

仓库同时包含桌面客户端：`macos-client/` 是共享的 Electron 客户端工程，可构建 macOS Apple Silicon 应用，以及 Windows x64 安装版和便携版。桌面端的具体构建方式见 [macos-client/WINDOWS.md](macos-client/WINDOWS.md)。

默认目录：

- 原始音乐目录：`/home/music`
- MP3 音乐目录：`/home/music_mp3`
- 文件管理根目录：`/home`

## 功能

- `/start`：展示功能菜单。
- `/search`：搜索单曲，弹出结果列表，用户选择后自动下载，并嵌入歌名、歌手、专辑、封面和歌词。
- `/searchmp3`：搜索单曲，弹出结果列表，用户选择后自动下载、转换为 MP3，并嵌入歌名、歌手、专辑、封面和歌词。
- `/playlist`：下载歌单。用户发送歌单链接后，Bot 先展示歌单内全部歌曲，再询问下载全部还是下载部分；部分下载支持 `1-100`、`1,3-5,8` 这类序号范围。
- `/playlistmp3`：流程同 `/playlist`，但会转换为 MP3 后保存到 MP3 目录，并嵌入歌曲信息和歌词。
- `/tasks`：展示当前正在进行的任务。
- `/stop`：停止当前任务，或退出当前等待输入的交互。
- `/file`：密码验证后进入文件管理模式，可查看、进入、创建、删除、压缩、解压 `FileRoot` 内的文件。
- `/status`：展示服务器硬件、磁盘和 Bot 实例运行信息，并静默检测 `MUSIC_U` 是否有效、VIP 下载是否可用。
- `/download`：询问打包 `music` 还是 `musicmp3`，压缩对应目录内歌曲，并返回公网下载直链。

## 行为说明

- 下载过程会实时更新进度和下载速度。
- 歌单批量下载时，每首歌曲之间间隔 2 秒，降低请求过快导致 IP 被限制的风险。
- 歌单中某首歌失败后，会继续处理后续歌曲；任务结束后会展示失败歌曲名称，并提供“重新下载失败歌曲”按钮。
- 下载前会检查目标目录中是否已有同名歌曲；如果已存在，会提示歌曲已存在并跳过。
- 歌词嵌入成功后，会自动删除对应 `.lrc` 文件。
- `/download` 是唯一生成文件下载直链的功能；其他功能只保存文件或显示本地路径。
- `/status` 的 VIP 检测是静默执行的，只显示检测结果，不显示检测方法或临时文件信息。

## 指令菜单

Bot 启动时会自动注册 Telegram 指令菜单：

```text
/start - 展示功能菜单
/search - 搜索并下载歌曲，自动嵌入信息和歌词
/searchmp3 - 搜索并下载歌曲为 MP3，自动嵌入信息和歌词
/playlist - 按歌单链接批量下载并嵌入信息和歌词
/playlistmp3 - 按歌单链接批量下载为 MP3 并嵌入信息和歌词
/stop - 停止当前任务
/status - 查看服务器和 Bot 运行状态
/tasks - 查看当前任务
/download - 打包 music 或 musicmp3 并生成下载直链
/file - 密码验证后管理服务器文件
```

如果 Telegram 客户端里菜单没有立刻刷新，重启 Bot 后等一会儿，Telegram 的命令缓存会自动更新。

## 配置

复制配置文件：

```bash
cp config_example.ini config.ini
```

编辑 `config.ini`：

```ini
BOT_TOKEN = YOUR_BOT_TOKEN
MUSIC_U = YOUR_MUSIC_U
BotAPI = https://api.telegram.org
BotDebug = false
DownloadPath = /home/music
Mp3Path = /home/music_mp3
FileURLBase = https://example.com/download
FilePassword = CHANGE_ME
FileRoot = /home
VipProbeKeyword = 徐良 飞机场
LogLevel = info
DownloadTimeout = 60
ReverseProxy =
```

配置说明：

- `BOT_TOKEN`：Telegram Bot Token，必填。
- `MUSIC_U`：网易云 Cookie 中的 `MUSIC_U` 值，用于登录网易云账号。留空时只能按公开权限下载。
- `BotAPI`：Telegram Bot API 地址，默认 `https://api.telegram.org`。
- `BotDebug`：是否开启 Telegram Bot API debug 日志。
- `DownloadPath`：原始音频下载目录，默认 `/home/music`。
- `Mp3Path`：MP3 输出目录，默认 `/home/music_mp3`。
- `FileURLBase`：文件公网访问地址前缀，仅 `/download` 用它生成直链。比如 `FileRoot=/home`、`FileURLBase=https://example.com/download`，则 `/home/music/a.zip` 会生成 `https://example.com/download/music/a.zip`。
- `FilePassword`：`/file` 文件管理密码。留空时文件管理功能不可用。
- `FileRoot`：文件管理和下载直链映射根目录，默认 `/home`。文件管理不能访问该目录之外的路径。
- `VipProbeKeyword`：`/status` 静默检测 VIP 下载能力时使用的付费歌曲关键词。
- `LogLevel`：日志等级，可选 `panic`、`fatal`、`error`、`warn`、`info`、`debug`、`trace`。
- `DownloadTimeout`：单次下载超时时间，单位秒。
- `ReverseProxy`：下载失败后的备用代理地址，可留空。

## 服务器部署

以下以 Ubuntu/Debian 为例。

1. 安装系统依赖：

```bash
sudo apt update
sudo apt install -y git wget nginx ffmpeg ca-certificates
```

2. 安装 Go，建议 Go 1.22 或更新版本：

```bash
wget https://go.dev/dl/go1.22.12.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.12.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
```

3. 上传或拉取项目：

```bash
cd /home
git clone <你的仓库地址> Music163bot-Go
cd Music163bot-Go
```

如果你不用 Git，也可以直接把项目目录上传到 `/home/Music163bot-Go`。

4. 准备目录和配置：

```bash
sudo mkdir -p /home/music /home/music_mp3
sudo chmod 755 /home/music /home/music_mp3
cp config_example.ini config.ini
nano config.ini
```

5. 下载依赖并编译：

```bash
go mod tidy
go build -o Music163bot-Go .
```

6. 前台运行测试：

```bash
./Music163bot-Go -c config.ini
```

看到 `验证成功` 后，在 Telegram 里发送 `/start`、`/search`、`/playlist` 或 `/status` 测试。

## 配置 Nginx 下载直链

`/download` 会在 `/home/music` 或 `/home/music_mp3` 中生成 zip 压缩包，并用 `FileURLBase` 拼出下载地址。你需要让 Nginx 把公网路径映射到 `FileRoot`。

示例：让 `https://你的域名/download/...` 对应服务器 `/home/...`：

```nginx
server {
    listen 80;
    server_name 你的域名;

    location /download/ {
        alias /home/;
        autoindex off;
    }
}
```

配置后检查并重载：

```bash
sudo nginx -t
sudo systemctl reload nginx
```

对应 `config.ini`：

```ini
FileRoot = /home
FileURLBase = https://你的域名/download
```

如果你暂时没有域名，也可以先用服务器 IP：

```ini
FileURLBase = http://服务器IP/download
```

## 使用 systemd 后台运行

创建服务文件：

```bash
sudo nano /etc/systemd/system/music163bot.service
```

写入以下内容，按你的实际路径调整：

```ini
[Unit]
Description=Music163 Telegram Bot
After=network.target

[Service]
Type=simple
WorkingDirectory=/home/Music163bot-Go
ExecStart=/home/Music163bot-Go/Music163bot-Go -c /home/Music163bot-Go/config.ini
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

启动并设置开机自启：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now music163bot
sudo systemctl status music163bot
```

查看日志：

```bash
journalctl -u music163bot -f
```

更新代码后重新编译并重启：

```bash
go mod tidy
go build -o Music163bot-Go .
sudo systemctl restart music163bot
```

## 文件管理模式

发送 `/file` 后输入 `FilePassword`，验证通过后可使用：

```text
ls
pwd
cd xxx
mkdir xxx
rm xxx
zip 压缩包名.zip 文件或文件夹
unzip 压缩包.zip [目录]
exit
```

文件管理被限制在 `FileRoot` 内，不能访问根目录之外的路径。`rm` 会直接删除文件或文件夹，使用前请确认路径。

## 本地测试

本地需要安装：

- Go 1.22 或更新版本
- `ffmpeg` 和 `ffprobe`

macOS 可用 Homebrew 安装：

```bash
brew install go ffmpeg
```

然后执行：

```bash
go mod tidy
go test ./...
go build -o Music163bot-Go .
./Music163bot-Go -c config.ini
```

本地测试时建议把 `DownloadPath`、`Mp3Path` 改成本机可写目录，例如：

```ini
DownloadPath = /Users/你的用户名/Downloads/music
Mp3Path = /Users/你的用户名/Downloads/music_mp3
```

## 注意事项

- 服务器必须安装 `ffmpeg` 和 `ffprobe`，否则 MP3 转换、元数据嵌入和 VIP 检测可能失败。
- 歌单批量下载耗时较长，建议先用小歌单测试。
- `/stop` 会停止后续任务；正在进行中的单首下载可能需要等当前下载函数返回。
- 歌单失败重试按钮依赖当前 Bot 进程内存；如果 Bot 重启，旧按钮对应的失败任务会失效。
- 某些歌曲可能因版权、会员权限、地区限制或无歌词导致下载失败。
- 请确认运行 Bot 的系统用户对 `DownloadPath`、`Mp3Path` 和 `FileRoot` 有读写权限。

## 相关项目

- `XiaoMengXinX/Music163Api-Go`：网易云音乐搜索、歌曲详情、下载地址、歌词和歌单接口。
- `XiaoMengXinX/SimpleDownloader`：音频下载、断点续传、多线程下载和进度统计。
- `go-telegram-bot-api/telegram-bot-api`：Telegram Bot 消息、命令、按钮和回调处理。
- `ffmpeg`：音频转换、封面和元数据写入。
