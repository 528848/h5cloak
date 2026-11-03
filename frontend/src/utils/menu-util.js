import { mapTree, isExternalLink } from 'ele-admin-plus';
import { APP_PORTAL, PORTAL_PREFIX } from '@portal';
const API_BASE_URL = import.meta.env.VITE_API_URL;

/** 若依默认菜单图标名称转换 */
const ruoYiIcons = {
  system: 'IconProSettingOutlined',
  user: 'IconProUserOutlined',
  peoples: 'IconProIdcardOutlined',
  'tree-table': 'IconProAppstoreOutlined',
  tree: 'IconProCityOutlined',
  post: 'IconProSuitcaseOutlined',
  dict: 'IconProBookOutlined',
  edit: 'IconProControlOutlined',
  message: 'IconProMessageOutlined',
  log: 'IconProLogOutlined',
  form: 'IconProFileOutlined',
  logininfor: 'IconProCalendarOutlined',
  monitor: 'IconProDashboardOutlined',
  online: 'IconProConnectionOutlined',
  job: 'IconProTimerOutlined',
  druid: 'IconProFundOutlined',
  server: 'IconProAnalysisOutlined',
  redis: 'IconProClusterOutlined',
  'redis-list': 'IconProDatabaseOutlined',
  tool: 'IconProAppstoreAddOutlined',
  build: 'IconProFormOutlined',
  code: 'IconProCodeOutlined',
  swagger: 'IconProLinkOutlined',
  guide: 'IconProLinkOutlined',
  '#': 'IconProLinkOutlined'
};

/**
 * 菜单图标名称转换以兼容若依
 * @param icon 图标名称
 */
export function getRuoYiIcon(icon) {
  if (!icon) {
    return;
  }
  if (icon.startsWith('Icon')) {
    return icon;
  }
  return ruoYiIcons[icon] ?? 'IconProLinkOutlined';
}

/**
 * 组件路径处理以兼容若依
 * @param component 组件路径
 */
function formatComponent(component) {
  if (!component || component === 'Layout' || component === '#') {
    return;
  }
  if (isExternalLink(component)) {
    return component;
  }
  return component.startsWith('/') ? component : `/${component}`;
}

/**
 * 菜单地址处理以兼容若依
 * @param mPath 菜单地址
 * @param pPath 父级菜单地址
 * @param query 路由参数
 */
function formatPath(mPath, pPath, query) {
  if (!mPath || isExternalLink(mPath)) {
    return { path: mPath };
  }
  const path = !pPath || mPath.startsWith('/') ? mPath : `${pPath}/${mPath}`;
  if (query) {
    try {
      const params = new URLSearchParams(JSON.parse(query)).toString();
      if (params) {
        return { path: `${path}?${params}`, rPath: path };
      }
    } catch (e) {
      console.error(e);
    }
  }
  return { path };
}

/**
 * 菜单数据处理为 EleProLayout 所需要的格式
 * @param data 菜单数据
 * @param childField 子级的字段名称
 */
function formatMenus(data, childField = 'children') {
  let homePath;
  let homeTitle;
  const menus = mapTree(
    data,
    (item, _index, parent) => {
      const meta =
        (typeof item.meta === 'string'
          ? JSON.parse(item.meta || '{}')
          : item.meta) || {};
      const { path, rPath } = formatPath(item.path, parent?.path, item.query);
      const menu = {
        path: path,
        component: formatComponent(item.component),
        meta: {
          hide: !!item.hidden,
          keepAlive: !meta.noCache,
          routePath: rPath,
          ...meta
        }
      };
      const children = item[childField]
        ? item[childField].filter((d) => !(d.meta?.hide ?? d.hide))
        : void 0;
      if (!children?.length) {
        if (!homePath && menu.path && !isExternalLink(menu.path)) {
          homePath = menu.path;
          homeTitle = menu.meta?.title;
        }
      } else {
        const childPath = children[0].path;
        if (childPath) {
          if (!menu.component && !menu.redirect) {
            menu.redirect = childPath;
          }
          if (!menu.path) {
            menu.path = childPath.substring(0, childPath.lastIndexOf('/'));
          }
        }
      }
      if (!menu.path) {
        console.error('菜单path不能为空且要唯一:', item);
        return;
      }
      return menu;
    },
    childField
  );
  return { menus, homePath, homeTitle };
}

/**
 * 处理用户菜单数据以兼容若依
 * @param userMenu 用户菜单
 */
export function formatUserMenu(userMenu) {
  // 修改图标
  const data = mapTree(userMenu, (item) => {
    const meta =
      (item.meta && typeof item.meta === 'string'
        ? JSON.parse(item.meta)
        : item.meta) || {};
    const temp = { ...item, meta: { ...meta, icon: getRuoYiIcon(meta.icon) } };
    // 修改内嵌格式
    if ('tool/swagger/index' === item.component) {
      temp.component = `${API_BASE_URL}/swagger-ui/index.html`;
    } else if ('monitor/druid/index' === item.component) {
      temp.component =
        import.meta.env.VITE_APP_MONITOR_DRUID ||
        `${API_BASE_URL}/druid/login.html`;
    }
    return temp;
  });
  // 一级菜单去掉父级
  data.forEach((item, i) => {
    if (item.path === '/') {
      const child = item.children[0];
      data[i] = {
        ...child,
        path:
          child.path?.startsWith?.('/') || isExternalLink(child.path)
            ? child.path
            : `/${child.path}`
      };
    }
  });
  const scopedMenus = data.filter((item) =>
    String(item.path || '').startsWith(PORTAL_PREFIX)
  );
  // 增加个人中心（隐藏菜单，供右上角入口跳转）
  scopedMenus.push({
    path: '/profile',
    component: 'profile',
    meta: {
      title: '个人中心',
      icon: 'IconProUserOutlined',
      hide: true
    }
  });
  if (APP_PORTAL === 'merchant') {
    scopedMenus.push({
      path: '/merchant/api-key/guide',
      component: 'merchant/api-key/guide',
      meta: {
        title: 'API 使用教程',
        icon: 'IconProLinkOutlined',
        hide: true,
        activeMenu: '/merchant/api-key'
      }
    });
  }
  return formatMenus(scopedMenus);
}
