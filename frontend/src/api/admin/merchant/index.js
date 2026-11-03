import request from '@/utils/request';

/**
 * 分页查询商户账号
 */
export async function pageMerchants(params) {
  const res = await request.get('/admin/merchant/list', { params });
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 新增商户账号
 */
export async function addMerchant(data) {
  const res = await request.post('/admin/merchant', data);
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 修改商户状态
 */
export async function updateMerchantStatus(merchantId, status) {
  const res = await request.put('/admin/merchant/changeStatus', {
    merchantId,
    status
  });
  if (res.data.code === 200) {
    return res.data.msg;
  }
  return Promise.reject(new Error(res.data.msg));
}
