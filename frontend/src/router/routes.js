import { menuToRoutes, eachTree } from 'ele-admin-plus';
import {
  LOGIN_PATH,
  HOME_PATH,
  LAYOUT_PATH,
  REDIRECT_PATH,
  WHITE_LIST
} from '@/config/setting';
import Layout from '@/layout/index.vue';
import RedirectLayout from '@/components/RedirectLayout/index.vue';
import { APP_PORTAL, viewModules } from '@portal';

/**
 * 静态路由
 */
export const routes = [
  {
    path: LOGIN_PATH,
    component: () => import('@/views/login/index.vue'),
    meta: { title: '登录' }
  }
];

/** 404 路由名称 */
export const NOT_FOUND_ROUTE_NAME = 'NotFound';

/**
 * 注册 404 兜底路由（需在动态菜单路由之后注册）
 * @param router 路由实例
 */
export function addNotFoundRoute(router) {
  if (!router.hasRoute(NOT_FOUND_ROUTE_NAME)) {
    router.addRoute({
      name: NOT_FOUND_ROUTE_NAME,
      path: '/:path(.*)*',
      component: () => import('@/views/exception/404/index.vue')
    });
  }
}

/**
 * 根据菜单生成动态路由
 * @param menus 菜单数据
 * @param homePath 主页地址
 */
export function getMenuRoutes(menus, homePath) {
  const childRoutes = [
    // 用于刷新的路由
    {
      path: REDIRECT_PATH + '/:path(.*)',
      component: RedirectLayout,
      meta: { hideFooter: true }
    }
  ];
  const layoutRoutes = [
    {
      path: LAYOUT_PATH,
      component: Layout,
      redirect: HOME_PATH ?? homePath,
      children: childRoutes
    }
  ];
  // 路由铺平处理
  eachTree(menuToRoutes(menus, getComponent, routes), (route) => {
    const temp = Object.assign({}, route, { children: void 0 });
    if (route.children?.length && !route.component) {
      temp.redirect = route.children[0].path; // 重设redirect以兼容若依菜单数据
    }
    if (!temp.component && !temp.redirect) {
      // 没有对应组件的路由页面使用 404 组件
      temp.component = () => import('@/views/exception/404/index.vue');
    }
    if (temp.meta?.layout === false) {
      layoutRoutes.push(temp); // 不需要外层布局的路由
    } else {
      childRoutes.push(temp); // 需要外层布局的路由
    }
  });
  // 站点配置是独立表单页，不进菜单
  if (APP_PORTAL === 'merchant') {
    childRoutes.push({
      path: '/merchant/site/form',
      component: () => import('@/views/merchant/site/form/index.vue'),
      meta: { title: '站点配置', active: '/merchant/site' }
    });
    childRoutes.push({
      path: '/merchant/alert/telegram',
      component: () => import('@/views/merchant/alert/telegram/index.vue'),
      meta: { title: '配置 Telegram', active: '/merchant/alert/settings' }
    });
    childRoutes.push({
      path: '/merchant/alert/telegram-setup',
      component: () => import('@/views/merchant/alert/telegram-setup/index.vue'),
      meta: { title: 'Telegram 机器人', active: '/merchant/alert/settings' }
    });
  }
  if (APP_PORTAL === 'admin') {
    childRoutes.push({
      path: '/admin/template/form',
      component: () => import('@/views/admin/template/form/index.vue'),
      meta: { title: '模板配置' }
    });
  }
  return layoutRoutes;
}

/**
 * 判断是否是白名单路由
 * @param path 路由地址
 */
export function isWhiteList(path) {
  if (!path) {
    return false;
  }
  return WHITE_LIST.some((whitePath) => {
    if (whitePath === path) {
      return true;
    }
    if (whitePath.endsWith('*') && path.startsWith(whitePath.slice(0, -1))) {
      return true;
    }
    return false;
  });
}

/**
 * 解析路由组件
 * @param component 组件名称
 */
function getComponent(component) {
  if (component) {
    const module = viewModules[`/src/views${component}.vue`];
    if (!module) {
      return viewModules[`/src/views${component}/index.vue`];
    }
    return module;
  }
}
