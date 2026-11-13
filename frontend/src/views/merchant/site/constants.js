export const siteStatusOptions = [
  { value: 'enabled', label: '启用', type: 'success' },
  { value: 'disabled', label: '停用', type: 'info' }
];

export const siteTypeOptions = [
  {
    value: 'hybrid',
    label: '混合接入',
    type: 'primary',
    desc: '托管入口与服务端密钥都可以用'
  },
  {
    value: 'hosted',
    label: '托管接入',
    type: 'success',
    desc: '访客从已绑定域名进入，由网关回源'
  },
  {
    value: 'server',
    label: '服务端接入',
    type: 'warning',
    desc: '由商户服务端携带密钥请求判定'
  }
];

export const domainStatusOptions = [
  { value: 'pending', label: '待校验', type: 'warning' },
  { value: 'bound', label: '已绑定', type: 'success' },
  { value: 'abnormal', label: '解析异常', type: 'danger' },
  { value: 'disabled', label: '已停用', type: 'info' }
];

export const domainTypeOptions = [
  { value: 'hosted', label: '托管域名', type: 'primary' },
  { value: 'own', label: '自有域名', type: 'info' }
];

export const healthOptions = [
  { value: 'unknown', label: '未检测', type: 'info' },
  { value: 'healthy', label: '可达', type: 'success' },
  { value: 'unhealthy', label: '不可达', type: 'danger' }
];

export const renewModeOptions = [
  { value: 'sliding', label: '滑动续期' },
  { value: 'fixed', label: '固定到期' }
];

export const noAttributionOptions = [
  { value: 'review', label: '返回审核内容' },
  { value: 'block', label: '拒绝访问' },
  { value: 'not_found', label: '返回 404' }
];

export const failModeOptions = [
  { value: 'review', label: '返回审核内容' },
  { value: 'real', label: '返回真实内容' },
  { value: 'maintenance', label: '返回维护页' }
];

export const confidenceOptions = [
  { value: 'low', label: '低' },
  { value: 'medium', label: '中' },
  { value: 'high', label: '高' }
];

export const reviewSourceOptions = [
  { value: 'template', label: '平台模板' },
  { value: 'origin', label: '审核源站' },
  { value: 'static', label: '静态包' }
];

export const matchTypeOptions = [
  { value: 'prefix', label: '前缀' },
  { value: 'exact', label: '精确' },
  { value: 'suffix', label: '后缀' }
];

export const pathActionOptions = [
  { value: 'bypass', label: '不拦截' },
  { value: 'always_real', label: '始终真实' },
  { value: 'always_review', label: '始终审核' },
  { value: 'block', label: '拒绝' }
];

export const getOptionLabel = (options, value) =>
  options.find((o) => o.value === value)?.label ?? value ?? '-';

export const getOptionType = (options, value) =>
  options.find((o) => o.value === value)?.type ?? 'info';

export const handoffCopyDefaults = () => ({
  mobile: {
    title: '在浏览器中打开',
    body: '用系统浏览器打开，页面和登录会更完整。',
    open: '打开',
    stay: '继续浏览'
  },
  desktop: {
    title: '用系统浏览器打开',
    body: '当前是应用内置浏览器。换到系统浏览器后可以继续这次访问。',
    open: '打开',
    stay: '稍后'
  }
});

export const emptySiteForm = () => ({
  name: '',
  siteType: 'hybrid',
  remark: '',
  cookieName: '_ck_vid',
  cookieDomain: '',
  attributionTtlDays: 30,
  renewMode: 'sliding',
  noAttributionAction: 'review',
  failMode: 'review',
  minConfidence: 'medium',
  handoffBanner: true,
  handoffBannerText: '在浏览器中打开，继续访问',
  handoffDelaySeconds: 10,
  handoffCooldownMinutes: 30,
  handoffMaxShows: 3,
  handoffDismissEnds: false,
  handoffMobile: handoffCopyDefaults().mobile,
  handoffDesktop: handoffCopyDefaults().desktop,
  handoffLinkRewrite: true,
  handoffTokenTtlHours: 24,
  handoffTokenMaxUses: 3,
  handoffMatchEnabled: true,
  handoffMatchMinutes: 15,
  handoffMatchSysOnly: true,
  secretIpAllowlistText: ''
});

export const siteToPayload = (form) => ({
  name: form.name,
  siteType: form.siteType,
  remark: form.remark,
  cookieName: form.cookieName,
  cookieDomain: form.cookieDomain,
  attributionTtlDays: Number(form.attributionTtlDays) || 30,
  renewMode: form.renewMode,
  noAttributionAction: form.noAttributionAction,
  failMode: form.failMode,
  minConfidence: form.minConfidence,
  handoffBanner: !!form.handoffBanner,
  handoffBannerText: form.handoffMobile?.body || form.handoffBannerText,
  handoffPrompt: {
    delaySeconds: form.handoffDelaySeconds == null || form.handoffDelaySeconds === '' ? null : Number(form.handoffDelaySeconds),
    cooldownMinutes: form.handoffCooldownMinutes == null || form.handoffCooldownMinutes === '' ? null : Number(form.handoffCooldownMinutes),
    maxShows: form.handoffMaxShows == null || form.handoffMaxShows === '' ? null : Number(form.handoffMaxShows),
    dismissEnds: !!form.handoffDismissEnds,
    mobile: { ...form.handoffMobile },
    desktop: { ...form.handoffDesktop }
  },
  handoffLinkRewrite: !!form.handoffLinkRewrite,
  handoffTokenTtlHours: Number(form.handoffTokenTtlHours) || 24,
  handoffTokenMaxUses: Number(form.handoffTokenMaxUses) || 3,
  handoffMatchEnabled: !!form.handoffMatchEnabled,
  handoffMatchMinutes: Number(form.handoffMatchMinutes) || 15,
  handoffMatchSysOnly: !!form.handoffMatchSysOnly,
  secretIpAllowlist: String(form.secretIpAllowlistText || '')
    .split(/[\s,]+/)
    .map((s) => s.trim())
    .filter(Boolean)
});
