import request from '@/utils/request';

const ok = (res) =>
  res.data.code === 200 ? res.data : Promise.reject(new Error(res.data.msg));

/** 判定测试（dry run，不写日志） */
export async function runDecisionTest(data) {
  return (await ok(await request.post('/merchant/decision/test', data))).data;
}

/** 站点快照状态 */
export async function getSnapshotStatus(siteId) {
  return (
    await ok(
      await request.get('/merchant/snapshot/status', { params: { siteId } })
    )
  ).data;
}

/** 手动发布站点快照 */
export async function publishSnapshot(siteId) {
  return (await ok(await request.post(`/merchant/snapshot/${siteId}/publish`)))
    .msg;
}

/** 轮换访客令牌密钥 */
export async function rotateSiteKey(siteId, graceHours) {
  return ok(
    await request.post(`/merchant/site/${siteId}/rotate-key`, { graceHours })
  );
}
