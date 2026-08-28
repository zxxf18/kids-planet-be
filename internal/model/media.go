package model

import "time"

type MediaItem struct {
	ID                int64     `json:"id"`
	SourceNo          int       `json:"-"`
	SourceCode        string    `json:"sourceCode"`
	Title             string    `json:"title"`
	TitleZH           string    `json:"titleZh"`
	AudioObjectKey    *string   `json:"-"`
	VideoObjectKey    *string   `json:"-"`
	LyricsObjectKey   *string   `json:"-"`
	PosterObjectKey   *string   `json:"-"`
	AudioDurationMS   *int64    `json:"audioDurationMs,omitempty"`
	VideoDurationMS   *int64    `json:"videoDurationMs,omitempty"`
	VideoWidth        *int      `json:"width,omitempty"`
	VideoHeight       *int      `json:"height,omitempty"`
	AudioCodec        *string   `json:"audioCodec,omitempty"`
	VideoCodec        *string   `json:"videoCodec,omitempty"`
	ValidationStatus  string    `json:"validationStatus"`
	ValidationMessage string    `json:"validationMessage,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	HasAudio          bool      `json:"hasAudio"`
	HasVideo          bool      `json:"hasVideo"`
	HasLyrics         bool      `json:"hasLyrics"`
	HasPoster         bool      `json:"hasPoster"`
	AudioURL          string    `json:"audioUrl,omitempty"`
	VideoURL          string    `json:"videoUrl,omitempty"`
	LyricsURL         string    `json:"lyricsUrl,omitempty"`
	PosterURL         string    `json:"posterUrl,omitempty"`
	Tags              []Tag     `json:"tags"`
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
	VideoObjectKey    *string
	LyricsObjectKey   *string
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
