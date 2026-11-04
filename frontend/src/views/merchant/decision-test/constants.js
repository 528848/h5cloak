export const contentOptions = [
  { value: 'real', label: '真实内容', type: 'success' },
  { value: 'review', label: '审核内容', type: 'warning' },
  { value: 'block', label: '拒绝访问', type: 'danger' },
  { value: 'not_found', label: '404', type: 'info' },
  { value: 'bypass', label: '不拦截', type: 'info' },
  { value: 'maintenance', label: '维护页', type: 'info' }
];

export const touchOptions = [
  { value: 'first', label: '首访', type: 'success' },
  { value: 'return', label: '回访', type: 'primary' },
  { value: 'handoff', label: '交接', type: 'warning' },
  { value: 'none', label: '无归因', type: 'info' }
];

export const confidenceOptions = [
  { value: 'high', label: '高', type: 'success' },
  { value: 'medium', label: '中', type: 'primary' },
  { value: 'low', label: '低', type: 'info' }
];

export const errorReasons = {
  not_registered: '未登记',
  disabled: '已停用',
  expired: '已过期',
  not_started: '未生效',
  path_not_allowed: '路径不允许',
  format_invalid: '格式不符',
  source_not_allowed: '来源不在允许列表',
  param_disabled: '参数已停用',
  click_reused: '点击参数被转发复用'
};

export const traceSteps = {
  site: '站点状态',
  path_rule: '路径规则',
  token: '访客令牌',
  params: '参数提取',
  confidence: '置信度',
  decision: '判定',
  handoff_token: '交接令牌',
  handoff_match: '交接匹配',
  token_issue: '签发令牌'
};

export const uaPresets = [
  {
    label: '桌面 Chrome',
    value:
      'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36'
  },
  {
    label: 'iPhone Safari',
    value:
      'Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1'
  },
  {
    label: 'iPhone Facebook 内置浏览器',
    value:
      'Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 [FBAN/FBIOS;FBAV/430.0.0;FBDV/iPhone15,2]'
  },
  {
    label: 'iPhone Instagram 内置浏览器',
    value:
      'Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Instagram 300.0.0.0'
  },
  {
    label: 'Android TikTok 内置浏览器',
    value:
      'Mozilla/5.0 (Linux; Android 14; SM-S918B Build/UP1A; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/120.0 Mobile Safari/537.36 musical_ly_2023'
  }
];

export const getOptionLabel = (options, value) =>
  options.find((o) => o.value === value)?.label ?? value ?? '-';

export const getOptionType = (options, value) =>
  options.find((o) => o.value === value)?.type ?? 'info';
