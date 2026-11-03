import { autocompleteProps as elAutocompleteProps } from 'element-plus';
export { autocompleteEmits } from 'element-plus';

/**
 * 属性
 */
export const autocompleteProps = {
  ...elAutocompleteProps,
  /** 建议数据 */
  fetchSuggestions: [Array, Function]
};
