import request from '@/utils/request';

/**
 * 分页查询商户域名
 */
export async function pageMerchantDomains(params) {
  const res = await request.get('/merchant/domain/list', { params });
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 获取域名 DNS 配置
 */
export async function getMerchantDomainDnsConfig() {
  const res = await request.get('/merchant/domain/dns-config');
  if (res.data.code === 200) {
    return res.data.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 新增商户域名
 */
export async function addMerchantDomain(data) {
  const res = await request.post('/merchant/domain', data);
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 校验商户域名
 */
export async function verifyMerchantDomain(domainId) {
  const res = await request.post(`/merchant/domain/${domainId}/verify`);
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}
