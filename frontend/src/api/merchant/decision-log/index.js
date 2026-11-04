import request from '@/utils/request';
import { APP_PORTAL } from '@portal';

const ok = (res) =>
  res.data.code === 200 ? res.data : Promise.reject(new Error(res.data.msg));

const prefix = `/${APP_PORTAL}/decision-log`;

/** 分页查询决策日志（商户端/总后台共用，按门户选路径） */
export async function pageDecisionLogs(params) {
  return ok(await request.get(`${prefix}/list`, { params }));
}

/** 最近 24 小时统计 */
export async function getDecisionLogStats(siteId) {
  return (
    await ok(await request.get(`${prefix}/stats`, { params: { siteId } }))
  ).data;
}

/** 单条链路详情 */
export async function getDecisionLog(requestId) {
  return (await ok(await request.get(`${prefix}/${requestId}`))).data;
}
