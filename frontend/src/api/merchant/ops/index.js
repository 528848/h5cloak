import request from '@/utils/request';
import { APP_PORTAL } from '@portal';

const ok = (res) =>
  res.data.code === 200 ? res.data : Promise.reject(new Error(res.data.msg));
const data = async (p) => (await ok(await p)).data;
const prefix = `/${APP_PORTAL}`;

/** 看板（商户端 / 总后台按门户选路径）。params: siteId, range, from, to */
export const getDashboard = (params) =>
  data(request.get(`${prefix}/dashboard`, { params }));

// ---- 报表 ----
export const getChannelReport = (params) =>
  data(request.get('/merchant/report/channel', { params }));
export const pageCodeReport = async (params) =>
  ok(await request.get('/merchant/report/code', { params }));
export const pagePathReport = async (params) =>
  ok(await request.get('/merchant/report/path', { params }));
export const pageConversionReport = async (params) =>
  ok(await request.get('/merchant/report/conversion', { params }));
export const getReconcile = (params) =>
  data(request.get('/merchant/report/reconcile', { params }));

// ---- 导出 ----
export const createExport = (body) =>
  data(request.post('/merchant/export', body));
export const pageExports = async (params) =>
  ok(await request.get('/merchant/export/list', { params }));
export const getExportLink = (id) =>
  data(request.get(`/merchant/export/${id}/link`));
export const exportDownloadUrl = (token) =>
  `${import.meta.env.VITE_API_URL}/export/download/${token}`;

// ---- 告警 ----
export const getAlertRules = () => data(request.get('/merchant/alert/rules'));
export const saveAlertRule = (body) =>
  data(request.post('/merchant/alert/rules', body));
export const deleteAlertRule = (id) =>
  data(request.delete(`/merchant/alert/rules/${id}`));
export const pageAlerts = async (params) =>
  ok(await request.get(`${prefix}/alert/list`, { params }));
export const getAlertStats = () => data(request.get(`${prefix}/alert/stats`));
export const resolveAlert = (id) =>
  data(request.post(`${prefix}/alert/${id}/resolve`));

// ---- 通知 ----
export const getNotifySettings = () =>
  data(request.get('/merchant/notify/settings'));
export const saveNotifySettings = (body) =>
  data(request.put('/merchant/notify/settings', body));
export const testNotifyChannel = (channel) =>
  data(request.post(`/merchant/notify/test/${channel}`));
export const pageNotifications = async (params) =>
  ok(await request.get(`${prefix}/notification/list`, { params }));
export const getUnreadCount = () =>
  data(request.get(`${prefix}/notification/unread`));
export const markNotificationsRead = (ids = []) =>
  data(request.post(`${prefix}/notification/read`, { ids }));

// ---- Webhook ----
export const getWebhooks = () => data(request.get('/merchant/webhook/list'));
export const createWebhook = (body) =>
  data(request.post('/merchant/webhook', body));
export const updateWebhook = (id, body) =>
  data(request.put(`/merchant/webhook/${id}`, body));
export const rotateWebhookSecret = (id) =>
  data(request.post(`/merchant/webhook/${id}/rotate`));
export const testWebhook = (id) =>
  data(request.post(`/merchant/webhook/${id}/test`));
export const deleteWebhook = (id) =>
  data(request.delete(`/merchant/webhook/${id}`));
export const pageDeliveries = async (params) =>
  ok(await request.get('/merchant/webhook/deliveries', { params }));
export const getDelivery = (id) =>
  data(request.get(`/merchant/webhook/deliveries/${id}`));
export const redeliver = (id) =>
  data(request.post(`/merchant/webhook/deliveries/${id}/redeliver`));
