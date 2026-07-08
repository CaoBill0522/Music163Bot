package bot

import "testing"

func TestParsePlaylistID(t *testing.T) {
	tests := map[string]int{
		"https://music.163.com/#/playlist?id=2476611280": 2476611280,
		"https://music.163.com/playlist?id=2476611280":   2476611280,
		"https://music.163.com/playlist/2476611280":      2476611280,
		"分享一个歌单 https://music.163.com/#/playlist?id=42":  42,
	}

	for input, want := range tests {
		if got := parsePlaylistID(input); got != want {
			t.Fatalf("parsePlaylistID(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestParseMusicIDHashRoute(t *testing.T) {
	input := "https://music.163.com/#/song?id=123456"
	if got := parseMusicID(input); got != 123456 {
		t.Fatalf("parseMusicID(%q) = %d, want 123456", input, got)
	}
}
