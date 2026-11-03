import request from '@/utils/request';

const ok = (res) =>
  res.data.code === 200 ? res.data : Promise.reject(new Error(res.data.msg));

/** 全平台站点分页 */
export async function pageAdminSites(params) {
  return ok(await request.get('/admin/site/list', { params }));
}

/** 全平台站点/域名统计 */
export async function getAdminSiteStats() {
  return (await ok(await request.get('/admin/site/stats'))).data;
}

/** 平台侧强制启停站点 */
export async function updateAdminSiteStatus(siteId, status) {
  return (
    await ok(await request.put('/admin/site/changeStatus', { siteId, status }))
  ).msg;
}
