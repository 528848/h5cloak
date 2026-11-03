/**
 * 属性
 */
export const toolbarProps = {
  /** 标题 */
  title: String,
  /** 二级标题 */
  subtitle: String,
  /** 主题风格 */
  theme: {
    type: String,
    default: 'plain'
  },
  /** 标题区样式 */
  titleStyle: Object,
  /** 内容区样式 */
  bodyStyle: Object,
  /** 按钮区样式 */
  toolsStyle: Object,
  /** 标题组件属性 */
  titleProps: Object,
  /** 二级标题组件属性 */
  subtitleProps: Object
};

/**
 * 共享数据key
 */
export const TOOLBAR_KEY = Symbol('toolbar');
