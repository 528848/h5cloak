import { iconProps as elIconProps } from 'element-plus';

/**
 * 属性
 */
export const iconProps = {
  ...elIconProps,
  /** 图标名称 */
  name: [String, Object, Function],
  /** 图标类型 */
  iconType: String
};
