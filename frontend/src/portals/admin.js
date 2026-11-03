export const APP_PORTAL = 'admin';

export const APP_NAME = '斗篷系统总后台';

export const PORTAL_PREFIX = '/admin';

export const portalDeniedMessage = '该账号属于商户端，请打开商户后台登录';

export const viewModules = {
  ...import.meta.glob('/src/views/admin/**/index.vue'),
  ...import.meta.glob('/src/views/login/index.vue'),
  ...import.meta.glob('/src/views/exception/**/index.vue'),
  ...import.meta.glob('/src/views/profile/**/index.vue')
};

export function isPortalUser(user) {
  return user != null && (user.merchantId == null || user.merchantId === '');
}
