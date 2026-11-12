-- 归因码访问人数：同一归因码下，同一个 IP 只算一个人。
CREATE TABLE promo_code_ips (
  code_id BIGINT UNSIGNED NOT NULL,
  ip_hash VARCHAR(64) NOT NULL,
  first_seen_at DATETIME(3) NOT NULL,
  PRIMARY KEY (code_id, ip_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO promo_code_ips (code_id, ip_hash, first_seen_at)
SELECT code_id, ip_hash, MIN(created_at)
FROM decision_logs
WHERE code_id > 0 AND ip_hash <> '' AND content = 'real' AND attributed = 1
GROUP BY code_id, ip_hash;

UPDATE promo_codes p
SET visit_count = (
  SELECT COUNT(*) FROM promo_code_ips i WHERE i.code_id = p.id
);
