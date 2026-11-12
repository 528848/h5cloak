import request from '@/utils/request';

/**
 * 分页查询商户 API 调用日志
 */
export async function pageMerchantApiLogs(params) {
  const res = await request.get('/merchant/api-log/list', { params });
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/** 单条 API 调用详情 */
export async function getMerchantApiLog(id) {
  const res = await request.get(`/merchant/api-log/${id}`);
  if (res.data.code === 200) {
    return res.data.data;
  }
  return Promise.reject(new Error(res.data.msg));
}
