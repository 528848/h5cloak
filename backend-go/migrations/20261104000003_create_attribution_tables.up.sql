-- 归因参数字典、站点校验规则、分销码/活动码。

CREATE TABLE attribution_params (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  site_id BIGINT UNSIGNED NOT NULL DEFAULT 0,               -- 0 为平台内置，站点行覆盖同名内置行
  name VARCHAR(64) NOT NULL,
  param_type VARCHAR(16) NOT NULL,                          -- click, source, distributor, campaign, free
  platform VARCHAR(32) NULL,                                -- facebook, google, tiktok ...
  aliases JSON NULL,                                        -- ["ref","aff"]
  min_len INT UNSIGNED NOT NULL DEFAULT 1,
  max_len INT UNSIGNED NOT NULL DEFAULT 512,
  pattern VARCHAR(255) NULL,                                -- 可选正则
  sufficient TINYINT(1) NOT NULL DEFAULT 0,                 -- 单独出现是否足以放行
  status VARCHAR(16) NOT NULL DEFAULT 'enabled',            -- enabled, disabled
  sort INT UNSIGNED NOT NULL DEFAULT 0,
  remark VARCHAR(255) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_params_site_name (site_id, name),
  KEY idx_params_site (site_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE site_attribution_rules (
  site_id BIGINT UNSIGNED NOT NULL,
  combine_rule VARCHAR(32) NOT NULL DEFAULT 'require_any',  -- require_any, require_click_or_code
  validate_codes TINYINT(1) NOT NULL DEFAULT 1,
  validate_click_format TINYINT(1) NOT NULL DEFAULT 1,
  allowed_sources JSON NULL,                                -- utm_source 允许列表，空为不限制
  click_reuse_limit INT UNSIGNED NOT NULL DEFAULT 0,        -- 0 为不限制
  click_reuse_window_hours INT UNSIGNED NOT NULL DEFAULT 24,
  click_reuse_action VARCHAR(16) NOT NULL DEFAULT 'downgrade', -- downgrade, review
  code_revoke_mode VARCHAR(16) NOT NULL DEFAULT 'window',   -- window: 已种 cookie 保留到窗口期; immediate: 立即失效
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (site_id),
  CONSTRAINT fk_attr_rules_site FOREIGN KEY (site_id) REFERENCES sites (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE promo_codes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_id BIGINT UNSIGNED NOT NULL,
  site_id BIGINT UNSIGNED NOT NULL,
  code VARCHAR(64) NOT NULL,
  code_type VARCHAR(16) NOT NULL DEFAULT 'distributor',     -- distributor, campaign
  name VARCHAR(64) NULL,
  channel VARCHAR(64) NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'enabled',            -- enabled, disabled
  starts_at DATETIME(3) NULL,
  expires_at DATETIME(3) NULL,
  allowed_paths JSON NULL,                                  -- 前缀列表，空为不限制
  remark VARCHAR(255) NULL,
  created_by BIGINT UNSIGNED NULL,
  visit_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
  first_pass_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
  return_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
  conversion_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
  last_visit_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_codes_site_code (site_id, code),
  KEY idx_codes_site_status (site_id, status, expires_at),
  KEY idx_codes_merchant (merchant_id, created_at),
  CONSTRAINT fk_codes_site FOREIGN KEY (site_id) REFERENCES sites (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 内置点击参数与来源参数。格式规则取各平台公开的点击 ID 特征，总后台可调整。
INSERT INTO attribution_params (site_id, name, param_type, platform, min_len, max_len, pattern, sufficient, sort, remark) VALUES
(0, 'fbclid', 'click', 'facebook', 20, 255, '^[A-Za-z0-9_-]+$', 1, 10, 'Facebook / Instagram 点击 ID'),
(0, 'gclid', 'click', 'google', 20, 255, '^[A-Za-z0-9_-]+$', 1, 20, 'Google Ads 点击 ID'),
(0, 'gbraid', 'click', 'google', 20, 255, '^[A-Za-z0-9_-]+$', 1, 21, 'Google iOS 应用归因'),
(0, 'wbraid', 'click', 'google', 20, 255, '^[A-Za-z0-9_-]+$', 1, 22, 'Google iOS 网页归因'),
(0, 'ttclid', 'click', 'tiktok', 20, 255, '^[A-Za-z0-9._-]+$', 1, 30, 'TikTok 点击 ID'),
(0, 'twclid', 'click', 'twitter', 10, 255, '^[A-Za-z0-9_-]+$', 1, 40, 'X / Twitter 点击 ID'),
(0, 'msclkid', 'click', 'microsoft', 16, 64, '^[A-Za-z0-9]+$', 1, 50, 'Microsoft Ads 点击 ID'),
(0, 'kwai_click_id', 'click', 'kwai', 8, 255, '^[A-Za-z0-9_-]+$', 1, 60, 'Kwai 点击 ID'),
(0, 'kwaiclickid', 'click', 'kwai', 8, 255, '^[A-Za-z0-9_-]+$', 1, 61, 'Kwai 点击 ID（旧写法）'),
(0, 'kpid', 'click', 'kwai', 4, 128, NULL, 0, 62, 'Kwai 推广 ID，不单独放行'),
(0, 'sclid', 'click', 'snapchat', 10, 255, '^[A-Za-z0-9_-]+$', 1, 70, 'Snapchat 点击 ID'),
(0, 'li_fat_id', 'click', 'linkedin', 10, 255, '^[A-Za-z0-9_-]+$', 1, 80, 'LinkedIn 点击 ID'),
(0, 'utm_source', 'source', NULL, 1, 128, NULL, 0, 100, '来源，可配合允许来源列表'),
(0, 'utm_medium', 'source', NULL, 1, 128, NULL, 0, 101, '媒介，不单独放行'),
(0, 'utm_campaign', 'source', NULL, 1, 255, NULL, 0, 102, '活动名，不单独放行'),
(0, 'utm_content', 'source', NULL, 1, 255, NULL, 0, 103, '内容，不单独放行'),
(0, 'utm_term', 'source', NULL, 1, 255, NULL, 0, 104, '关键词，不单独放行'),
(0, 'ref', 'distributor', NULL, 2, 64, '^[A-Za-z0-9_-]+$', 1, 200, '分销码，须在登记列表内'),
(0, 'cid', 'campaign', NULL, 2, 64, '^[A-Za-z0-9_-]+$', 1, 210, '活动码，须在登记列表内');

UPDATE attribution_params SET aliases = JSON_ARRAY('r', 'aff', 'affiliate', 'referral') WHERE site_id = 0 AND name = 'ref';
UPDATE attribution_params SET aliases = JSON_ARRAY('campaign', 'promo') WHERE site_id = 0 AND name = 'cid';
