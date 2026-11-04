-- 决策日志、归因、访客、计量。本地用 MySQL；测试服务器日志可切 ClickHouse（仓库接口相同）。

CREATE TABLE decision_logs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at DATETIME(3) NOT NULL,
  request_id VARCHAR(64) NOT NULL,
  merchant_id BIGINT UNSIGNED NOT NULL,
  site_id BIGINT UNSIGNED NOT NULL,
  entry VARCHAR(16) NOT NULL,                 -- hosted, api_v2, api_v1, js
  method VARCHAR(8) NOT NULL,
  host VARCHAR(255) NOT NULL,
  path VARCHAR(1000) NOT NULL,
  query_param_names JSON NULL,
  touch_type VARCHAR(16) NOT NULL,
  content VARCHAR(16) NOT NULL,
  attributed TINYINT(1) NOT NULL DEFAULT 0,
  confidence VARCHAR(8) NOT NULL DEFAULT '',
  in_app_browser VARCHAR(32) NOT NULL DEFAULT '',
  device_family VARCHAR(64) NOT NULL DEFAULT '',
  handoff_kind VARCHAR(16) NOT NULL DEFAULT '',
  channel VARCHAR(128) NOT NULL DEFAULT '',
  code_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  code VARCHAR(64) NOT NULL DEFAULT '',
  matched_params JSON NULL,
  validation_errors JSON NULL,
  visitor_id VARCHAR(64) NOT NULL DEFAULT '',
  attribution_id VARCHAR(64) NOT NULL DEFAULT '',
  attribution_expires_at DATETIME NULL,
  ip_hash VARCHAR(64) NOT NULL DEFAULT '',
  ip_prefix VARCHAR(64) NOT NULL DEFAULT '',
  country VARCHAR(2) NOT NULL DEFAULT '',
  os VARCHAR(16) NOT NULL DEFAULT '',
  ua_summary VARCHAR(255) NOT NULL DEFAULT '',
  referer_summary VARCHAR(255) NOT NULL DEFAULT '',
  path_rule VARCHAR(255) NOT NULL DEFAULT '',
  origin_kind VARCHAR(16) NOT NULL DEFAULT '',
  origin_status SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  fallback TINYINT(1) NOT NULL DEFAULT 0,
  latency_ms INT UNSIGNED NOT NULL DEFAULT 0,
  degraded TINYINT(1) NOT NULL DEFAULT 0,
  trace JSON NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_logs_request (request_id),
  KEY idx_logs_site_time (site_id, created_at),
  KEY idx_logs_merchant_time (merchant_id, created_at),
  KEY idx_logs_visitor (visitor_id),
  KEY idx_logs_code (code_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE visitors (
  site_id BIGINT UNSIGNED NOT NULL,
  visitor_id VARCHAR(64) NOT NULL,
  merchant_id BIGINT UNSIGNED NOT NULL,
  first_seen_at DATETIME(3) NOT NULL,
  last_seen_at DATETIME(3) NOT NULL,
  visit_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
  first_country VARCHAR(2) NOT NULL DEFAULT '',
  first_device_family VARCHAR(64) NOT NULL DEFAULT '',
  first_in_app VARCHAR(32) NOT NULL DEFAULT '',
  PRIMARY KEY (site_id, visitor_id),
  KEY idx_visitors_merchant (merchant_id, last_seen_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE attributions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  attribution_id VARCHAR(64) NOT NULL,
  merchant_id BIGINT UNSIGNED NOT NULL,
  site_id BIGINT UNSIGNED NOT NULL,
  visitor_id VARCHAR(64) NOT NULL,
  first_channel VARCHAR(128) NOT NULL DEFAULT '',
  first_code_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  first_params JSON NULL,
  first_landing_path VARCHAR(1000) NOT NULL DEFAULT '',
  first_request_id VARCHAR(64) NOT NULL DEFAULT '',
  first_touch_at DATETIME(3) NOT NULL,
  last_channel VARCHAR(128) NOT NULL DEFAULT '',
  last_code_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  last_touch_at DATETIME(3) NOT NULL,
  touch_count BIGINT UNSIGNED NOT NULL DEFAULT 1,
  handoff_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
  confidence VARCHAR(8) NOT NULL DEFAULT '',
  expires_at DATETIME NULL,
  merged_into_visitor_id VARCHAR(64) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_attr_site_visitor (site_id, visitor_id),
  KEY idx_attr_id (attribution_id),
  KEY idx_attr_expires (expires_at),
  KEY idx_attr_last_code (site_id, last_code_id),
  KEY idx_attr_merchant_time (merchant_id, first_touch_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE usage_counters (
  merchant_id BIGINT UNSIGNED NOT NULL,
  metric VARCHAR(32) NOT NULL,                -- decisions
  period VARCHAR(8) NOT NULL,                 -- yyyymm
  value BIGINT UNSIGNED NOT NULL DEFAULT 0,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (merchant_id, metric, period)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE api_logs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  created_at DATETIME(3) NOT NULL,
  request_id VARCHAR(64) NOT NULL,
  merchant_id BIGINT UNSIGNED NOT NULL,
  site_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  key_prefix VARCHAR(16) NOT NULL DEFAULT '',
  endpoint VARCHAR(128) NOT NULL,
  status SMALLINT UNSIGNED NOT NULL,
  error_code VARCHAR(64) NOT NULL DEFAULT '',
  latency_ms INT UNSIGNED NOT NULL DEFAULT 0,
  ip VARCHAR(45) NOT NULL DEFAULT '',
  PRIMARY KEY (id),
  KEY idx_api_merchant_time (merchant_id, created_at),
  KEY idx_api_request (request_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
