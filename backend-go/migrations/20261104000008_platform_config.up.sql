ALTER TABLE sites
  ADD COLUMN platform_hold TINYINT(1) NOT NULL DEFAULT 0 AFTER status;

CREATE TABLE system_settings (
  id TINYINT UNSIGNED NOT NULL PRIMARY KEY,
  emergency_review TINYINT(1) NOT NULL DEFAULT 0,
  entry_hosted TINYINT(1) NOT NULL DEFAULT 1,
  entry_api_v2 TINYINT(1) NOT NULL DEFAULT 1,
  entry_api_v1 TINYINT(1) NOT NULL DEFAULT 1,
  entry_js TINYINT(1) NOT NULL DEFAULT 1,
  log_retention_days INT NOT NULL DEFAULT 90,
  rate_per_key INT NOT NULL DEFAULT 50,
  rate_per_site INT NOT NULL DEFAULT 2000,
  webhook_max_attempts INT NOT NULL DEFAULT 5,
  ad_referers JSON NULL,
  inapp_markers JSON NULL,
  default_path_rules JSON NULL,
  updated_at DATETIME(3) NULL
);

INSERT INTO system_settings (id) VALUES (1);
