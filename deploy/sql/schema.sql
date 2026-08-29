CREATE DATABASE IF NOT EXISTS kids_media
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;

USE kids_media;
SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS media_item (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  source_no INT UNSIGNED NOT NULL COMMENT '文件名前导数字，用于跨 Bucket 关联',
  source_code VARCHAR(32) NOT NULL COMMENT '保留前导零的展示编号',
  title VARCHAR(255) NOT NULL,
  audio_object_key VARCHAR(1024) NULL,
  video_480_object_key VARCHAR(1024) NULL,
  video_720_object_key VARCHAR(1024) NULL,
  lyrics_en_object_key VARCHAR(1024) NULL,
  lyrics_zh_object_key VARCHAR(1024) NULL,
  lyrics_bilingual_object_key VARCHAR(1024) NULL,
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

CREATE TABLE IF NOT EXISTS media_catalog (
  source_no INT UNSIGNED NOT NULL,
  title_zh VARCHAR(255) NOT NULL,
  PRIMARY KEY (source_no),
  KEY idx_media_catalog_title_zh (title_zh)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS media_tag (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  slug VARCHAR(64) NOT NULL,
  name VARCHAR(32) NOT NULL,
  icon VARCHAR(16) NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  min_items INT UNSIGNED NOT NULL DEFAULT 8,
  PRIMARY KEY (id),
  UNIQUE KEY uk_media_tag_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS media_catalog_tag (
  source_no INT UNSIGNED NOT NULL,
  tag_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (source_no, tag_id),
  KEY idx_media_catalog_tag_tag (tag_id, source_no),
  CONSTRAINT fk_media_catalog_tag_catalog FOREIGN KEY (source_no) REFERENCES media_catalog (source_no) ON DELETE CASCADE,
  CONSTRAINT fk_media_catalog_tag_tag FOREIGN KEY (tag_id) REFERENCES media_tag (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO media_tag (slug, name, icon, sort_order, min_items) VALUES
  ('animals', '动物', 'animals', 10, 8),
  ('numbers', '数字', 'numbers', 20, 8),
  ('colors', '颜色', 'colors', 30, 8),
  ('holidays', '节日', 'holidays', 40, 8),
  ('vehicles', '交通工具', 'vehicles', 50, 8),
  ('bedtime', '睡前', 'bedtime', 60, 8),
  ('alphabet', '字母', 'alphabet', 70, 8),
  ('movement', '身体与运动', 'movement', 80, 8),
  ('routines', '生活习惯', 'routines', 90, 8),
  ('food', '食物', 'food', 100, 8),
  ('nature', '自然天气', 'nature', 110, 8),
  ('friends', '朋友与问候', 'friends', 120, 8)
ON DUPLICATE KEY UPDATE
  name = VALUES(name), icon = VALUES(icon), sort_order = VALUES(sort_order), min_items = VALUES(min_items);
