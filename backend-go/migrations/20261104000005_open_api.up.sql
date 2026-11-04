-- 开放接口：密钥加密副本（用于 HMAC 验签）、转化回传、访客邮箱映射。

ALTER TABLE sites ADD COLUMN secret_enc VARBINARY(255) NULL AFTER secret_hash;

CREATE TABLE conversions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_id BIGINT UNSIGNED NOT NULL,
  site_id BIGINT UNSIGNED NOT NULL,
  order_no VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'paid',            -- paid, refunded, pending
  amount DECIMAL(18,4) NOT NULL DEFAULT 0,
  currency VARCHAR(8) NOT NULL DEFAULT 'USD',
  visitor_id VARCHAR(64) NOT NULL DEFAULT '',
  attribution_id VARCHAR(64) NOT NULL DEFAULT '',
  code_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  channel VARCHAR(128) NOT NULL DEFAULT '',
  email_hash VARCHAR(64) NOT NULL DEFAULT '',
  occurred_at DATETIME(3) NOT NULL,
  request_id VARCHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_conv_order (merchant_id, order_no, status),
  KEY idx_conv_site_time (site_id, occurred_at),
  KEY idx_conv_visitor (site_id, visitor_id),
  KEY idx_conv_code (code_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE visitor_emails (
  site_id BIGINT UNSIGNED NOT NULL,
  email_hash VARCHAR(64) NOT NULL,
  visitor_id VARCHAR(64) NOT NULL,
  first_seen_at DATETIME(3) NOT NULL,
  PRIMARY KEY (site_id, email_hash, visitor_id),
  KEY idx_ve_visitor (site_id, visitor_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
