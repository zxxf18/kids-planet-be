package storage

import (
	"path/filepath"
	"testing"
)

func TestLocalPosterUsesResourceRoot(t *testing.T) {
	resourceRoot := t.TempDir()
	generatedRoot := t.TempDir()
	assets := &Assets{mode: "local", resourceRoot: resourceRoot, generatedRoot: generatedRoot}

	actual, err := assets.LocalPath("poster", "poster/001. Song.jpg")
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(resourceRoot, "poster", "001. Song.jpg")
	if actual != expected {
		t.Fatalf("poster path = %q, want %q", actual, expected)
	}
}

func TestLocalKindUsesTextLyrics(t *testing.T) {
	tests := map[string]string{
		"lyrs/001. Song.lrc":           "lyricsEn",
		"lyrs_zh/001. Song.lrc":        "lyricsZh",
		"lyrs_bilingual/001. Song.lrc": "lyricsBilingual",
		"video_480/001. Song.mp4":      "video480",
		"video_720/001. Song.mp4":      "video720",
	}
	for key, expected := range tests {
		if actual := localKind(key); actual != expected {
			t.Fatalf("localKind(%q) = %q, want %q", key, actual, expected)
		}
	}
}
