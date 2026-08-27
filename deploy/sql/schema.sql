CREATE DATABASE IF NOT EXISTS kids_media
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;

USE kids_media;

CREATE TABLE IF NOT EXISTS media_item (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  source_no INT UNSIGNED NOT NULL COMMENT '文件名前导数字，用于跨目录关联',
  source_code VARCHAR(32) NOT NULL COMMENT '保留前导零的展示编号',
  title VARCHAR(255) NOT NULL,
  audio_object_key VARCHAR(1024) NULL,
  video_object_key VARCHAR(1024) NULL,
  lyric_object_key VARCHAR(1024) NULL,
  poster_object_key VARCHAR(1024) NULL,
  audio_duration_ms BIGINT UNSIGNED NULL,
  video_duration_ms BIGINT UNSIGNED NULL,
  video_width INT UNSIGNED NULL,
  video_height INT UNSIGNED NULL,
  audio_codec VARCHAR(64) NULL,
  video_codec VARCHAR(64) NULL,
  validation_status VARCHAR(20) NOT NULL DEFAULT 'partial',
  validation_message TEXT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_media_item_source_no (source_no),
  KEY idx_media_item_title (title),
  KEY idx_media_item_status (validation_status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
