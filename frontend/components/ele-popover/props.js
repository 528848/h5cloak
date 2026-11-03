import {
  popoverProps as elPopoverProps,
  popoverEmits as elPopoverEmits
} from 'element-plus';
import { omit } from '../utils/common';
import { tooltipProps } from '../ele-tooltip/props';
const normalizeProps = omit(elPopoverProps, ['onUpdate:visible']);

/**
 * 属性
 */
export const popoverProps = {
  ...omit(tooltipProps, ['rawContent', 'isPopover']),
  popperStyle: [String, Array, Object],
  popperClass: String,
  ...normalizeProps,
  transition: {
    type: String,
    default: 'el-fade-in-linear'
  },
  /** 自定义主体类名 */
  bodyClass: String,
  /** 自定义主体样式 */
  bodyStyle: Object,
  /** 自定义标题样式 */
  titleStyle: Object,
  /** 自定义内容样式 */
  contentStyle: Object
};

/**
 * 事件
 */
export const popoverEmits = elPopoverEmits;
