package bot

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func embedMetadataIntoAudio(ctx context.Context, audioPath, lyricPath string, info songInfo) error {
	lyrics, err := os.ReadFile(lyricPath)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(lyrics)) == "" {
		return fmt.Errorf("歌词内容为空")
	}
	if _, err = exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("未找到 ffmpeg，请先安装 ffmpeg")
	}

	ext := strings.ToLower(filepath.Ext(audioPath))
	if ext != ".mp3" && ext != ".flac" && ext != ".m4a" {
		return fmt.Errorf("暂不支持嵌入 %s 文件", ext)
	}

	coverPath, coverErr := downloadCoverImage(ctx, info.PicURL)
	if coverErr == nil {
		defer os.Remove(coverPath)
	}

	dir := filepath.Dir(audioPath)
	base := strings.TrimSuffix(filepath.Base(audioPath), ext)
	tmpPath := filepath.Join(dir, fmt.Sprintf(".%s.%d.embed%s", base, time.Now().UnixMicro(), ext))
	args := []string{
		"-y",
		"-i", audioPath,
	}
	if coverErr == nil {
		args = append(args, "-i", coverPath)
	}
	args = append(args, "-map", "0")
	if coverErr == nil {
		args = append(args, "-map", "1", "-disposition:v:0", "attached_pic")
	}
	args = append(args,
		"-c", "copy",
		"-metadata", "title="+info.SongName,
		"-metadata", "artist="+info.SongArtists,
		"-metadata", "album="+info.SongAlbum,
		"-metadata", "lyrics="+string(lyrics),
		"-metadata", "unsyncedlyrics="+string(lyrics),
	)
	if ext == ".mp3" {
		args = append(args, "-id3v2_version", "3")
	}
	args = append(args, tmpPath)

	cmdCtx, cancel := commandContext(ctx)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(tmpPath)
		if cmdCtx.Err() != nil {
			return fmt.Errorf("ffmpeg 嵌入超时或任务已停止: %w", cmdCtx.Err())
		}
		return fmt.Errorf("ffmpeg 嵌入失败: %s", strings.TrimSpace(string(output)))
	}
	err = os.Rename(tmpPath, audioPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func downloadCoverImage(ctx context.Context, picURL string) (string, error) {
	if picURL == "" {
		return "", fmt.Errorf("缺少封面地址")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, "GET", picURL, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: downloadStallTimeout()}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("封面下载失败: %s", resp.Status)
	}
	ext := strings.ToLower(path.Ext(picURL))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		ext = ".jpg"
	}
	file, err := os.CreateTemp("", "music-cover-*"+ext)
	if err != nil {
		return "", err
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		_ = os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}
