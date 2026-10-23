-- 007: user sync status for the TKPostAI promote platform
ALTER TABLE users ADD COLUMN tk_synced TINYINT NOT NULL DEFAULT 0;
