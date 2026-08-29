package model

import "time"

type MediaItem struct {
	ID                int64             `json:"id"`
	SourceNo          int               `json:"-"`
	SourceCode        string            `json:"sourceCode"`
	Title             string            `json:"title"`
	TitleZH           string            `json:"titleZh"`
	AudioObjectKey    *string           `json:"-"`
	Video480ObjectKey *string           `json:"-"`
	Video720ObjectKey *string           `json:"-"`
	LyricsEnObjectKey *string           `json:"-"`
	LyricsZhObjectKey *string           `json:"-"`
	LyricsBiObjectKey *string           `json:"-"`
	PosterObjectKey   *string           `json:"-"`
	AudioDurationMS   *int64            `json:"audioDurationMs,omitempty"`
	VideoDurationMS   *int64            `json:"videoDurationMs,omitempty"`
	VideoWidth        *int              `json:"width,omitempty"`
	VideoHeight       *int              `json:"height,omitempty"`
	AudioCodec        *string           `json:"audioCodec,omitempty"`
	VideoCodec        *string           `json:"videoCodec,omitempty"`
	ValidationStatus  string            `json:"validationStatus"`
	ValidationMessage string            `json:"validationMessage,omitempty"`
	CreatedAt         time.Time         `json:"createdAt"`
	UpdatedAt         time.Time         `json:"updatedAt"`
	HasAudio          bool              `json:"hasAudio"`
	HasVideo          bool              `json:"hasVideo"`
	HasLyrics         bool              `json:"hasLyrics"`
	HasPoster         bool              `json:"hasPoster"`
	AudioURL          string            `json:"audioUrl,omitempty"`
	VideoSources      map[string]string `json:"videoSources,omitempty"`
	LyricsSources     map[string]string `json:"lyricsSources,omitempty"`
	PosterURL         string            `json:"posterUrl,omitempty"`
	Tags              []Tag             `json:"tags"`
}

type Tag struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Count int64  `json:"count,omitempty"`
}

type UpsertMedia struct {
	SourceNo          int
	SourceCode        string
	Title             string
	AudioObjectKey    *string
	Video480ObjectKey *string
	Video720ObjectKey *string
	LyricsEnObjectKey *string
	LyricsZhObjectKey *string
	LyricsBiObjectKey *string
	PosterObjectKey   *string
	AudioDurationMS   *int64
	VideoDurationMS   *int64
	VideoWidth        *int
	VideoHeight       *int
	AudioCodec        *string
	VideoCodec        *string
	ValidationStatus  string
	ValidationMessage string
}
