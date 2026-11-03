import { checkboxGroupProps as elCheckboxGroupProps } from 'element-plus';
export { checkboxGroupEmits } from 'element-plus';

/**
 * 属性
 */
export const checkboxGroupProps = {
  ...elCheckboxGroupProps,
  /** 风格类型 */
  type: String,
  /** 选项数据 */
  options: [Array, Function]
};
