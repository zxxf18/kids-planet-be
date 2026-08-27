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

const mediaColumns = `id, source_no, source_code, title,
 audio_object_key, video_object_key, lyric_object_key, poster_object_key,
 audio_duration_ms, video_duration_ms, video_width, video_height,
 audio_codec, video_codec, validation_status, COALESCE(validation_message, ''),
 created_at, updated_at`

func (s *MySQL) List(ctx context.Context, query string, page, pageSize int) ([]model.MediaItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 30
	}
	if pageSize > 500 {
		pageSize = 500
	}
	where := ""
	args := make([]any, 0, 4)
	if q := strings.TrimSpace(query); q != "" {
		where = " WHERE title LIKE ? OR source_code LIKE ?"
		like := "%" + q + "%"
		args = append(args, like, like)
	}
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM media_item"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	listArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := s.db.QueryContext(ctx, "SELECT "+mediaColumns+" FROM media_item"+where+" ORDER BY source_no LIMIT ? OFFSET ?", listArgs...)
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
	return items, total, rows.Err()
}

func (s *MySQL) Get(ctx context.Context, id int64) (model.MediaItem, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+mediaColumns+" FROM media_item WHERE id = ?", id)
	item, err := scanMedia(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.MediaItem{}, ErrNotFound
	}
	return item, err
}

func (s *MySQL) Upsert(ctx context.Context, item model.UpsertMedia) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO media_item (
 source_no, source_code, title, audio_object_key, video_object_key, lyric_object_key,
 poster_object_key, audio_duration_ms, video_duration_ms, video_width, video_height,
 audio_codec, video_codec, validation_status, validation_message
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
 source_code = VALUES(source_code), title = VALUES(title),
 audio_object_key = VALUES(audio_object_key), video_object_key = VALUES(video_object_key),
 lyric_object_key = VALUES(lyric_object_key), poster_object_key = VALUES(poster_object_key),
 audio_duration_ms = VALUES(audio_duration_ms), video_duration_ms = VALUES(video_duration_ms),
 video_width = VALUES(video_width), video_height = VALUES(video_height),
 audio_codec = VALUES(audio_codec), video_codec = VALUES(video_codec),
 validation_status = VALUES(validation_status), validation_message = VALUES(validation_message)`,
		item.SourceNo, item.SourceCode, item.Title, item.AudioObjectKey, item.VideoObjectKey,
		item.LyricObjectKey, item.PosterObjectKey, item.AudioDurationMS, item.VideoDurationMS,
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
		&item.ID, &item.SourceNo, &item.SourceCode, &item.Title,
		&item.AudioObjectKey, &item.VideoObjectKey, &item.LyricObjectKey, &item.PosterObjectKey,
		&item.AudioDurationMS, &item.VideoDurationMS, &item.VideoWidth, &item.VideoHeight,
		&item.AudioCodec, &item.VideoCodec, &item.ValidationStatus, &item.ValidationMessage,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return model.MediaItem{}, err
	}
	item.HasAudio = item.AudioObjectKey != nil
	item.HasVideo = item.VideoObjectKey != nil
	item.HasLyric = item.LyricObjectKey != nil
	item.HasPoster = item.PosterObjectKey != nil
	return item, nil
}

func nullableText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
