import request from '@/utils/request';

const ok = (res) =>
  res.data.code === 200 ? res.data : Promise.reject(new Error(res.data.msg));

/** 站点生效的参数字典（内置 + 站点覆盖） */
export async function listAttributionParams(siteId) {
  return (
    await ok(
      await request.get('/merchant/attribution/params', { params: { siteId } })
    )
  ).data;
}

/** 新增或覆盖站点参数 */
export async function saveAttributionParam(data) {
  return ok(await request.post('/merchant/attribution/params', data));
}

/** 删除站点参数（覆盖行删除后恢复内置） */
export async function removeAttributionParam(siteId, paramId) {
  return (
    await ok(
      await request.delete(`/merchant/attribution/params/${paramId}`, {
        params: { siteId }
      })
    )
  ).msg;
}

/** 站点校验规则 */
export async function getAttributionRule(siteId) {
  return (
    await ok(
      await request.get('/merchant/attribution/rule', { params: { siteId } })
    )
  ).data;
}

/** 保存站点校验规则 */
export async function saveAttributionRule(data) {
  return (await ok(await request.put('/merchant/attribution/rule', data))).msg;
}
