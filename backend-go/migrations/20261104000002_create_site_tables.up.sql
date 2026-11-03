-- 站点、域名、源站、审核内容、路径规则。

CREATE TABLE sites (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(64) NOT NULL,
  site_type VARCHAR(16) NOT NULL DEFAULT 'hybrid',          -- hosted, server, hybrid
  status VARCHAR(32) NOT NULL DEFAULT 'enabled',            -- enabled, disabled
  remark VARCHAR(500) NULL,
  publishable_key VARCHAR(64) NOT NULL,
  secret_prefix VARCHAR(16) NOT NULL,
  secret_hash VARCHAR(255) NOT NULL,
  secret_ip_allowlist JSON NULL,
  secret_expires_at DATETIME(3) NULL,
  cookie_name VARCHAR(64) NOT NULL DEFAULT '_ck_vid',
  cookie_domain VARCHAR(255) NULL,
  attribution_ttl_days INT UNSIGNED NOT NULL DEFAULT 30,
  renew_mode VARCHAR(16) NOT NULL DEFAULT 'sliding',        -- fixed, sliding
  no_attribution_action VARCHAR(16) NOT NULL DEFAULT 'review', -- review, block, not_found
  fail_mode VARCHAR(16) NOT NULL DEFAULT 'review',          -- review, real, maintenance
  min_confidence VARCHAR(16) NOT NULL DEFAULT 'medium',     -- low, medium, high
  handoff_banner TINYINT(1) NOT NULL DEFAULT 1,
  handoff_banner_text VARCHAR(120) NOT NULL DEFAULT '在浏览器中打开，继续访问',
  handoff_link_rewrite TINYINT(1) NOT NULL DEFAULT 0,
  handoff_token_ttl_hours INT UNSIGNED NOT NULL DEFAULT 24,
  handoff_token_max_uses INT UNSIGNED NOT NULL DEFAULT 3,
  handoff_match_enabled TINYINT(1) NOT NULL DEFAULT 1,
  handoff_match_minutes INT UNSIGNED NOT NULL DEFAULT 15,
  token_key_version INT UNSIGNED NOT NULL DEFAULT 1,
  token_keys JSON NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_sites_pk (publishable_key),
  KEY idx_sites_secret_prefix (secret_prefix),
  KEY idx_sites_merchant_status (merchant_id, status),
  CONSTRAINT fk_sites_merchant FOREIGN KEY (merchant_id) REFERENCES merchants (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE site_origins (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  site_id BIGINT UNSIGNED NOT NULL,
  kind VARCHAR(16) NOT NULL,                                -- real, review
  url VARCHAR(1000) NOT NULL,
  host_header VARCHAR(255) NULL,
  timeout_ms INT UNSIGNED NOT NULL DEFAULT 15000,
  health_status VARCHAR(16) NOT NULL DEFAULT 'unknown',     -- unknown, healthy, unhealthy
  health_message VARCHAR(255) NULL,
  last_health_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_origins_site_kind (site_id, kind),
  CONSTRAINT fk_origins_site FOREIGN KEY (site_id) REFERENCES sites (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE review_templates (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  code VARCHAR(64) NOT NULL,
  name VARCHAR(64) NOT NULL,
  description VARCHAR(255) NULL,
  html MEDIUMTEXT NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_templates_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE static_packages (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  site_id BIGINT UNSIGNED NOT NULL,
  version INT UNSIGNED NOT NULL,
  object_key VARCHAR(500) NOT NULL,
  entry VARCHAR(255) NOT NULL DEFAULT 'index.html',
  status VARCHAR(16) NOT NULL DEFAULT 'ready',              -- ready, archived
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_packages_site_ver (site_id, version),
  CONSTRAINT fk_packages_site FOREIGN KEY (site_id) REFERENCES sites (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE site_review_contents (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  site_id BIGINT UNSIGNED NOT NULL,
  source VARCHAR(16) NOT NULL DEFAULT 'template',           -- origin, template, static
  template_id BIGINT UNSIGNED NULL,
  template_vars JSON NULL,
  static_package_id BIGINT UNSIGNED NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_review_site (site_id),
  CONSTRAINT fk_review_site FOREIGN KEY (site_id) REFERENCES sites (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE domains (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_id BIGINT UNSIGNED NOT NULL,
  site_id BIGINT UNSIGNED NOT NULL,
  domain VARCHAR(255) NOT NULL,
  domain_type VARCHAR(16) NOT NULL DEFAULT 'hosted',        -- hosted, own
  verify_type VARCHAR(16) NOT NULL DEFAULT 'CNAME',         -- CNAME, TXT
  verify_value VARCHAR(255) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',            -- pending, bound, abnormal, disabled
  verify_message VARCHAR(255) NULL,
  last_checked_at DATETIME(3) NULL,
  cert_status VARCHAR(32) NOT NULL DEFAULT 'none',
  cert_expires_at DATETIME(3) NULL,
  health_status VARCHAR(16) NOT NULL DEFAULT 'unknown',
  remark VARCHAR(255) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_domains_domain (domain),
  KEY idx_domains_site (site_id),
  KEY idx_domains_merchant_status (merchant_id, status),
  CONSTRAINT fk_domains_site FOREIGN KEY (site_id) REFERENCES sites (id) ON DELETE CASCADE,
  CONSTRAINT fk_domains_merchant FOREIGN KEY (merchant_id) REFERENCES merchants (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO review_templates (code, name, description, html) VALUES
('shop', '合规商城', '简洁的商品展示页，适合作为审核内容', '<!doctype html><html lang="zh"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.siteName}}</title><style>body{margin:0;font-family:sans-serif;background:#f6f7f9;color:#1f2329}header{padding:28px 20px;background:#1f4b99;color:#fff}main{max-width:880px;margin:24px auto;padding:0 16px}article{background:#fff;border-radius:12px;padding:20px;margin-bottom:12px}</style></head><body><header><h1>{{.siteName}}</h1><p>{{.slogan}}</p></header><main><article><h2>精选商品</h2><p>品质生活，安心选购。如需了解更多，请通过页面公示的联系方式与我们沟通。</p></article><article><h2>配送与售后</h2><p>订单支付后按承诺时效发货，支持查询物流与售后申请。</p></article></main></body></html>'),
('company', '企业官网', '公司介绍页，适合品牌站审核内容', '<!doctype html><html lang="zh"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.siteName}}</title><style>body{margin:0;font-family:sans-serif;background:#fff;color:#222}header{padding:48px 20px;border-bottom:1px solid #eee}main{max-width:760px;margin:32px auto;padding:0 16px;line-height:1.7}</style></head><body><header><h1>{{.siteName}}</h1><p>{{.slogan}}</p></header><main><p>我们专注于为客户提供稳定、透明的产品与服务。本页介绍公司业务范围、服务流程与联系方式。</p><p>业务咨询请使用官网公示邮箱，我们将在工作时间内回复。</p></main></body></html>');

CREATE TABLE site_path_rules (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  site_id BIGINT UNSIGNED NOT NULL,
  pattern VARCHAR(255) NOT NULL,
  match_type VARCHAR(16) NOT NULL DEFAULT 'prefix',         -- prefix, exact, suffix
  action VARCHAR(32) NOT NULL,                              -- bypass, always_real, always_review, block
  sort INT UNSIGNED NOT NULL DEFAULT 0,
  remark VARCHAR(255) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_path_rules_site (site_id, sort),
  CONSTRAINT fk_path_rules_site FOREIGN KEY (site_id) REFERENCES sites (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
