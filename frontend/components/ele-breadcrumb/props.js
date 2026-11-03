import { breadcrumbProps as elBreadcrumbProps } from 'element-plus';

/**
 * 属性
 */
export const breadcrumbProps = {
  ...elBreadcrumbProps,
  separator: {
    type: [String, Object, Function],
    default: '/'
  },
  /** 面包屑数据 */
  items: {
    type: Array,
    required: true
  }
};
