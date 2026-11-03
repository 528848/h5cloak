import request from '@/utils/request';

/**
 * 分页查询全平台域名
 */
export async function pageAdminDomains(params) {
  const res = await request.get('/admin/domain/list', { params });
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 平台侧修改域名状态
 */
export async function updateAdminDomainStatus(domainId, status) {
  const res = await request.put('/admin/domain/changeStatus', {
    domainId,
    status
  });
  if (res.data.code === 200) {
    return res.data.msg;
  }
  return Promise.reject(new Error(res.data.msg));
}
