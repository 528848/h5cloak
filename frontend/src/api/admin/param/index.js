import request from '@/utils/request';

const ok = (res) =>
  res.data.code === 200 ? res.data : Promise.reject(new Error(res.data.msg));

/** 平台内置参数 */
export async function listBuiltinParams() {
  return (await ok(await request.get('/admin/param/list'))).data;
}

/** 新增内置参数 */
export async function addBuiltinParam(data) {
  return ok(await request.post('/admin/param', data));
}

/** 修改内置参数 */
export async function updateBuiltinParam(paramId, data) {
  return (await ok(await request.put(`/admin/param/${paramId}`, data))).msg;
}

/** 删除内置参数 */
export async function removeBuiltinParam(paramId) {
  return (await ok(await request.delete(`/admin/param/${paramId}`))).msg;
}
