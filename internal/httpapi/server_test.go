package httpapi

import (
	"testing"

	"github.com/zxxf18/kids-planet-be/internal/model"
)

func TestObjectKeySelectsVideoQualityAndLyricLanguage(t *testing.T) {
	video480 := "001.mp4"
	video720 := "001.mp4"
	lyricsEn := "001.lrc"
	lyricsZh := "001.lrc"
	lyricsBi := "001.lrc"
	item := model.MediaItem{
		Video480ObjectKey: &video480,
		Video720ObjectKey: &video720,
		LyricsEnObjectKey: &lyricsEn,
		LyricsZhObjectKey: &lyricsZh,
		LyricsBiObjectKey: &lyricsBi,
	}
	tests := []struct {
		kind, quality, language string
		wantStorage, wantKey    string
	}{
		{kind: "video", quality: "480", wantStorage: "video480", wantKey: video480},
		{kind: "video", quality: "720", wantStorage: "video720", wantKey: video720},
		{kind: "lyrics", language: "en", wantStorage: "lyricsEn", wantKey: lyricsEn},
		{kind: "lyrics", language: "zh", wantStorage: "lyricsZh", wantKey: lyricsZh},
		{kind: "lyrics", language: "bilingual", wantStorage: "lyricsBilingual", wantKey: lyricsBi},
	}
	for _, test := range tests {
		storageKind, key, err := objectKey(item, test.kind, test.quality, test.language)
		if err != nil {
			t.Fatal(err)
		}
		if storageKind != test.wantStorage || key != test.wantKey {
			t.Fatalf("objectKey(%q, %q, %q) = %q, %q; want %q, %q", test.kind, test.quality, test.language, storageKind, key, test.wantStorage, test.wantKey)
		}
	}
}

func TestObjectKeyRejectsUnknownVariants(t *testing.T) {
	item := model.MediaItem{}
	if _, _, err := objectKey(item, "video", "1080", ""); err == nil {
		t.Fatal("expected unsupported video quality to fail")
	}
	if _, _, err := objectKey(item, "lyrics", "", "fr"); err == nil {
		t.Fatal("expected unsupported lyric language to fail")
	}
}
