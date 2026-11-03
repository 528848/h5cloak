import { transferProps as elTransferProps } from 'element-plus';
export { transferEmits } from 'element-plus';

/**
 * 属性
 */
export const transferProps = {
  ...elTransferProps,
  /** 数据 */
  data: [Array, Function]
};
