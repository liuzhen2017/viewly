-- 006: TKPostAI promote-link cache/history
CREATE TABLE IF NOT EXISTS tk_entries (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  tenant_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  drama_id BIGINT UNSIGNED NOT NULL,
  entry_id VARCHAR(128) NOT NULL DEFAULT '',
  url VARCHAR(500) NOT NULL DEFAULT '',
  expires_at DATETIME NOT NULL,
  remaining_uses INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_tenant_user (tenant_id, user_id),
  KEY idx_lookup (tenant_id, user_id, drama_id, expires_at)
);
