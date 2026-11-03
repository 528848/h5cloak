import { paginationProps as elPaginationProps } from 'element-plus';

/**
 * 属性
 */
export const paginationProps = {
  ...elPaginationProps,
  /** 总条目数 */
  total: [String, Number],
  /** 是否还有下一页 */
  hasNext: {
    type: Boolean,
    default: true
  },
  /** 风格 */
  type: {
    type: String,
    default: 'circle'
  },
  /** 每页数量选择下拉是否使用固定定位 */
  isFixedPopper: {
    type: Boolean,
    default: true
  }
};

/**
 * 事件
 */
export const paginationEmits = {
  /** 更新页码 */
  'update:currentPage': (_currentPage) => true,
  /** 更新每页数量 */
  'update:pageSize': (_pageSize) => true
};
