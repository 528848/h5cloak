import request from '@/utils/request';

/**
 * 分页查询黑白名单
 */
export async function pageAccessLists(params) {
  const res = await request.get('/merchant/access-list/list', { params });
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 新增黑白名单
 */
export async function addAccessList(data) {
  const res = await request.post('/merchant/access-list', data);
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 修改黑白名单
 */
export async function updateAccessList(listId, data) {
  const res = await request.put(`/merchant/access-list/${listId}`, data);
  if (res.data.code === 200) {
    return res.data;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 删除黑白名单
 */
export async function removeAccessList(listId) {
  const res = await request.delete(`/merchant/access-list/${listId}`);
  if (res.data.code === 200) {
    return res.data.msg;
  }
  return Promise.reject(new Error(res.data.msg));
}

/**
 * 修改黑白名单状态
 */
export async function updateAccessListStatus(listId, status) {
  const res = await request.put('/merchant/access-list/changeStatus', {
    listId,
    status
  });
  if (res.data.code === 200) {
    return res.data.msg;
  }
  return Promise.reject(new Error(res.data.msg));
}
