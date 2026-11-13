-- 引导弹框的延迟、冷却、次数和两端文案。NULL 表示沿用预设：10 秒、30 分钟、3 次。
ALTER TABLE sites
  ADD COLUMN handoff_prompt JSON NULL AFTER handoff_banner_text;
