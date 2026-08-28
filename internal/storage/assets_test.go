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
	if actual := localKind("lyrs/001. Song.lrc"); actual != "lyrics" {
		t.Fatalf("lrc kind = %q, want lyrics", actual)
	}
}
