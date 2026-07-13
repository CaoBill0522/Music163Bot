# 163MUSIC Desktop

Windows 与 macOS 共用的 Electron 桌面客户端。它提供歌曲搜索、歌单解析、试听、下载队列、原格式或 MP3 下载、歌词和元数据写入，以及 `MUSIC_U` 扫码登录。

## 目录

```text
desktop-client/
├── src/        React + MUI 界面
├── electron/   Electron 主进程、网易云接口和下载队列
├── assets/     macOS、Windows 图标
└── scripts/    打包后界面检查脚本
```

## 环境要求

- Node.js 22 或更新的 LTS 版本
- Windows 10/11 x64，或 macOS Apple Silicon

依赖中已包含 Electron 和 FFmpeg；打包后的应用不需要额外安装运行环境。

## 本地开发

在当前目录安装依赖并启动开发模式：

```bash
npm install
npm run electron:dev
```

## 构建

先执行一次依赖安装：

```bash
npm install
```

Windows x64 安装版与便携版：

```powershell
npm run app:win
```

也可以直接运行 [build-windows.ps1](build-windows.ps1)。生成文件位于 `release/`：

- `163MUSIC Setup *.exe`：安装版，可选择安装目录，创建桌面和开始菜单快捷方式。
- `163MUSIC *.exe`：便携版，双击即可运行。

macOS Apple Silicon 应用：

```bash
npm run build
npx electron-builder --mac dir --arm64
```

生成的应用位于 `release/mac-arm64/163MUSIC.app`。对外分发前应使用 Apple Developer ID 签名并完成公证。

## 数据位置

- 默认原始音乐：系统“音乐”目录下的 `163MUSIC/Source`
- 默认 MP3：系统“音乐”目录下的 `163MUSIC/MP3`
- 设置、下载队列和日志：Electron 用户数据目录。Windows 通常为 `%APPDATA%\163MUSIC`，macOS 通常为 `~/Library/Application Support/163MUSIC`。

首次启动后可在“设置”中修改下载目录，并通过扫码登录或填写 `MUSIC_U` 使用账号权限。

## 发布版本

用户可从仓库的 [GitHub Release](https://github.com/CaoBill0522/Music163Bot/releases/latest) 下载 macOS Apple Silicon、Windows 安装版或 Windows 便携版。
