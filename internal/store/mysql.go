package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zxxf18/kids-planet-be/internal/model"
)

type MySQL struct {
	db *sql.DB
}

func NewMySQL(dsn string) (*MySQL, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(8)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect mysql: %w", err)
	}
	return &MySQL{db: db}, nil
}

func (s *MySQL) Close() error { return s.db.Close() }

func (s *MySQL) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

const mediaColumns = `m.id, m.source_no, m.source_code, m.title,
 COALESCE(c.title_zh, ''), m.audio_object_key, m.video_480_object_key, m.video_720_object_key,
 m.lyrics_en_object_key, m.lyrics_zh_object_key, m.lyrics_bilingual_object_key, m.poster_object_key,
 m.audio_duration_ms, m.video_duration_ms, m.video_width, m.video_height,
 m.audio_codec, m.video_codec, m.validation_status, COALESCE(m.validation_message, ''),
 m.created_at, m.updated_at`

const mediaFrom = ` FROM media_item m LEFT JOIN media_catalog c ON c.source_no = m.source_no`

type ListFilter struct {
	Query string
	Tag   string
	IDs   []int64
}

func (s *MySQL) List(ctx context.Context, filter ListFilter, page, pageSize int) ([]model.MediaItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 30
	}
	if pageSize > 50 {
		pageSize = 50
	}
	conditions := make([]string, 0, 3)
	args := make([]any, 0, 8)
	if q := strings.TrimSpace(filter.Query); q != "" {
		conditions = append(conditions, "(m.title LIKE ? OR c.title_zh LIKE ? OR m.source_code LIKE ?)")
		like := "%" + q + "%"
		args = append(args, like, like, like)
	}
	if tag := strings.TrimSpace(filter.Tag); tag != "" {
		conditions = append(conditions, `EXISTS (
 SELECT 1 FROM media_catalog_tag mct JOIN media_tag mt ON mt.id = mct.tag_id
 WHERE mct.source_no = m.source_no AND mt.slug = ?
)`)
		args = append(args, tag)
	}
	if len(filter.IDs) > 0 {
		placeholders := make([]string, len(filter.IDs))
		for index, id := range filter.IDs {
			placeholders[index] = "?"
			args = append(args, id)
		}
		conditions = append(conditions, "m.id IN ("+strings.Join(placeholders, ",")+")")
	}
	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*)"+mediaFrom+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	listArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := s.db.QueryContext(ctx, "SELECT "+mediaColumns+mediaFrom+where+" ORDER BY m.source_no LIMIT ? OFFSET ?", listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]model.MediaItem, 0, pageSize)
	for rows.Next() {
		item, err := scanMedia(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if err := s.attachTags(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *MySQL) Get(ctx context.Context, id int64) (model.MediaItem, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+mediaColumns+mediaFrom+" WHERE m.id = ?", id)
	item, err := scanMedia(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.MediaItem{}, ErrNotFound
	}
	if err != nil {
		return item, err
	}
	items := []model.MediaItem{item}
	if err := s.attachTags(ctx, items); err != nil {
		return model.MediaItem{}, err
	}
	return items[0], nil
}

func (s *MySQL) Upsert(ctx context.Context, item model.UpsertMedia) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO media_item (
	 source_no, source_code, title, audio_object_key, video_480_object_key, video_720_object_key,
	 lyrics_en_object_key, lyrics_zh_object_key, lyrics_bilingual_object_key,
 poster_object_key, audio_duration_ms, video_duration_ms, video_width, video_height,
 audio_codec, video_codec, validation_status, validation_message
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
 source_code = VALUES(source_code), title = VALUES(title),
	 audio_object_key = VALUES(audio_object_key), video_480_object_key = VALUES(video_480_object_key),
	 video_720_object_key = VALUES(video_720_object_key), lyrics_en_object_key = VALUES(lyrics_en_object_key),
	 lyrics_zh_object_key = VALUES(lyrics_zh_object_key), lyrics_bilingual_object_key = VALUES(lyrics_bilingual_object_key),
	 poster_object_key = VALUES(poster_object_key),
 audio_duration_ms = VALUES(audio_duration_ms), video_duration_ms = VALUES(video_duration_ms),
 video_width = VALUES(video_width), video_height = VALUES(video_height),
 audio_codec = VALUES(audio_codec), video_codec = VALUES(video_codec),
 validation_status = VALUES(validation_status), validation_message = VALUES(validation_message)`,
		item.SourceNo, item.SourceCode, item.Title, item.AudioObjectKey, item.Video480ObjectKey,
		item.Video720ObjectKey, item.LyricsEnObjectKey, item.LyricsZhObjectKey, item.LyricsBiObjectKey,
		item.PosterObjectKey, item.AudioDurationMS, item.VideoDurationMS,
		item.VideoWidth, item.VideoHeight, item.AudioCodec, item.VideoCodec,
		item.ValidationStatus, nullableText(item.ValidationMessage))
	return err
}

var ErrNotFound = errors.New("media not found")

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMedia(row rowScanner) (model.MediaItem, error) {
	var item model.MediaItem
	err := row.Scan(
		&item.ID, &item.SourceNo, &item.SourceCode, &item.Title, &item.TitleZH,
		&item.AudioObjectKey, &item.Video480ObjectKey, &item.Video720ObjectKey,
		&item.LyricsEnObjectKey, &item.LyricsZhObjectKey, &item.LyricsBiObjectKey, &item.PosterObjectKey,
		&item.AudioDurationMS, &item.VideoDurationMS, &item.VideoWidth, &item.VideoHeight,
		&item.AudioCodec, &item.VideoCodec, &item.ValidationStatus, &item.ValidationMessage,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return model.MediaItem{}, err
	}
	item.HasAudio = item.AudioObjectKey != nil
	item.HasVideo = item.Video480ObjectKey != nil || item.Video720ObjectKey != nil
	item.HasLyrics = item.LyricsEnObjectKey != nil || item.LyricsZhObjectKey != nil || item.LyricsBiObjectKey != nil
	item.HasPoster = item.PosterObjectKey != nil
	item.Tags = []model.Tag{}
	return item, nil
}

func (s *MySQL) ListTags(ctx context.Context) ([]model.Tag, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT mt.slug, mt.name, mt.icon, COUNT(m.source_no)
FROM media_tag mt
JOIN media_catalog_tag mct ON mct.tag_id = mt.id
JOIN media_item m ON m.source_no = mct.source_no
GROUP BY mt.id, mt.slug, mt.name, mt.icon, mt.min_items
HAVING COUNT(m.source_no) >= mt.min_items
ORDER BY mt.sort_order, mt.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := make([]model.Tag, 0, 12)
	for rows.Next() {
		var tag model.Tag
		if err := rows.Scan(&tag.Slug, &tag.Name, &tag.Icon, &tag.Count); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (s *MySQL) attachTags(ctx context.Context, items []model.MediaItem) error {
	if len(items) == 0 {
		return nil
	}
	placeholders := make([]string, len(items))
	args := make([]any, len(items))
	byID := make(map[int64]int, len(items))
	for index := range items {
		placeholders[index] = "?"
		args[index] = items[index].ID
		byID[items[index].ID] = index
	}
	rows, err := s.db.QueryContext(ctx, `SELECT m.id, mt.slug, mt.name, mt.icon
FROM media_item m
JOIN media_catalog_tag mct ON mct.source_no = m.source_no
JOIN media_tag mt ON mt.id = mct.tag_id
WHERE m.id IN (`+strings.Join(placeholders, ",")+`)
ORDER BY mt.sort_order, mt.id`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var mediaID int64
		var tag model.Tag
		if err := rows.Scan(&mediaID, &tag.Slug, &tag.Name, &tag.Icon); err != nil {
			return err
		}
		if index, ok := byID[mediaID]; ok {
			items[index].Tags = append(items[index].Tags, tag)
		}
	}
	return rows.Err()
}

func nullableText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
