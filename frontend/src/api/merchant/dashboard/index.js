import request from '@/utils/request';

/**
 * 查询商户数据看板
 */
export async function getMerchantDashboard() {
  const res = await request.get('/merchant/dashboard');
  if (res.data.code === 200) {
    return res.data.data;
  }
  return Promise.reject(new Error(res.data.msg));
}
