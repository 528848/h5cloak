-- 看板聚合、告警、通知、Webhook、导出。聚合表全部由 worker 从 decision_logs / conversions 重算，可随时重建。

CREATE TABLE metrics_hourly (
  merchant_id BIGINT UNSIGNED NOT NULL,
  site_id BIGINT UNSIGNED NOT NULL,
  hour DATETIME NOT NULL,
  visits BIGINT UNSIGNED NOT NULL DEFAULT 0,
  real_total BIGINT UNSIGNED NOT NULL DEFAULT 0,
  first_real BIGINT UNSIGNED NOT NULL DEFAULT 0,
  return_real BIGINT UNSIGNED NOT NULL DEFAULT 0,
  handoff_real BIGINT UNSIGNED NOT NULL DEFAULT 0,
  review BIGINT UNSIGNED NOT NULL DEFAULT 0,
  block BIGINT UNSIGNED NOT NULL DEFAULT 0,
  not_found BIGINT UNSIGNED NOT NULL DEFAULT 0,
  bypass BIGINT UNSIGNED NOT NULL DEFAULT 0,
  validation_failed BIGINT UNSIGNED NOT NULL DEFAULT 0,
  in_app BIGINT UNSIGNED NOT NULL DEFAULT 0,
  degraded BIGINT UNSIGNED NOT NULL DEFAULT 0,
  origin_errors BIGINT UNSIGNED NOT NULL DEFAULT 0,
  latency_sum BIGINT UNSIGNED NOT NULL DEFAULT 0,
  conversions BIGINT UNSIGNED NOT NULL DEFAULT 0,
  conversion_amount DECIMAL(18,4) NOT NULL DEFAULT 0,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (site_id, hour),
  KEY idx_mh_merchant_hour (merchant_id, hour)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE code_stats_daily (
  site_id BIGINT UNSIGNED NOT NULL,
  code_id BIGINT UNSIGNED NOT NULL,
  day DATE NOT NULL,
  merchant_id BIGINT UNSIGNED NOT NULL,
  code VARCHAR(64) NOT NULL DEFAULT '',
  channel VARCHAR(128) NOT NULL DEFAULT '',
  visits BIGINT UNSIGNED NOT NULL DEFAULT 0,
  first_visits BIGINT UNSIGNED NOT NULL DEFAULT 0,
  return_visits BIGINT UNSIGNED NOT NULL DEFAULT 0,
  handoff_visits BIGINT UNSIGNED NOT NULL DEFAULT 0,
  in_app BIGINT UNSIGNED NOT NULL DEFAULT 0,
  conversions BIGINT UNSIGNED NOT NULL DEFAULT 0,
  conversion_amount DECIMAL(18,4) NOT NULL DEFAULT 0,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (site_id, code_id, day),
  KEY idx_csd_merchant_day (merchant_id, day)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE path_stats_daily (
  site_id BIGINT UNSIGNED NOT NULL,
  day DATE NOT NULL,
  path VARCHAR(255) NOT NULL,
  merchant_id BIGINT UNSIGNED NOT NULL,
  visits BIGINT UNSIGNED NOT NULL DEFAULT 0,
  real_total BIGINT UNSIGNED NOT NULL DEFAULT 0,
  review BIGINT UNSIGNED NOT NULL DEFAULT 0,
  block BIGINT UNSIGNED NOT NULL DEFAULT 0,
  no_attribution BIGINT UNSIGNED NOT NULL DEFAULT 0,
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (site_id, day, path),
  KEY idx_psd_merchant_day (merchant_id, day)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE alert_rules (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_id BIGINT UNSIGNED NOT NULL,
  site_id BIGINT UNSIGNED NOT NULL DEFAULT 0,        -- 0 全部站点
  type VARCHAR(32) NOT NULL,                         -- origin.unhealthy, certificate.expiring, domain.unhealthy, quota.threshold, traffic.review_spike, code.abnormal, validation.spike, decision.degraded
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  severity VARCHAR(8) NOT NULL DEFAULT 'warning',    -- info, warning, critical
  threshold JSON NULL,                               -- 各类型自定义：{ "pct": 80, "count": 500, "days": 14, "minVisits": 50 }
  window_minutes INT NOT NULL DEFAULT 15,
  cooldown_minutes INT NOT NULL DEFAULT 60,
  channels JSON NULL,                                -- ["inapp","email","telegram","webhook"]
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_rule (merchant_id, site_id, type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE alerts (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_id BIGINT UNSIGNED NOT NULL,
  site_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  rule_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  type VARCHAR(32) NOT NULL,
  severity VARCHAR(8) NOT NULL DEFAULT 'warning',
  target_key VARCHAR(255) NOT NULL DEFAULT '',       -- 去重键：origin:12 / domain:shop.example.com / code:ALICE / site:3
  title VARCHAR(255) NOT NULL,
  message TEXT NULL,
  detail JSON NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'open',        -- open, resolved
  fired_at DATETIME(3) NOT NULL,
  last_seen_at DATETIME(3) NOT NULL,
  resolved_at DATETIME(3) NULL,
  notified JSON NULL,                                -- { "inapp": "ok", "email": "skipped:not_configured", ... }
  ack_user_id BIGINT UNSIGNED NULL,
  ack_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_alerts_merchant (merchant_id, status, fired_at),
  KEY idx_alerts_target (merchant_id, type, target_key, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE notifications (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_id BIGINT UNSIGNED NOT NULL DEFAULT 0,    -- 0 为总后台
  kind VARCHAR(32) NOT NULL DEFAULT 'alert',         -- alert, alert_resolved, export, system
  severity VARCHAR(8) NOT NULL DEFAULT 'info',
  title VARCHAR(255) NOT NULL,
  content TEXT NULL,
  link VARCHAR(512) NULL,
  ref_type VARCHAR(32) NULL,
  ref_id BIGINT UNSIGNED NULL,
  read_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_notif_merchant (merchant_id, read_at, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE notify_settings (
  merchant_id BIGINT UNSIGNED NOT NULL,
  emails JSON NULL,
  telegram_chat_id VARCHAR(64) NOT NULL DEFAULT '',
  channels JSON NULL,                                -- 商户级默认开启的通知渠道
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (merchant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE webhook_endpoints (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(128) NOT NULL,
  url VARCHAR(1024) NOT NULL,
  secret VARCHAR(128) NOT NULL,
  events JSON NULL,                                  -- 订阅事件；空为全部
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  last_status SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  last_delivered_at DATETIME(3) NULL,
  failure_count INT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_wh_merchant (merchant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE webhook_deliveries (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_id BIGINT UNSIGNED NOT NULL,
  endpoint_id BIGINT UNSIGNED NOT NULL,
  event VARCHAR(64) NOT NULL,
  event_id VARCHAR(64) NOT NULL,
  payload JSON NOT NULL,
  attempt INT UNSIGNED NOT NULL DEFAULT 0,
  max_attempts INT UNSIGNED NOT NULL DEFAULT 5,
  status VARCHAR(16) NOT NULL DEFAULT 'pending',     -- pending, success, failed
  response_status SMALLINT UNSIGNED NOT NULL DEFAULT 0,
  response_body VARCHAR(1000) NOT NULL DEFAULT '',
  last_error VARCHAR(500) NOT NULL DEFAULT '',
  next_attempt_at DATETIME(3) NOT NULL,
  delivered_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_whd_due (status, next_attempt_at),
  KEY idx_whd_endpoint (endpoint_id, created_at),
  KEY idx_whd_merchant (merchant_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE exports (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  kind VARCHAR(32) NOT NULL,                         -- decision_logs, report_channel, report_code, report_campaign, report_path, report_conversion
  params JSON NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'pending',     -- pending, running, done, failed
  file_path VARCHAR(512) NOT NULL DEFAULT '',
  file_size BIGINT UNSIGNED NOT NULL DEFAULT 0,
  row_count BIGINT UNSIGNED NOT NULL DEFAULT 0,
  error VARCHAR(500) NOT NULL DEFAULT '',
  expires_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  finished_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_exports_merchant (merchant_id, created_at),
  KEY idx_exports_status (status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
