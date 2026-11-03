import { useTooltipProps, tooltipEmits as elTooltipEmits } from 'element-plus';
import { omit } from '../utils/common';
const normalizeProps = omit(useTooltipProps, [
  'onUpdate:visible',
  'style',
  'onMouseenter',
  'onMouseleave',
  'onClick',
  'onKeydown',
  'onFocus',
  'onBlur',
  'onContextmenu'
]);

/**
 * 属性
 */
export const tooltipProps = {
  className: String,
  popperClass: String,
  popperStyle: Object,
  ...normalizeProps,
  /** 自定义内容样式 */
  bodyStyle: Object,
  /** 自定义背景色 */
  bg: String,
  /** 自定义箭头颜色 */
  arrowBg: String,
  /** 宽度 */
  width: [String, Number],
  /** 是否是气泡卡片 */
  isPopover: Boolean
};

/**
 * 事件
 */
export const tooltipEmits = elTooltipEmits;

/**
 * 属性名
 */
export const tooltipPropKeys = Object.keys(tooltipProps);
