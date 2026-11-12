DROP TABLE IF EXISTS review_template_files;

ALTER TABLE review_templates
  DROP COLUMN updated_at,
  DROP COLUMN active_version,
  DROP COLUMN spa_fallback,
  DROP COLUMN entry,
  DROP COLUMN timeout_ms,
  DROP COLUMN host_header,
  DROP COLUMN url,
  DROP COLUMN status,
  DROP COLUMN kind;
