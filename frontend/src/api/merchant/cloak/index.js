import request from '@/utils/request';

/**
 * 分页查询商户斗篷
 */
export async function pageMerchantCloaks(params) {
  const res = await request.get('/merchant/cloak/list', { params });
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 新增斗篷
 */
export async function addMerchantCloak(data) {
  const res = await request.post('/merchant/cloak', data);
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 修改斗篷
 */
export async function updateMerchantCloak(cloakId, data) {
  const res = await request.put(`/merchant/cloak/${cloakId}`, data);
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 复制斗篷
 */
export async function copyMerchantCloak(cloakId) {
  const res = await request.post(`/merchant/cloak/${cloakId}/copy`);
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 删除斗篷
 */
export async function removeMerchantCloak(cloakId) {
  const res = await request.delete(`/merchant/cloak/${cloakId}`);
  if (res.data.code === 200) {
    return res.data.msg;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 修改斗篷状态
 */
export async function updateMerchantCloakStatus(cloakId, status) {
  const res = await request.put('/merchant/cloak/changeStatus', {
    cloakId,
    status
  });
  if (res.data.code === 200) {
    return res.data.msg;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 分页查询商户斗篷日志
 */
export async function pageMerchantCloakLogs(params) {
  const res = await request.get('/merchant/cloak-log/list', { params });
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}
