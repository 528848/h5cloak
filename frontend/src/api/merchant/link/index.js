import request from '@/utils/request';

const ok = (res) =>
  res.data.code === 200 ? res.data : Promise.reject(new Error(res.data.msg));

/** 站点域名与参数名 */
export async function getLinkOptions(siteId) {
  return (
    await ok(
      await request.get('/merchant/link/options', { params: { siteId } })
    )
  ).data;
}

/** 生成推广链接 */
export async function buildLink(data) {
  return (await ok(await request.post('/merchant/link/build', data))).data;
}
