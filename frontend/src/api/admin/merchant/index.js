import request from '@/utils/request';

const ok = (res) =>
  res.data.code === 200 ? res.data : Promise.reject(new Error(res.data.msg));

/** 分页查询商户 */
export async function pageMerchants(params) {
  return ok(await request.get('/admin/merchant/list', { params }));
}

/** 概览统计 */
export async function getMerchantStats() {
  return (await ok(await request.get('/admin/merchant/stats'))).data;
}

/** 商户详情 */
export async function getMerchant(merchantId) {
  return (await ok(await request.get(`/admin/merchant/${merchantId}`))).data;
}

/** 开通商户 */
export async function addMerchant(data) {
  return ok(await request.post('/admin/merchant', data));
}

/** 编辑商户 */
export async function updateMerchant(merchantId, data) {
  return (await ok(await request.put(`/admin/merchant/${merchantId}`, data)))
    .msg;
}

/** 修改状态：enabled / disabled / frozen */
export async function updateMerchantStatus(merchantId, status) {
  return (
    await ok(
      await request.put('/admin/merchant/changeStatus', { merchantId, status })
    )
  ).msg;
}

/** 额度与功能开关 */
export async function updateMerchantQuota(merchantId, quota, features) {
  return (
    await ok(
      await request.put(`/admin/merchant/${merchantId}/quota`, {
        quota,
        features
      })
    )
  ).msg;
}

/** 重置商户管理员密码 */
export async function resetMerchantAdminPassword(merchantId, password) {
  return ok(
    await request.put(`/admin/merchant/${merchantId}/resetPwd`, { password })
  );
}
