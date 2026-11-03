import {
  tabsProps as elTabsProps,
  tabsEmits as elTabsEmits
} from 'element-plus';

/**
 * 属性
 */
export const tabsProps = {
  ...elTabsProps,
  /** 标签页数据 */
  items: {
    type: Array,
    required: true
  },
  /** 尺寸 */
  size: String,
  /** 风格类型 */
  type: String,
  /** 标签是否居中显示 */
  center: Boolean,
  /** 是否支持右键菜单 */
  contextMenu: [Boolean, Object],
  /** 右键菜单 */
  contextMenus: [Array, Function],
  /** 是否支持拖动排序 */
  sortable: Boolean,
  /** 是否支持鼠标滚轮滑动 */
  mousewheel: Boolean,
  /** 是否自己处理页签点击事件 */
  handleClick: Boolean,
  /** 内部表格是否弹性布局 */
  flexTable: [Boolean, String]
};

/**
 * 事件
 */
export const tabsEmits = {
  ...elTabsEmits,
  /** 页签点击事件 */
  tabItemClick: (_opt) => true,
  /** 页签右键菜单项点击事件 */
  tabContextMenu: (_opt) => true,
  /** 页签右键菜单打开事件 */
  tabContextOpen: (_ref, _item, _name) => true,
  /** 拖动排序改变事件 */
  tabSortChange: (_data) => true
};

/**
 * 标签页组件属性名
 */
export const tabPropKeys = Object.keys(elTabsProps);

/**
 * 标签页容器依赖注入key
 */
export const TAB_WRAP_KEY = Symbol('tabWrap');
