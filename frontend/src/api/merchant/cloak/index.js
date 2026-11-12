import request from '@/utils/request';

const ok = (res) => (res.data.code === 200 ? res.data : Promise.reject(new Error(res.data.msg)));

export async function getCloakStats() {
  return (await ok(await request.get('/merchant/cloak/stats'))).data;
}

export async function pageCloaks(params) {
  return ok(await request.get('/merchant/cloak/list', { params }));
}

export async function getCloak(id) {
  return (await ok(await request.get(`/merchant/cloak/${id}`))).data;
}

export async function listCloakDomains(params) {
  return (await ok(await request.get('/merchant/cloak/domain-options', { params }))).data;
}

export async function addCloak(data) {
  return (await ok(await request.post('/merchant/cloak', data))).data;
}

export async function updateCloak(id, data) {
  return (await ok(await request.put(`/merchant/cloak/${id}`, data))).msg;
}

export async function copyCloak(id) {
  return (await ok(await request.post(`/merchant/cloak/${id}/copy`))).msg;
}

export async function updateCloakStatus(id, status) {
  return (await ok(await request.put('/merchant/cloak/changeStatus', { id, status }))).msg;
}

export async function removeCloak(id) {
  return (await ok(await request.delete(`/merchant/cloak/${id}`))).msg;
}
