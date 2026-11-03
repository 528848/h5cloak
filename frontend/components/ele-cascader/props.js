import { cascaderProps as elCascaderProps } from 'element-plus';
export { cascaderEmits } from 'element-plus';

/**
 * 属性
 */
export const cascaderProps = {
  ...elCascaderProps,
  /** 选项数据 */
  options: [Array, Function],
  /** 是否多选 */
  multiple: Boolean
};
