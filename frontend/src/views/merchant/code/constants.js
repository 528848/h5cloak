export const codeTypeOptions = [
  { value: 'distributor', label: '分销码', type: 'success' },
  { value: 'campaign', label: '活动码', type: 'warning' }
];

export const codeStatusOptions = [
  { value: 'enabled', label: '有效', type: 'success' },
  { value: 'disabled', label: '已停用', type: 'info' },
  { value: 'expired', label: '已过期', type: 'danger' },
  { value: 'pending', label: '未生效', type: 'warning' }
];

export const getOptionLabel = (options, value) =>
  options.find((o) => o.value === value)?.label ?? value ?? '-';

export const getOptionType = (options, value) =>
  options.find((o) => o.value === value)?.type ?? 'info';
