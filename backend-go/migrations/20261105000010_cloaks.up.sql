ALTER TABLE merchants
  ADD COLUMN cloak_quota INT UNSIGNED NOT NULL DEFAULT 20 AFTER code_quota;

CREATE TABLE cloaks (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_id BIGINT UNSIGNED NOT NULL,
  domain_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'enabled',
  real_urls JSON NOT NULL,
  review_mode VARCHAR(32) NOT NULL DEFAULT 'external',
  review_url VARCHAR(1000) NOT NULL DEFAULT '',
  review_tag VARCHAR(32) NOT NULL DEFAULT '',
  code VARCHAR(32) NOT NULL,
  code_custom TINYINT(1) NOT NULL DEFAULT 0,
  allowed_sources JSON NOT NULL,
  mode VARCHAR(32) NOT NULL DEFAULT 'cloak',
  target_regions JSON NOT NULL,
  spider_whitelist JSON NOT NULL,
  device_whitelist VARCHAR(32) NOT NULL DEFAULT '',
  block_empty_language TINYINT(1) NOT NULL DEFAULT 0,
  block_pc TINYINT(1) NOT NULL DEFAULT 0,
  block_proxy TINYINT(1) NOT NULL DEFAULT 0,
  intermediate_page VARCHAR(32) NOT NULL DEFAULT 'none',
  intermediate_text VARCHAR(500) NOT NULL DEFAULT '',
  remark VARCHAR(200) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_cloaks_code (code),
  KEY idx_cloaks_merchant_status (merchant_id, status),
  KEY idx_cloaks_domain_status (domain_id, status),
  CONSTRAINT fk_cloaks_merchant FOREIGN KEY (merchant_id) REFERENCES merchants (id),
  CONSTRAINT fk_cloaks_domain FOREIGN KEY (domain_id) REFERENCES domains (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE cloak_access_logs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_id BIGINT UNSIGNED NULL,
  cloak_id BIGINT UNSIGNED NULL,
  code VARCHAR(32) NOT NULL DEFAULT '',
  country VARCHAR(8) NOT NULL DEFAULT '',
  device VARCHAR(16) NOT NULL DEFAULT '',
  hit_rule VARCHAR(64) NOT NULL DEFAULT '',
  final_action VARCHAR(32) NOT NULL,
  latency_ms INT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_cloak_logs_cloak_time (cloak_id, created_at),
  KEY idx_cloak_logs_merchant_time (merchant_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
