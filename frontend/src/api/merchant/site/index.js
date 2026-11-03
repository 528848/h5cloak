import request from '@/utils/request';

const ok = (res) =>
  res.data.code === 200 ? res.data : Promise.reject(new Error(res.data.msg));

/** 分页查询站点 */
export async function pageSites(params) {
  return ok(await request.get('/merchant/site/list', { params }));
}

/** 站点/域名统计 */
export async function getSiteStats() {
  return (await ok(await request.get('/merchant/site/stats'))).data;
}

/** 站点下拉选项 */
export async function listSiteOptions() {
  return (await ok(await request.get('/merchant/site/options'))).data;
}

/** 接入指引（网关入口域名等） */
export async function getSiteGuide() {
  return (await ok(await request.get('/merchant/site/guide'))).data;
}

/** 站点详情（含源站、审核内容、路径规则） */
export async function getSite(siteId) {
  return (await ok(await request.get(`/merchant/site/${siteId}`))).data;
}

/** 新建站点，返回一次性服务端密钥 */
export async function addSite(data) {
  return ok(await request.post('/merchant/site', data));
}

/** 更新站点设置 */
export async function updateSite(siteId, data) {
  return (await ok(await request.put(`/merchant/site/${siteId}`, data))).msg;
}

/** 启停站点 */
export async function updateSiteStatus(siteId, status) {
  return (
    await ok(
      await request.put('/merchant/site/changeStatus', { siteId, status })
    )
  ).msg;
}

/** 删除站点 */
export async function removeSite(siteId) {
  return (await ok(await request.delete(`/merchant/site/${siteId}`))).msg;
}

/** 重置服务端密钥 */
export async function resetSiteSecret(siteId, secretIpAllowlist) {
  return ok(
    await request.post(`/merchant/site/${siteId}/secret`, { secretIpAllowlist })
  );
}

/** 保存源站 */
export async function saveSiteOrigin(siteId, data) {
  return (await ok(await request.put(`/merchant/site/${siteId}/origin`, data)))
    .msg;
}

/** 检测源站 */
export async function probeSiteOrigin(siteId, kind) {
  return (
    await ok(
      await request.post(`/merchant/site/${siteId}/origin/probe`, null, {
        params: { kind }
      })
    )
  ).data;
}

/** 保存审核内容 */
export async function saveSiteReview(siteId, data) {
  return (await ok(await request.put(`/merchant/site/${siteId}/review`, data)))
    .msg;
}

/** 审核页模板列表 */
export async function listReviewTemplates() {
  return (await ok(await request.get('/merchant/site/templates'))).data;
}

/** 预览模板（返回 HTML 字符串） */
export async function previewReviewTemplate(templateId, vars) {
  const res = await request.post(
    `/merchant/site/templates/${templateId}/preview`,
    vars || {},
    { responseType: 'text' }
  );
  return typeof res.data === 'string' ? res.data : JSON.stringify(res.data);
}

/** 上传静态包 */
export async function uploadSitePackage(siteId, file) {
  const fd = new FormData();
  fd.append('file', file);
  return ok(await request.post(`/merchant/site/${siteId}/package`, fd));
}

/** 保存路径规则（整表替换） */
export async function saveSitePathRules(siteId, rules) {
  return (
    await ok(await request.put(`/merchant/site/${siteId}/path-rules`, rules))
  ).msg;
}
