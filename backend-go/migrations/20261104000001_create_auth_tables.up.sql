-- 账号、角色、菜单、商户基础表。字段与 V1 Laravel 结构兼容，便于 I13 迁移。

CREATE TABLE merchants (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  contact_name VARCHAR(255) NULL,
  contact_email VARCHAR(255) NULL,
  contact_phone VARCHAR(64) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'enabled',          -- enabled, disabled, expired, frozen
  expire_at DATETIME(3) NULL,
  timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
  decision_quota BIGINT UNSIGNED NOT NULL DEFAULT 1000000, -- 判定次数/月
  site_quota INT UNSIGNED NOT NULL DEFAULT 5,
  domain_quota INT UNSIGNED NOT NULL DEFAULT 20,
  code_quota INT UNSIGNED NOT NULL DEFAULT 1000,
  user_quota INT UNSIGNED NOT NULL DEFAULT 10,
  log_retention_days INT UNSIGNED NOT NULL DEFAULT 90,
  quota_policy VARCHAR(32) NOT NULL DEFAULT 'alert',       -- alert, review_all, deny
  features JSON NULL,
  remark VARCHAR(500) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_merchants_status (status),
  KEY idx_merchants_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE users (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_id BIGINT UNSIGNED NULL,
  username VARCHAR(64) NOT NULL,
  nickname VARCHAR(255) NULL,
  password VARCHAR(255) NOT NULL,
  avatar VARCHAR(500) NULL,
  email VARCHAR(255) NULL,
  phone VARCHAR(32) NULL,
  sex VARCHAR(8) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'enabled',
  last_login_at DATETIME(3) NULL,
  last_login_ip VARCHAR(64) NULL,
  password_changed_at DATETIME(3) NULL,
  remark VARCHAR(500) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_users_username (username),
  KEY idx_users_merchant_status (merchant_id, status),
  CONSTRAINT fk_users_merchant FOREIGN KEY (merchant_id) REFERENCES merchants (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE roles (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(64) NOT NULL,
  role_key VARCHAR(64) NOT NULL,
  scope VARCHAR(32) NOT NULL DEFAULT 'merchant',           -- platform, merchant
  merchant_id BIGINT UNSIGNED NULL,                         -- 商户自定义角色归属，平台角色为空
  status VARCHAR(32) NOT NULL DEFAULT 'enabled',
  sort INT UNSIGNED NOT NULL DEFAULT 0,
  remark VARCHAR(500) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_roles_key (role_key),
  KEY idx_roles_scope_status (scope, status),
  KEY idx_roles_merchant (merchant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE menus (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  parent_id BIGINT UNSIGNED NULL,
  portal VARCHAR(16) NOT NULL DEFAULT 'admin',              -- admin, merchant
  name VARCHAR(64) NOT NULL,
  path VARCHAR(255) NULL,
  component VARCHAR(255) NULL,
  query VARCHAR(255) NULL,
  menu_type CHAR(1) NOT NULL DEFAULT 'C',                   -- M 目录, C 菜单, F 按钮
  visible TINYINT(1) NOT NULL DEFAULT 1,
  status VARCHAR(32) NOT NULL DEFAULT 'enabled',
  perms VARCHAR(128) NULL,
  icon VARCHAR(64) NULL,
  sort INT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_menus_parent (parent_id),
  KEY idx_menus_portal_type (portal, menu_type),
  KEY idx_menus_perms (perms)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE user_roles (
  user_id BIGINT UNSIGNED NOT NULL,
  role_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (user_id, role_id),
  CONSTRAINT fk_user_roles_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT fk_user_roles_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE role_menus (
  role_id BIGINT UNSIGNED NOT NULL,
  menu_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (role_id, menu_id),
  CONSTRAINT fk_role_menus_role FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE,
  CONSTRAINT fk_role_menus_menu FOREIGN KEY (menu_id) REFERENCES menus (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE login_logs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  username VARCHAR(64) NULL,
  user_id BIGINT UNSIGNED NULL,
  merchant_id BIGINT UNSIGNED NULL,
  portal VARCHAR(16) NULL,
  ip VARCHAR(64) NULL,
  location VARCHAR(128) NULL,
  browser VARCHAR(128) NULL,
  os VARCHAR(128) NULL,
  status VARCHAR(32) NOT NULL,                              -- success, failed, locked, logout
  message VARCHAR(255) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_login_logs_username (username),
  KEY idx_login_logs_created (created_at),
  KEY idx_login_logs_merchant (merchant_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE operation_logs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  request_id VARCHAR(64) NULL,
  actor_user_id BIGINT UNSIGNED NULL,
  actor_username VARCHAR(64) NULL,
  actor_role VARCHAR(255) NULL,
  portal VARCHAR(16) NULL,
  merchant_id BIGINT UNSIGNED NULL,
  module VARCHAR(64) NOT NULL,
  action VARCHAR(64) NOT NULL,
  method VARCHAR(16) NOT NULL,
  path VARCHAR(255) NOT NULL,
  object_id VARCHAR(64) NULL,
  before_snapshot JSON NULL,
  after_snapshot JSON NULL,
  request_summary JSON NULL,
  status_code INT NOT NULL DEFAULT 200,
  result_code INT NOT NULL DEFAULT 200,
  ip VARCHAR(64) NULL,
  latency_ms INT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_oplogs_request (request_id),
  KEY idx_oplogs_module_action (module, action),
  KEY idx_oplogs_merchant_created (merchant_id, created_at),
  KEY idx_oplogs_actor_created (actor_user_id, created_at),
  KEY idx_oplogs_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
