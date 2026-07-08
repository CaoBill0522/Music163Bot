package bot

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/XiaoMengXinX/Music163Api-Go/api"
)

func probeVipByPaidSong() string {
	musicID, _, err := findVipProbeSong()
	if err != nil {
		return "VIP: 未知"
	}

	info, songURL, err := getDownloadSongInfo(musicID)
	if err != nil {
		return "VIP: 不可用"
	}
	if songURL.Url == "" {
		return "VIP: 不可用"
	}

	tmpPath, err := downloadProbeFile(songURL.Url, info.FileExt)
	if err != nil {
		return "VIP: 不可用"
	}
	defer os.Remove(tmpPath)

	duration, err := probeAudioDuration(tmpPath)
	if err == nil && duration > 0 {
		if duration <= 45 {
			return "VIP: 不可用"
		}
		return "VIP: 可用"
	}

	size, statErr := os.Stat(tmpPath)
	if statErr == nil {
		if size.Size() <= 2*1024*1024 {
			return "VIP: 不可用"
		}
		return "VIP: 可用"
	}
	return "VIP: 未知"
}

func findVipProbeSong() (int, string, error) {
	result, err := api.SearchSong(data, api.SearchSongConfig{
		Keyword: vipProbeKeyword,
		Limit:   10,
	})
	if err != nil {
		return 0, "", err
	}
	if len(result.Result.Songs) == 0 {
		return 0, "", fmt.Errorf("未找到测试歌: %s", vipProbeKeyword)
	}
	for _, song := range result.Result.Songs {
		artists := make([]string, 0, len(song.Artists))
		for _, artist := range song.Artists {
			artists = append(artists, artist.Name)
		}
		display := song.Name
		if len(artists) > 0 {
			display = strings.Join(artists, "/") + " - " + song.Name
		}
		if strings.Contains(song.Name, "飞机场") && strings.Contains(strings.Join(artists, "/"), "徐良") {
			return song.Id, display, nil
		}
	}
	song := result.Result.Songs[0]
	artists := make([]string, 0, len(song.Artists))
	for _, artist := range song.Artists {
		artists = append(artists, artist.Name)
	}
	display := song.Name
	if len(artists) > 0 {
		display = strings.Join(artists, "/") + " - " + song.Name
	}
	return song.Id, display, nil
}

func downloadProbeFile(rawURL, ext string) (string, error) {
	if ext == "" {
		ext = "mp3"
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %s", resp.Status)
	}

	file, err := os.CreateTemp("", "music-vip-probe-*"+ext)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := io.Copy(file, resp.Body); err != nil {
		_ = os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

func probeAudioDuration(filePath string) (float64, error) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return 0, err
	}
	cmdCtx, cancel := commandContext(context.Background())
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
}
