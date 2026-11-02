-- 011: TK clone MVP — user-owned TikTok assets, 7-day auth links,
-- batch publish tasks, ffmpeg slice jobs.
-- user_id NULL on tiktok_accounts/clips/posts = legacy admin/tenant-owned rows.

ALTER TABLE tiktok_accounts
  ADD COLUMN user_id BIGINT UNSIGNED NULL AFTER tenant_id,
  ADD KEY idx_tenant_user (tenant_id, user_id);

ALTER TABLE tiktok_clips
  ADD COLUMN user_id BIGINT UNSIGNED NULL AFTER tenant_id;

ALTER TABLE tiktok_posts
  ADD COLUMN task_id BIGINT UNSIGNED NULL AFTER tenant_id,
  ADD COLUMN user_id BIGINT UNSIGNED NULL AFTER task_id,
  ADD COLUMN scheduled_at DATETIME NULL AFTER error,
  ADD COLUMN checked_at DATETIME NULL AFTER scheduled_at,
  ADD COLUMN disable_comment TINYINT NOT NULL DEFAULT 0,
  ADD COLUMN disable_duet TINYINT NOT NULL DEFAULT 0,
  ADD COLUMN disable_stitch TINYINT NOT NULL DEFAULT 0,
  ADD KEY idx_task (task_id),
  ADD KEY idx_user_status (user_id, status);

-- OAuth state store: direct connects (kind=user, single-use, 30 min) and
-- shareable authorization links (kind=link, 7 days, many accounts can join
-- the owner's matrix through the same link).
CREATE TABLE IF NOT EXISTS tk_auth_states (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  tenant_id BIGINT UNSIGNED NOT NULL,
  state VARCHAR(64) NOT NULL,
  kind VARCHAR(8) NOT NULL,               -- user | link
  owner_user_id BIGINT UNSIGNED NOT NULL,
  title VARCHAR(255) NOT NULL DEFAULT '',
  expires_at DATETIME NOT NULL,
  used_at DATETIME NULL,
  use_count INT NOT NULL DEFAULT 0,
  used_by_open_id VARCHAR(64) NOT NULL DEFAULT '',
  used_by_name VARCHAR(128) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_state (state),
  KEY idx_owner (tenant_id, owner_user_id)
);

-- batch publish task header; the per-account/clip jobs live in tiktok_posts (task_id)
CREATE TABLE IF NOT EXISTS tk_tasks (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  tenant_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(255) NOT NULL DEFAULT '',
  rule VARCHAR(20) NOT NULL DEFAULT 'ONE_TO_ONE',   -- ONE_TO_ONE | ALL_VIDEOS | RANDOM_EVEN
  total INT NOT NULL DEFAULT 0,
  status VARCHAR(20) NOT NULL DEFAULT 'running',    -- scheduled | running | done | canceled
  scheduled_at DATETIME NULL,
  privacy_level VARCHAR(32) NOT NULL DEFAULT 'SELF_ONLY',
  title VARCHAR(2200) NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_user (tenant_id, user_id)
);

-- ffmpeg slice jobs: one episode -> N short clips in the owner's library
CREATE TABLE IF NOT EXISTS tk_slice_jobs (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  tenant_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  episode_id BIGINT UNSIGNED NOT NULL,
  seg_sec INT NOT NULL DEFAULT 60,
  max_count INT NOT NULL DEFAULT 10,
  status VARCHAR(20) NOT NULL DEFAULT 'queued',     -- queued | running | done | failed
  error VARCHAR(500) NOT NULL DEFAULT '',
  clip_count INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_user (tenant_id, user_id)
);
