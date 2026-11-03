import { mentionProps as elMentionProps } from 'element-plus';
export { mentionEmits } from 'element-plus';

/**
 * 属性
 */
export const mentionProps = {
  ...elMentionProps,
  /** 数据 */
  options: [Array, Function]
};
