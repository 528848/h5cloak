/**
 * 属性
 */
export const timelineProps = {
  /** 数据 */
  data: Array,
  /** 每项最小宽度 */
  itemWidth: {
    type: Number,
    default: 168
  }
};

/**
 * 事件
 */
export const timelineEmits = {
  itemClick: (_item, _e) => true
};
