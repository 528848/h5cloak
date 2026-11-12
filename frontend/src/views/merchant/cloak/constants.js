export const modeOptions = [
  { value: 'cloak', label: '斗篷', type: 'primary' },
  { value: 'green', label: '绿色', type: 'success' },
  { value: 'full_open', label: '全开', type: 'warning' },
  { value: 'review', label: '审核', type: 'info' }
];

export const statusOptions = [
  { value: 'enabled', label: '启用', type: 'success' },
  { value: 'disabled', label: '停用', type: 'info' }
];

export const sourceOptions = [
  { value: 'all', label: '不限制' },
  { value: 'facebook', label: 'Facebook' },
  { value: 'google', label: 'Google' },
  { value: 'tiktok', label: 'TikTok' },
  { value: 'kwai', label: 'Kwai' }
];

export const reviewTagOptions = [
  { value: 'google', label: 'Google' },
  { value: 'facebook', label: 'Facebook' },
  { value: 'tiktok', label: 'TikTok' },
  { value: 'generic', label: '通用安全页' }
];

export const deviceOptions = [
  { value: '', label: '不限' },
  { value: 'mobile', label: '仅移动端' },
  { value: 'tablet', label: '仅平板' },
  { value: 'pc', label: '仅 PC' },
  { value: 'mobile,tablet', label: '移动端 + 平板' }
];

export const spiderOptions = [
  { value: 'googlebot', label: 'Googlebot' },
  { value: 'bingbot', label: 'Bingbot' },
  { value: 'facebookexternalhit', label: 'Facebook External Hit' },
  { value: 'tiktokbot', label: 'TikTok Bot' }
];

export const intermediateOptions = [
  { value: 'none', label: '不使用' },
  { value: 'system_template', label: '系统模版' },
  { value: 'custom_template', label: '自定义模版' }
];

export const labelOf = (options, value) => options.find((item) => item.value === value)?.label || value || '-';
export const typeOf = (options, value) => options.find((item) => item.value === value)?.type || 'info';
