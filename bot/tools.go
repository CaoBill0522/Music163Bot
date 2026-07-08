package bot

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/XiaoMengXinX/Music163Api-Go/types"
	"github.com/sirupsen/logrus"
)

// 解析作曲家信息
func parseArtist(songDetail types.SongDetailData) string {
	var artists string
	for i, ar := range songDetail.Ar {
		if i == 0 {
			artists = ar.Name
		} else {
			artists = fmt.Sprintf("%s/%s", artists, ar.Name)
		}
	}
	return artists
}

// 判断文件夹是否存在/新建文件夹
func dirExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			logrus.Errorf("mkdir %v failed: %v\n", path, err)
		}
		return false
	}
	logrus.Errorf("Error: %v\n", err)
	return false
}

// 校验 md5
func verifyMD5(filePath string, md5str string) (bool, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer f.Close()
	md5hash := md5.New()
	if _, err := io.Copy(md5hash, f); err != nil {
		return false, err
	}
	if hex.EncodeToString(md5hash.Sum(nil)) != md5str {
		return false, fmt.Errorf(md5VerFailed)
	}
	return true, nil
}

// 解析 MusicID
func parseMusicID(text string) int {
	var replacer = strings.NewReplacer("\n", "", " ", "")
	messageText := replacer.Replace(text)
	musicUrl := regUrl.FindStringSubmatch(messageText)
	if len(musicUrl) != 0 {
		if strings.Contains(musicUrl[0], "playlist") {
			return 0
		}
		if strings.Contains(musicUrl[0], "song") {
			ur, _ := url.Parse(musicUrl[0])
			if musicid := parseURLID(ur, "song"); musicid != 0 {
				return musicid
			}
		}
	}
	musicid, _ := strconv.Atoi(linkTestMusic(messageText))
	return musicid
}

// 解析 PlaylistID
func parsePlaylistID(text string) int {
	var replacer = strings.NewReplacer("\n", "", " ", "")
	messageText := replacer.Replace(text)
	musicUrl := regUrl.FindStringSubmatch(messageText)
	if len(musicUrl) == 0 || !strings.Contains(musicUrl[0], "playlist") {
		return 0
	}
	ur, err := url.Parse(musicUrl[0])
	if err != nil {
		return 0
	}
	return parseURLID(ur, "playlist")
}

func parseURLID(ur *url.URL, kind string) int {
	if ur == nil {
		return 0
	}
	if id := queryID(ur.RawQuery); id != 0 {
		return id
	}
	if id := pathID(ur.Path, kind); id != 0 {
		return id
	}
	fragment := strings.TrimPrefix(ur.Fragment, "/")
	if fragment == "" {
		return 0
	}
	if index := strings.Index(fragment, "?"); index >= 0 {
		if id := queryID(fragment[index+1:]); id != 0 {
			return id
		}
		fragment = fragment[:index]
	}
	return pathID("/"+fragment, kind)
}

func queryID(rawQuery string) int {
	if rawQuery == "" {
		return 0
	}
	id, _ := strconv.Atoi(urlParseQuery(rawQuery).Get("id"))
	return id
}

func urlParseQuery(rawQuery string) url.Values {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return url.Values{}
	}
	return values
}

func pathID(pathText, kind string) int {
	pathText = strings.TrimPrefix(pathText, "/")
	if !strings.HasPrefix(pathText, kind) {
		return 0
	}
	id, _ := strconv.Atoi(extractInt(strings.TrimPrefix(pathText, kind)))
	return id
}

// 提取数字
func extractInt(text string) string {
	matchArr := regInt.FindStringSubmatch(text)
	if len(matchArr) == 0 {
		return ""
	}
	return matchArr[0]
}

// 解析分享链接
func linkTestMusic(text string) string {
	return extractInt(reg5.ReplaceAllString(reg4.ReplaceAllString(reg3.ReplaceAllString(reg2.ReplaceAllString(reg1.ReplaceAllString(text, ""), ""), ""), ""), ""))
}

// 获取重定向后的地址
func getRedirectUrl(text string) string {
	var replacer = strings.NewReplacer("\n", "", " ", "")
	messageText := replacer.Replace(text)
	musicUrl := regUrl.FindStringSubmatch(messageText)
	if len(musicUrl) != 0 {
		if strings.Contains(musicUrl[0], "163cn.tv") || strings.Contains(musicUrl[0], "163cn.link") {
			var url = musicUrl[0]
			// 创建新的请求
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return text
			}

			// 设置 CheckRedirect 函数来处理重定向
			client := &http.Client{
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}

			// 执行请求
			resp, err := client.Do(req)
			if err != nil {
				return text
			}
			defer resp.Body.Close()

			// 返回最终重定向的网址
			location := resp.Header.Get("location")
			return location
		}
	}
	return text
}
