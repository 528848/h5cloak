import { selectProps as elSelectProps } from 'element-plus/es/components/select/src/select';

/**
 * 属性
 */
export const selectProps = {
  ...elSelectProps,
  /** 选项数据 */
  options: [Array, Function]
};

/**
 * 事件
 */
export const selectEmits = {
  'update:modelValue': (_value) => true,
  change: (_value) => true,
  'remove-tag': (_value) => true,
  clear: () => true,
  'visible-change': (_visible) => true,
  focus: (_e) => true,
  blur: (_e) => true
};
