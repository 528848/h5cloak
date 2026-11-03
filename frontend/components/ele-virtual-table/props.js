import { dataTableProps, dataTableEmits } from '../ele-data-table/props';

/**
 * 属性
 */
export const virtualTableProps = {
  ...dataTableProps,
  /** 行高 */
  rowHeight: Number
};

/**
 * 事件
 */
export const virtualTableEmits = {
  ...dataTableEmits,
  endEeached: (_params) => true,
  scroll: (_params) => true,
  rowsRendered: (_params) => true
};
