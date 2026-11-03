import { radioGroupProps as elRadioGroupProps } from 'element-plus';
export { radioGroupEmits } from 'element-plus';

/**
 * 属性
 */
export const radioGroupProps = {
  ...elRadioGroupProps,
  /** 风格类型 */
  type: String,
  /** 选项数据 */
  options: [Array, Function]
};
