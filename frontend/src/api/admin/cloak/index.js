import request from '@/utils/request';

/**
 * 分页查询全平台斗篷
 */
export async function pageAdminCloaks(params) {
  const res = await request.get('/admin/cloak/list', { params });
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 平台侧修改斗篷状态
 */
export async function updateAdminCloakStatus(cloakId, status) {
  const res = await request.put('/admin/cloak/changeStatus', {
    cloakId,
    status
  });
  if (res.data.code === 200) {
    return res.data.msg;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 分页查询全平台斗篷日志
 */
export async function pageAdminCloakLogs(params) {
  const res = await request.get('/admin/cloak-log/list', { params });
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}
