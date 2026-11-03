export const paramTypeOptions = [
  {
    value: 'click',
    label: '点击参数',
    type: 'primary',
    desc: '广告平台的点击 ID'
  },
  {
    value: 'source',
    label: '来源参数',
    type: 'info',
    desc: 'utm_* 之类的来源标识'
  },
  {
    value: 'distributor',
    label: '分销码',
    type: 'success',
    desc: '须在分销码列表中登记'
  },
  {
    value: 'campaign',
    label: '活动码',
    type: 'warning',
    desc: '须在活动码列表中登记'
  },
  { value: 'free', label: '自由参数', type: 'info', desc: '只校验格式' }
];

export const paramSourceOptions = [
  { value: 'builtin', label: '内置', type: 'info' },
  { value: 'override', label: '已覆盖', type: 'warning' },
  { value: 'site', label: '自定义', type: 'success' }
];

export const paramStatusOptions = [
  { value: 'enabled', label: '启用', type: 'success' },
  { value: 'disabled', label: '停用', type: 'info' }
];

export const combineOptions = [
  {
    value: 'require_any',
    label: '任一归因参数即可',
    desc: '任意一个「单独足以放行」的参数通过校验即放行'
  },
  {
    value: 'require_click_or_code',
    label: '至少一个点击参数或分销码',
    desc: '只有来源参数时不放行'
  }
];

export const reuseActionOptions = [
  { value: 'downgrade', label: '降低置信度' },
  { value: 'review', label: '按无归因处理' }
];

export const revokeModeOptions = [
  {
    value: 'window',
    label: '保留到归因窗口到期',
    desc: '码停用后，已经放行的访客在窗口内仍可访问'
  },
  {
    value: 'immediate',
    label: '立即失效',
    desc: '码停用后，带该码归因的访客立即回到审核内容'
  }
];

export const getOptionLabel = (options, value) =>
  options.find((o) => o.value === value)?.label ?? value ?? '-';

export const getOptionType = (options, value) =>
  options.find((o) => o.value === value)?.type ?? 'info';

export const emptyParamForm = () => ({
  name: '',
  paramType: 'click',
  platform: '',
  aliasesText: '',
  minLen: 1,
  maxLen: 512,
  pattern: '',
  sufficient: true,
  status: 'enabled',
  sort: 0,
  remark: ''
});

export const paramToPayload = (form) => ({
  name: form.name,
  paramType: form.paramType,
  platform: form.platform,
  aliases: String(form.aliasesText || '')
    .split(/[\s,]+/)
    .map((s) => s.trim())
    .filter(Boolean),
  minLen: Number(form.minLen) || 1,
  maxLen: Number(form.maxLen) || 512,
  pattern: form.pattern,
  sufficient: !!form.sufficient,
  status: form.status,
  sort: Number(form.sort) || 0,
  remark: form.remark
});
