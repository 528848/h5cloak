-- 跨浏览器交接：匹配是否要求新请求来自系统浏览器（默认要求，避免 WebView 内两个人互相匹配）
ALTER TABLE sites ADD COLUMN handoff_match_system_only TINYINT(1) NOT NULL DEFAULT 1 AFTER handoff_match_minutes;
-- 决策日志记录被交接的那次放行请求
ALTER TABLE decision_logs ADD COLUMN handoff_from_request_id VARCHAR(64) NOT NULL DEFAULT '' AFTER handoff_kind;
