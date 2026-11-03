export const userStatusOptions = [
  { value: 'enabled', label: '启用', type: 'success' },
  { value: 'disabled', label: '停用', type: 'info' }
];

export const roleOptions = [
  {
    value: 'merchant_admin',
    label: '商户管理员',
    type: 'danger',
    desc: '管理全部资源与账号'
  },
  {
    value: 'merchant_operator',
    label: '操作员',
    type: 'primary',
    desc: '操作站点、归因规则、分销码，查看日志'
  },
  {
    value: 'merchant_viewer',
    label: '只读',
    type: 'info',
    desc: '仅查看看板、日志、报表'
  }
];

export const getOptionLabel = (options, value) =>
  options.find((o) => o.value === value)?.label ?? value ?? '-';

export const getOptionType = (options, value) =>
  options.find((o) => o.value === value)?.type ?? 'info';
