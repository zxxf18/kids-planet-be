package media

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/zxxf18/kids-planet-be/internal/model"
)

type fakeResourceSource struct {
	entries []ResourceEntry
}

func (f fakeResourceSource) ListResources(context.Context, string) ([]ResourceEntry, error) {
	return f.entries, nil
}

func (fakeResourceSource) ProbeResource(_ context.Context, kind, key string) (ProbeResult, error) {
	result := ProbeResult{ResourcePath: key, DurationMS: 1000}
	if kind == "audio" {
		result.AudioCodec = "mp3"
	} else if kind == "video480" || kind == "video720" {
		result.VideoCodec = "h264"
		result.Width = 1920
		result.Height = 1080
	}
	return result, nil
}

type captureWriter struct {
	items []model.UpsertMedia
}

func (w *captureWriter) Upsert(_ context.Context, item model.UpsertMedia) error {
	w.items = append(w.items, item)
	return nil
}

func TestCleanTitle(t *testing.T) {
	tests := map[string]string{
		"001. Five Little Monkeys": "Five Little Monkeys",
		"003 Hickory Dickory Dock": "Hickory Dickory Dock",
		"  42_-_Twinkle Twinkle":   "Twinkle Twinkle",
		"no leading number":        "no leading number",
	}
	for input, expected := range tests {
		if actual := cleanTitle(input); actual != expected {
			t.Fatalf("cleanTitle(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestCandidateNeedsPlayableMedia(t *testing.T) {
	tests := []struct {
		name     string
		item     candidate
		playable bool
	}{
		{name: "audio only", item: candidate{audio: "song/001.mp3"}, playable: true},
		{name: "video only", item: candidate{video480: "video_480/001.mp4"}, playable: true},
		{name: "lyrics only", item: candidate{lyricsEn: "lyrs/001.lrc"}, playable: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := hasPlayableMedia(&test.item)
			if actual != test.playable {
				t.Fatalf("playable = %v, want %v", actual, test.playable)
			}
		})
	}
}

func TestScannerReadsS3StyleResources(t *testing.T) {
	source := fakeResourceSource{entries: []ResourceEntry{
		{Kind: "audio", Key: "001. Five Little Monkeys.mp3", Name: "001. Five Little Monkeys.mp3"},
		{Kind: "video480", Key: "001. Five Little Monkeys.mp4", Name: "001. Five Little Monkeys.mp4"},
		{Kind: "video720", Key: "001. Five Little Monkeys.mp4", Name: "001. Five Little Monkeys.mp4"},
		{Kind: "lyricsEn", Key: "001. Five Little Monkeys.lrc", Name: "001. Five Little Monkeys.lrc"},
		{Kind: "lyricsZh", Key: "001. Five Little Monkeys.lrc", Name: "001. Five Little Monkeys.lrc"},
		{Kind: "lyricsBilingual", Key: "001. Five Little Monkeys.lrc", Name: "001. Five Little Monkeys.lrc"},
		{Kind: "poster", Key: "001. Five Little Monkeys.jpg", Name: "001. Five Little Monkeys.jpg"},
	}}
	writer := &captureWriter{}
	result, err := NewScanner(source, writer).Scan(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Saved != 1 || result.Ready != 1 || len(writer.items) != 1 {
		t.Fatalf("scan result = %+v, items = %d", result, len(writer.items))
	}
	item := writer.items[0]
	if item.AudioObjectKey == nil || *item.AudioObjectKey != "001. Five Little Monkeys.mp3" {
		t.Fatalf("audio key = %v", item.AudioObjectKey)
	}
	if item.LyricsEnObjectKey == nil || *item.LyricsEnObjectKey != "001. Five Little Monkeys.lrc" {
		t.Fatalf("lyrics key = %v", item.LyricsEnObjectKey)
	}
}

func TestMatchNamedImageUsesTitleBeforeSourceNumber(t *testing.T) {
	items := map[int]*candidate{
		3:  {no: 3, code: "003", audio: "song/003.mp3", titles: map[string]string{"audio": "Hickory Dickory...Crash!"}},
		10: {no: 10, code: "010", audio: "song/010.mp3", titles: map[string]string{"audio": "Open Shut Them"}},
		78: {no: 78, code: "078", audio: "song/078.mp3", titles: map[string]string{"audio": "Good Morning, Mr. Rooster"}},
	}

	if item := matchNamedImage(items, namedImageCandidate{no: 10, title: "Good Morning, Mr. Rooster"}); item == nil || item.no != 78 {
		t.Fatalf("Good Morning lyric matched item %v, want source 78", item)
	}
	if item := matchNamedImage(items, namedImageCandidate{no: 3, title: "Hickory Dickory Dock"}); item == nil || item.no != 3 {
		t.Fatalf("Hickory lyric matched item %v, want source 3", item)
	}
	if item := matchNamedImage(items, namedImageCandidate{no: 10, title: "Unrelated Song"}); item != nil {
		t.Fatalf("unrelated lyric matched source %d", item.no)
	}
}

func TestMatchNamedResourceUsesSourceNumberWhenTitlesRepeat(t *testing.T) {
	items := map[int]*candidate{
		10:  {no: 10, code: "010", audio: "song/010.mp3", titles: map[string]string{"audio": "Open Shut Them"}},
		171: {no: 171, code: "171", audio: "song/171.mp3", titles: map[string]string{"audio": "Open Shut Them"}},
	}

	image := namedImageCandidate{no: 171, kind: "poster", title: "Open Shut Them"}
	if item := matchNamedImage(items, image); item == nil || item.no != 171 {
		t.Fatalf("poster matched item %v, want source 171", item)
	}
}

func TestResolveWithinRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	generated := t.TempDir()
	prober, err := NewProber(root, generated, "ffprobe", "ffmpeg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prober.ResolveResource("../outside.mp4"); err == nil {
		t.Fatal("expected parent traversal to be rejected")
	}

	outside := filepath.Join(t.TempDir(), "outside.mp4")
	if err := os.WriteFile(outside, []byte("not media"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape.mp4")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if _, err := prober.ResolveResource("escape.mp4"); err == nil {
		t.Fatal("expected symlink escape to be rejected")
	}
}
