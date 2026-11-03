import request from '@/utils/request';
import { download, checkDownloadRes } from '@/utils/common';

const ok = (res) =>
  res.data.code === 200 ? res.data : Promise.reject(new Error(res.data.msg));

/** 分页查询分销码 */
export async function pageCodes(params) {
  return ok(await request.get('/merchant/code/list', { params }));
}

/** 统计 */
export async function getCodeStats(siteId) {
  return (
    await ok(await request.get('/merchant/code/stats', { params: { siteId } }))
  ).data;
}

/** 渠道标签 */
export async function listCodeChannels() {
  return (await ok(await request.get('/merchant/code/channels'))).data;
}

/** 登记单个码 */
export async function addCode(data) {
  return ok(await request.post('/merchant/code', data));
}

/** 编辑码 */
export async function updateCode(codeId, data) {
  return (await ok(await request.put(`/merchant/code/${codeId}`, data))).msg;
}

/** 批量启停 */
export async function updateCodesStatus(codeIds, status) {
  return (
    await ok(
      await request.put('/merchant/code/changeStatus', { codeIds, status })
    )
  ).msg;
}

/** 批量删除 */
export async function removeCodes(codeIds) {
  return (await ok(await request.post('/merchant/code/remove', { codeIds })))
    .msg;
}

/** 批量生成 */
export async function generateCodes(data) {
  return ok(await request.post('/merchant/code/generate', data));
}

/** 文本导入 */
export async function importCodes(data) {
  return ok(await request.post('/merchant/code/import', data));
}

/** 导出 CSV */
export async function exportCodes(params) {
  const res = await request.get('/merchant/code/export', {
    params,
    responseType: 'blob'
  });
  await checkDownloadRes(res);
  download(res.data, `promo-codes-${Date.now()}.csv`);
}
