/**
 * 属性
 */
export const cronPanelProps = {
  /** 绑定值 */
  modelValue: String,
  /** 国际化 */
  locale: Object
};

/**
 * 事件
 */
export const cronPanelEmits = {
  /** 更新绑定值 */
  'update:modelValue': (_value) => true
};
