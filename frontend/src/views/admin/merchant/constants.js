/** 商户状态 */
export const merchantStatusOptions = [
  { value: 'enabled', label: '启用', type: 'success' },
  { value: 'disabled', label: '停用', type: 'info' },
  { value: 'frozen', label: '冻结', type: 'danger' },
  { value: 'expired', label: '已到期', type: 'warning' }
];

/** 额度策略 */
export const quotaPolicyOptions = [
  { value: 'alert', label: '仅告警', type: 'info' },
  { value: 'review_all', label: '全部返回审核内容', type: 'warning' },
  { value: 'deny', label: '拒绝服务', type: 'danger' }
];

/** 功能开关 */
export const featureOptions = [
  { key: 'hosted', label: '托管接入', desc: '域名解析到网关，由网关反向代理' },
  {
    key: 'server_api',
    label: '服务端接入',
    desc: '判定接口 v2、插件、V1 兼容接口'
  },
  { key: 'js_sdk', label: 'JS SDK', desc: '纯静态页辅助方案' },
  { key: 'conversion', label: '转化回传', desc: '订单回传与访客合并' },
  { key: 'webhook', label: 'Webhook', desc: '告警与事件推送' },
  { key: 'enhancements', label: '增强层', desc: '名单、审核机构识别、风险信号' }
];

export const getOptionLabel = (options, value) =>
  options.find((o) => o.value === value)?.label ?? value ?? '-';

export const getOptionType = (options, value) =>
  options.find((o) => o.value === value)?.type ?? 'info';
