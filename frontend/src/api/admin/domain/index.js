import request from '@/utils/request';

const ok = (res) =>
  res.data.code === 200 ? res.data : Promise.reject(new Error(res.data.msg));

/** 全平台域名分页 */
export async function pageAdminDomains(params) {
  return ok(await request.get('/admin/domain/list', { params }));
}

/** 平台侧校验域名解析 */
export async function verifyAdminDomain(domainId) {
  return ok(await request.post(`/admin/domain/${domainId}/verify`));
}
