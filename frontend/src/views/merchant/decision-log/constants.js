export {
  contentOptions,
  touchOptions,
  confidenceOptions,
  errorReasons,
  traceSteps,
  getOptionLabel,
  getOptionType
} from '@/views/merchant/decision-test/constants';

export const reasonOptions = [
  { value: 'not_registered', label: '未登记' },
  { value: 'disabled', label: '已停用' },
  { value: 'expired', label: '已过期' },
  { value: 'not_started', label: '未生效' },
  { value: 'path_not_allowed', label: '路径不允许' },
  { value: 'format_invalid', label: '格式不符' },
  { value: 'source_not_allowed', label: '来源不在允许列表' },
  { value: 'param_disabled', label: '参数已停用' },
  { value: 'click_reused', label: '点击参数被转发复用' }
];

export const inAppOptions = [
  { value: 'facebook', label: 'Facebook' },
  { value: 'instagram', label: 'Instagram' },
  { value: 'tiktok', label: 'TikTok' },
  { value: 'snapchat', label: 'Snapchat' },
  { value: 'twitter', label: 'X / Twitter' },
  { value: 'linkedin', label: 'LinkedIn' },
  { value: 'line', label: 'Line' },
  { value: 'wechat', label: '微信' },
  { value: 'kwai', label: 'Kwai' },
  { value: 'webview', label: '其它 WebView' }
];

export const originKindLabels = {
  real: '真实源站',
  review: '审核源站',
  template: '平台模板',
  static: '静态包',
  block: '拒绝页',
  not_found: '404 页',
  bypass: '直通源站',
  maintenance: '维护页',
  debug: '调试输出'
};
