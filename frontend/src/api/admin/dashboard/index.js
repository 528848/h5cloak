import request from '@/utils/request';

/**
 * 查询总后台数据看板
 */
export async function getAdminDashboard() {
  const res = await request.get('/admin/dashboard');
  if (res.data.code === 200) {
    return res.data.data;
  }
  return Promise.reject(new Error(res.data.msg));
}
