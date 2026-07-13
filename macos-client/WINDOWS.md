# 163MUSIC Windows 客户端

## 系统要求

- Windows 10/11 64 位
- 安装版或免安装版均已内置 Electron、网易云接口和 FFmpeg，不需要另外安装运行环境

## 构建

在此目录打开 PowerShell，安装 Node.js 22 或更新的 LTS 版本后执行：

```powershell
npm install
npm run app:win
```

生成文件位于 `release`：

- `163MUSIC Setup *.exe`：安装版，可选择安装目录，并创建桌面和开始菜单快捷方式
- `163MUSIC *.exe`：免安装版，双击即可运行

## 数据位置

- 默认原始音乐：用户“音乐”目录下的 `163MUSIC\Source`
- 默认 MP3：用户“音乐”目录下的 `163MUSIC\MP3`
- 设置和下载队列：`%APPDATA%\163MUSIC`

首次运行后可在“设置”中修改下载目录，并通过扫码登录或填写 `MUSIC_U` 使用账号权限。

## 说明

未签名的自行构建版本首次启动时，Windows 可能显示 SmartScreen 提示。点击“更多信息”后可选择继续运行；若要分发给其他用户，建议为安装包配置代码签名证书。
