import { stepsProps as elStepsProps } from 'element-plus';

/**
 * 属性
 */
export const stepsProps = {
  ...elStepsProps,
  /** 步骤条数据 */
  items: Array,
  /** 类型 */
  type: String
};
