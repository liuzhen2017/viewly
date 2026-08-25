-- 002: shared-table multi-tenancy.
-- Every content/config table gains tenant_id; user identity uniqueness becomes
-- per-tenant (same email/device can exist independently on each tenant site).
-- User-derived rows (coins, unlocks, tasks, favorites...) stay keyed by
-- user_id only — the user row itself is tenant-scoped and verified on auth.
-- admin_users.tenant_id is NULL for platform super admins.

SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS tenants (
  id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name       VARCHAR(64) NOT NULL,
  slug       VARCHAR(32) NOT NULL,
  status     TINYINT NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO tenants (id, name, slug) VALUES (1, 'Viewly Main', 'main')
  ON DUPLICATE KEY UPDATE name = VALUES(name);

-- ---------- content tables ----------
ALTER TABLE dramas ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0;
UPDATE dramas SET tenant_id = 1 WHERE tenant_id = 0;
ALTER TABLE dramas ADD KEY idx_tenant (tenant_id);

ALTER TABLE episodes ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0;
UPDATE episodes SET tenant_id = 1 WHERE tenant_id = 0;
ALTER TABLE episodes ADD KEY idx_tenant (tenant_id);

ALTER TABLE categories ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0;
UPDATE categories SET tenant_id = 1 WHERE tenant_id = 0;
ALTER TABLE categories DROP INDEX uk_name;
ALTER TABLE categories ADD UNIQUE KEY uk_tenant_name (tenant_id, name);

ALTER TABLE banners ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0;
UPDATE banners SET tenant_id = 1 WHERE tenant_id = 0;
ALTER TABLE banners ADD KEY idx_tenant (tenant_id);

ALTER TABLE coin_packages ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0;
UPDATE coin_packages SET tenant_id = 1 WHERE tenant_id = 0;
ALTER TABLE coin_packages ADD KEY idx_tenant (tenant_id);

ALTER TABLE vip_plans ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0;
UPDATE vip_plans SET tenant_id = 1 WHERE tenant_id = 0;
ALTER TABLE vip_plans ADD KEY idx_tenant (tenant_id);

ALTER TABLE orders ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0;
UPDATE orders SET tenant_id = 1 WHERE tenant_id = 0;
ALTER TABLE orders ADD KEY idx_tenant (tenant_id);

-- ---------- users: per-tenant identity ----------
ALTER TABLE users ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0;
UPDATE users SET tenant_id = 1 WHERE tenant_id = 0;
ALTER TABLE users DROP INDEX uk_guest_key;
ALTER TABLE users DROP INDEX uk_email;
ALTER TABLE users ADD UNIQUE KEY uk_tenant_guest (tenant_id, guest_key);
ALTER TABLE users ADD UNIQUE KEY uk_tenant_email (tenant_id, email);
ALTER TABLE users ADD KEY idx_tenant (tenant_id);

-- ---------- admins: tenant-scoped, NULL tenant = platform super admin ----------
ALTER TABLE admin_users ADD COLUMN tenant_id BIGINT UNSIGNED NULL DEFAULT NULL;
UPDATE admin_users SET tenant_id = 1 WHERE tenant_id IS NULL;
ALTER TABLE admin_users DROP INDEX uk_username;
ALTER TABLE admin_users ADD UNIQUE KEY uk_tenant_username (tenant_id, username);
ALTER TABLE admin_users ADD KEY idx_tenant (tenant_id);
UPDATE admin_users SET role = 'super', tenant_id = NULL WHERE username = 'admin';
