export const severityOptions = [
  { value: 'info', label: '提示', type: 'info' },
  { value: 'warning', label: '警告', type: 'warning' },
  { value: 'critical', label: '严重', type: 'danger' }
];

export const alertStatusOptions = [
  { value: 'open', label: '未恢复', type: 'danger' },
  { value: 'resolved', label: '已恢复', type: 'success' }
];

export const channelOptions = [
  { value: 'inapp', label: '站内' },
  { value: 'email', label: '邮件' },
  { value: 'telegram', label: 'Telegram' },
  { value: 'webhook', label: 'Webhook' }
];

export const severityLabel = (value) =>
  severityOptions.find((item) => item.value === value)?.label || value || '-';

export const severityType = (value) =>
  severityOptions.find((item) => item.value === value)?.type || 'info';

export const statusLabel = (value) =>
  alertStatusOptions.find((item) => item.value === value)?.label || value || '-';

export const statusType = (value) =>
  alertStatusOptions.find((item) => item.value === value)?.type || 'info';

export const channelLabel = (key) => {
  const raw = String(key || '');
  const resolved = raw.endsWith(':resolved');
  const name = raw.replace(':resolved', '');
  const label = channelOptions.find((item) => item.value === name)?.label || name;
  return resolved ? `${label}（恢复）` : label;
};

export const kindLabel = (value) =>
  ({ alert: '告警', alert_resolved: '恢复', export: '导出', system: '系统' })[value] ||
  value ||
  '-';

export const thresholdText = (rule) => {
  const threshold = rule?.threshold || {};
  const parts = [];
  if (threshold.pct) parts.push(`阈值 ${threshold.pct}%`);
  if (threshold.count) parts.push(`阈值 ${threshold.count} 次`);
  if (threshold.days) parts.push(`提前 ${threshold.days} 天`);
  if (threshold.minVisits) parts.push(`最少 ${threshold.minVisits} 次样本`);
  return parts.join('，') || '-';
};

export const channelText = (channels) =>
  (channels || []).map((item) => channelLabel(item)).join('、') || '无';
