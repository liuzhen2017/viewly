-- 009: episode file metadata for integration APIs (cached from CDN HEAD)
ALTER TABLE episodes
  ADD COLUMN file_size BIGINT NOT NULL DEFAULT 0,
  ADD COLUMN file_etag VARCHAR(128) NOT NULL DEFAULT '',
  ADD COLUMN file_meta_at DATETIME NULL;
