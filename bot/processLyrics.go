package bot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/XiaoMengXinX/Music163Api-Go/api"
	"github.com/XiaoMengXinX/Music163Api-Go/types"
)

type lyricData struct {
	Lrc struct {
		Lyric string `json:"lyric"`
	} `json:"lrc"`
	Tlyric struct {
		Lyric string `json:"lyric"`
	} `json:"tlyric"`
	Romalrc struct {
		Lyric string `json:"lyric"`
	} `json:"romalrc"`
}

func downloadLyricToServer(musicID int) (string, error) {
	info, lyric, err := getLyricInfo(musicID)
	if err != nil {
		return "", err
	}
	content := buildLyricContent(lyric)
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf(getLrcFailed)
	}

	err = os.MkdirAll(downloadDir, 0755)
	if err != nil {
		return "", err
	}
	fileName := safeLyricFileName(info)
	filePath := uniqueFilePath(filepath.Join(downloadDir, fileName))
	err = os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return filePath, nil
}

func getLyricInfo(musicID int) (songInfo, lyricData, error) {
	b := api.NewBatch(
		api.BatchAPI{
			Key:  api.SongDetailAPI,
			Json: api.CreateSongDetailReqJson([]int{musicID}),
		},
		api.BatchAPI{
			Key:  api.SongLyricAPI,
			Json: api.CreateSongLyricReqJson(musicID),
		},
	)
	if doErr := b.Do(data).Error; doErr != nil {
		return songInfo{}, lyricData{}, doErr
	}
	_, result := b.Parse()

	var detail types.SongsDetailData
	_ = json.Unmarshal([]byte(result[api.SongDetailAPI]), &detail)

	var lyric lyricData
	_ = json.Unmarshal([]byte(result[api.SongLyricAPI]), &lyric)

	if len(detail.Songs) == 0 {
		return songInfo{}, lyricData{}, fmt.Errorf(fetchInfoFailed)
	}
	info := songInfo{
		MusicID:     musicID,
		SongName:    detail.Songs[0].Name,
		SongArtists: parseArtist(detail.Songs[0]),
		SongAlbum:   detail.Songs[0].Al.Name,
		PicURL:      detail.Songs[0].Al.PicUrl,
		FileExt:     "lrc",
	}
	return info, lyric, nil
}

func buildLyricContent(lyric lyricData) string {
	var parts []string
	if strings.TrimSpace(lyric.Lrc.Lyric) != "" {
		parts = append(parts, lyric.Lrc.Lyric)
	}
	if strings.TrimSpace(lyric.Tlyric.Lyric) != "" {
		parts = append(parts, "\n[translation]\n"+lyric.Tlyric.Lyric)
	}
	if strings.TrimSpace(lyric.Romalrc.Lyric) != "" {
		parts = append(parts, "\n[romaji]\n"+lyric.Romalrc.Lyric)
	}
	return strings.Join(parts, "\n")
}

func safeLyricFileName(info songInfo) string {
	replacer := strings.NewReplacer("/", " ", "?", " ", "*", " ", ":", " ", "|", " ", "\\", " ", "<", " ", ">", " ", "\"", " ")
	base := fmt.Sprintf("%s - %s.lrc", strings.ReplaceAll(info.SongArtists, "/", ","), info.SongName)
	name := strings.TrimSpace(replacer.Replace(base))
	if name == ".lrc" || name == "" {
		return strconv.FormatInt(time.Now().UnixMicro(), 10) + ".lrc"
	}
	return name
}
