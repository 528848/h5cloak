import request from '@/utils/request';

/**
 * 分页查询 API Key
 */
export async function pageApiKeys(params) {
  const res = await request.get('/merchant/api-key/list', { params });
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 新增 API Key
 */
export async function addApiKey(data) {
  const res = await request.post('/merchant/api-key', data);
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 重置 API Key
 */
export async function resetApiKey(apiKeyId) {
  const res = await request.post(`/merchant/api-key/${apiKeyId}/reset`);
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 删除 API Key
 */
export async function removeApiKey(apiKeyId) {
  const res = await request.delete(`/merchant/api-key/${apiKeyId}`);
  if (res.data.code === 200) {
    return res.data.msg;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 修改 API Key 状态
 */
export async function updateApiKeyStatus(apiKeyId, status) {
  const res = await request.put('/merchant/api-key/changeStatus', {
    apiKeyId,
    status
  });
  if (res.data.code === 200) {
    return res.data.msg;
  }
  return Promise.reject(new Error(res.data.msg));
}
