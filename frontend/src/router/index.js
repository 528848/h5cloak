/**
 * 路由配置
 */
import NProgress from 'nprogress';
import { createRouter, createWebHistory } from 'vue-router';
import { LOGIN_PATH, REDIRECT_PATH, LAYOUT_PATH } from '@/config/setting';
import { EleMessage } from 'ele-admin-plus';
import { useUserStore } from '@/store/modules/user';
import { getToken, removeToken } from '@/utils/token-util';
import { isPortalUser, portalDeniedMessage } from '@portal';
import { setPageTitle } from '@/utils/page-title-util';
import {
  routes,
  getMenuRoutes,
  isWhiteList,
  addNotFoundRoute
} from './routes';

NProgress.configure({
  speed: 200,
  minimum: 0.02,
  trickleSpeed: 200,
  showSpinner: false
});

const router = createRouter({
  routes,
  history: createWebHistory(),
  scrollBehavior: () => {
    return { top: 0 };
  }
});

/**
 * 路由守卫
 */
router.beforeEach(async (to) => {
  if (!to.path.includes(REDIRECT_PATH)) {
    NProgress.start();
    setPageTitle(to.meta?.title);
  }
  if (!getToken()) {
    // 未登录跳转登录界面
    if (!isWhiteList(to.path)) {
      const query = { from: encodeURIComponent(to.fullPath) };
      return { path: LOGIN_PATH, query: to.path === LAYOUT_PATH ? {} : query };
    }
    return;
  }
  // 注册动态路由
  const userStore = useUserStore();
  if (!userStore.menus) {
    const { menus, homePath } = await userStore.fetchUserInfo(to);
    if (menus) {
      if (!isPortalUser(userStore.info)) {
        removeToken();
        userStore.clearData();
        EleMessage.error({ message: portalDeniedMessage, plain: true });
        return { path: LOGIN_PATH, replace: true };
      }
      getMenuRoutes(menus, homePath).forEach((r) => {
        router.addRoute(r);
      });
      addNotFoundRoute(router);
      return { ...to, replace: true };
    }
    // token 无效或接口异常时跳转登录页
    removeToken();
    userStore.clearData();
    return { path: LOGIN_PATH, replace: true };
  }
});

router.afterEach((to) => {
  if (!to.path.includes(REDIRECT_PATH) && NProgress.isStarted()) {
    setTimeout(() => {
      NProgress.done(true);
    }, 200);
  }
});

export default router;
