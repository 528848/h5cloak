import request from '@/utils/request';

const ok = (res) =>
  res.data.code === 200 ? res.data : Promise.reject(new Error(res.data.msg));

/** 分页查询商户域名（可按 siteId 过滤） */
export async function pageMerchantDomains(params) {
  return ok(await request.get('/merchant/domain/list', { params }));
}

/** 为站点添加域名 */
export async function addMerchantDomain(siteId, data) {
  return ok(await request.post(`/merchant/site/${siteId}/domain`, data));
}

/** 校验域名解析，返回最新域名行 */
export async function verifyMerchantDomain(domainId) {
  return ok(await request.post(`/merchant/domain/${domainId}/verify`));
}

/** 删除域名 */
export async function removeMerchantDomain(domainId) {
  return (await ok(await request.delete(`/merchant/domain/${domainId}`))).msg;
}
