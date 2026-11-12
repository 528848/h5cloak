import request from '@/utils/request';

const ok = (res) =>
  res.data.code === 200 ? res.data : Promise.reject(new Error(res.data.msg));

/** 可分配角色（启用中） */
export async function listMerchantRoles() {
  return (await ok(await request.get('/merchant/role/list'))).data;
}

/** 角色管理列表，含内置角色和本商户自建角色 */
export async function listManageRoles() {
  return (await ok(await request.get('/merchant/role/manage'))).data;
}

export async function saveMerchantRole(id, data) {
  const req = id
    ? request.put(`/merchant/role/${id}`, data)
    : request.post('/merchant/role', data);
  return ok(await req);
}

export async function removeMerchantRole(id) {
  return (await ok(await request.delete(`/merchant/role/${id}`))).msg;
}

/** 商户端菜单，只读，用于角色勾选 */
export async function listMerchantMenus() {
  return (await ok(await request.get('/merchant/menu/list'))).data;
}

/** 分页查询子账号 */
export async function pageMerchantUsers(params) {
  return ok(await request.get('/merchant/user/list', { params }));
}

/** 新增子账号 */
export async function addMerchantUser(data) {
  return ok(await request.post('/merchant/user', data));
}

/** 编辑子账号 */
export async function updateMerchantUser(userId, data) {
  return (await ok(await request.put(`/merchant/user/${userId}`, data))).msg;
}

/** 启停子账号 */
export async function updateMerchantUserStatus(userId, status) {
  return (
    await ok(
      await request.put('/merchant/user/changeStatus', { userId, status })
    )
  ).msg;
}

/** 重置子账号密码 */
export async function resetMerchantUserPassword(userId, password) {
  return (
    await ok(
      await request.put(`/merchant/user/${userId}/resetPwd`, { password })
    )
  ).msg;
}

/** 删除子账号 */
export async function removeMerchantUser(userId) {
  return (await ok(await request.delete(`/merchant/user/${userId}`))).msg;
}
