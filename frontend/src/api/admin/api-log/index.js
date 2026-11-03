import request from '@/utils/request';

/**
 * 分页查询全平台 API 调用日志
 */
export async function pageAdminApiLogs(params) {
  const res = await request.get('/admin/api-log/list', { params });
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}
