export const APP_PORTAL = 'merchant';

export const APP_NAME = 'H5Cloak 商户端';

export const PORTAL_PREFIX = '/merchant';

export const portalDeniedMessage = '该账号属于总后台，请打开总后台登录';

export const viewModules = {
  ...import.meta.glob('/src/views/merchant/**/index.vue'),
  ...import.meta.glob('/src/views/login/index.vue'),
  ...import.meta.glob('/src/views/exception/**/index.vue'),
  ...import.meta.glob('/src/views/profile/**/index.vue')
};

export function isPortalUser(user) {
  return Boolean(user?.merchantId);
}
