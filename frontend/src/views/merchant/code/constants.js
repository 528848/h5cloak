export const codeTypeOptions = [
  { value: 'distributor', label: '归因码', type: 'success' }
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
