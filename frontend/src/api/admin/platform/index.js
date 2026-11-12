import request from '@/utils/request';

const ok = (res) =>
  res.data.code === 200 ? res.data : Promise.reject(new Error(res.data.msg));

export async function getSettings() {
  return (await ok(await request.get('/admin/system/settings'))).data;
}

export async function saveSettings(data) {
  return (await ok(await request.put('/admin/system/settings', data))).msg;
}

export async function listNodes() {
  return (await ok(await request.get('/admin/system/nodes'))).data;
}

export async function listTemplates(params) {
  return ok(await request.get('/admin/template/list', { params }));
}

export async function getTemplateStats() {
  return (await ok(await request.get('/admin/template/stats'))).data;
}

export async function getTemplate(id) {
  return (await ok(await request.get(`/admin/template/${id}`))).data;
}

export async function saveTemplate(id, data) {
  const req = id
    ? request.put(`/admin/template/${id}`, data)
    : request.post('/admin/template', data);
  const body = await ok(await req);
  return body;
}

export async function removeTemplate(id) {
  return (await ok(await request.delete(`/admin/template/${id}`))).msg;
}

export async function updateTemplateStatus(id, status) {
  return (await ok(await request.put(`/admin/template/${id}/status`, { status }))).msg;
}

export async function uploadTemplatePackage(id, file, entry, spaFallback) {
  const data = new FormData();
  data.append('file', file);
  if (entry) data.append('entry', entry);
  data.append('spaFallback', spaFallback ? 'true' : 'false');
  return (await ok(await request.post(`/admin/template/${id}/package`, data))).data;
}

export async function rollbackTemplate(id, version) {
  return (await ok(await request.post(`/admin/template/${id}/rollback`, { version }))).msg;
}

export async function previewTemplate(id, data) {
  return (await ok(await request.post(`/admin/template/${id}/preview`, data || {}))).data;
}

export async function pagePlatformCodes(params) {
  return ok(await request.get('/admin/code/list', { params }));
}

export async function updatePlatformCodeStatus(ids, status) {
  return (
    await ok(await request.put('/admin/code/changeStatus', { ids, status }))
  ).msg;
}

export async function pageAccounts(params) {
  return ok(await request.get('/admin/account/list', { params }));
}

export async function saveAccount(id, data) {
  const req = id
    ? request.put(`/admin/account/${id}`, data)
    : request.post('/admin/account', data);
  return ok(await req);
}

export async function updateAccountStatus(userId, status) {
  return (
    await ok(
      await request.put('/admin/account/changeStatus', { userId, status })
    )
  ).msg;
}

export async function resetAccountPassword(id, password) {
  return (
    await ok(await request.put(`/admin/account/${id}/resetPwd`, { password }))
  ).msg;
}

export async function removeAccount(id) {
  return (await ok(await request.delete(`/admin/account/${id}`))).msg;
}

export async function listRoles() {
  return (await ok(await request.get('/admin/role/list'))).data;
}

export async function saveRole(id, data) {
  const req = id
    ? request.put(`/admin/role/${id}`, data)
    : request.post('/admin/role', data);
  return ok(await req);
}

export async function removeRole(id) {
  return (await ok(await request.delete(`/admin/role/${id}`))).msg;
}

export async function listMenus() {
  return (await ok(await request.get('/admin/menu/list'))).data;
}

export async function saveMenu(id, data) {
  return (await ok(await request.put(`/admin/menu/${id}`, data))).msg;
}

export async function impersonateMerchant(id) {
  return (await ok(await request.post(`/admin/merchant/${id}/impersonate`)))
    .data;
}
