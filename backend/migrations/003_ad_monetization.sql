-- 003: per-tenant ad monetization configuration.
-- Each tenant site runs its own AdSense/AdMob accounts; the platform only
-- routes and verifies. rewarded_ad_mode:
--   off    — no rewarded ads
--   client — H5 flow: client reports completion, server enforces daily cap
--            (simple, abusable; acceptable while payouts are small)
--   ssv    — App flow: AdMob server-side verification callback, signature
--            checked against Google's public keys before crediting

SET NAMES utf8mb4;

ALTER TABLE tenants
  ADD COLUMN adsense_client VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN adsense_enabled TINYINT NOT NULL DEFAULT 0,
  ADD COLUMN rewarded_ad_mode VARCHAR(8) NOT NULL DEFAULT 'off',
  ADD COLUMN rewarded_ad_coins INT NOT NULL DEFAULT 50,
  ADD COLUMN rewarded_ad_daily_limit INT NOT NULL DEFAULT 5,
  ADD COLUMN admob_app_id VARCHAR(64) NOT NULL DEFAULT '',
  ADD COLUMN admob_rewarded_unit_id VARCHAR(64) NOT NULL DEFAULT '';

UPDATE tenants SET rewarded_ad_mode = 'client', rewarded_ad_coins = 50, rewarded_ad_daily_limit = 5 WHERE slug = 'main';

-- AdMob SSV transaction dedupe (reward is granted exactly once per ad view)
CREATE TABLE IF NOT EXISTS ad_ssv_transactions (
  transaction_id VARCHAR(64) PRIMARY KEY,
  user_id BIGINT UNSIGNED NOT NULL,
  tenant_id BIGINT UNSIGNED NOT NULL,
  coins INT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
