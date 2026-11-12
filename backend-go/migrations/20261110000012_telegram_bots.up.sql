-- 商户专属 Telegram 机器人：一个商户一条。Token 加密存储。
CREATE TABLE telegram_bots (
  merchant_id       BIGINT UNSIGNED NOT NULL,
  bot_token_enc     VARBINARY(512) NOT NULL,
  bot_user_id       BIGINT       NOT NULL DEFAULT 0,
  bot_username      VARCHAR(64)  NOT NULL DEFAULT '',
  status            VARCHAR(16)  NOT NULL DEFAULT 'active',
  last_update_id    BIGINT       NOT NULL DEFAULT 0,
  daily_report      TINYINT(1)   NOT NULL DEFAULT 0,
  daily_report_hour TINYINT      NOT NULL DEFAULT 9,
  daily_report_site BIGINT UNSIGNED NOT NULL DEFAULT 0,
  last_report_on    DATE NULL,
  created_at        DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at        DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (merchant_id),
  UNIQUE KEY uk_tg_bot_user (bot_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 已绑定的接收会话：一个商户可多条（私聊 + 若干群）。
CREATE TABLE telegram_chats (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  merchant_id BIGINT UNSIGNED NOT NULL,
  chat_id     VARCHAR(64) NOT NULL,
  chat_type   VARCHAR(16) NOT NULL DEFAULT 'private',
  title       VARCHAR(128) NOT NULL DEFAULT '',
  status      VARCHAR(16) NOT NULL DEFAULT 'active',
  created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_tg_merchant_chat (merchant_id, chat_id),
  KEY idx_tg_merchant (merchant_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
