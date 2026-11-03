<!-- 文本省略 -->
<template>
  <EleText
    v-bind="textProps"
    ref="textRef"
    :style="rootStyle"
    class="ele-ellipsis"
    :class="{ 'is-multi-line': multiLine }"
    @mouseover="handleHover"
  >
    <slot></slot>
    <EleTooltip v-if="virtualRef" v-bind="tooltipProps" />
  </EleText>
</template>

<script setup>
  import { ref, computed } from 'vue';
  import { omit, pick, contentIsEllipsis } from '../utils/common';
  import EleTooltip from '../ele-tooltip/index.vue';
  import EleText from '../ele-text/index.vue';
  import { textPropKeys } from '../ele-text/props';
  import { ellipsisProps } from './props';

  defineOptions({ name: 'EleEllipsis' });

  const props = defineProps(ellipsisProps);

  /** 文本组件 */
  const textRef = ref(null);

  /** 提示单例目标 */
  const virtualRef = ref();

  /** 提示文本 */
  const text = ref('');

  /** 原生的提示内容 */
  const title = computed(() => {
    if (
      !props.tooltip ||
      typeof props.tooltip !== 'object' ||
      props.tooltip.original !== true
    ) {
      return;
    }
    return text.value;
  });

  /** 提示组件属性 */
  const tooltipProps = computed(() => {
    const isObj = props.tooltip && props.tooltip !== true;
    const opt = isObj ? { ...props.tooltip } : {};
    opt.content = text.value;
    opt.virtualRef = virtualRef.value;
    opt.virtualTriggering = true;
    return omit(opt, ['original']);
  });

  /** 是否是多行 */
  const multiLine = computed(() => {
    return !!(props.maxLine && props.maxLine > 1);
  });

  /** 文本组件属性 */
  const textProps = computed(() => {
    return Object.assign({ title: title.value }, pick(props, textPropKeys));
  });

  /** 根节点样式 */
  const rootStyle = computed(() => {
    const style = {};
    const { lineHeight, maxLine } = props;
    if (lineHeight != null) {
      const h = typeof lineHeight === 'number' ? `${lineHeight}px` : lineHeight;
      style.lineHeight = h;
      if (multiLine.value) {
        style.height = `calc(${h} * ${maxLine})`;
      }
    }
    if (multiLine.value) {
      style['-webkit-line-clamp'] = maxLine;
    }
    return style;
  });

  /** 打开提示 */
  const handleHover = (e) => {
    if (!props.tooltip) {
      virtualRef.value = void 0;
      return;
    }
    const target = e.currentTarget;
    // 获取提示文本
    if (typeof props.tooltip === 'object' && props.tooltip.content) {
      if (text.value !== props.tooltip.content) {
        text.value = props.tooltip.content;
      }
    } else {
      if (target) {
        const temp = contentIsEllipsis(target) ? target.innerText : '';
        if (text.value !== temp) {
          text.value = temp;
        }
      }
    }
    // 判断是原生提示还是组件提示
    if (text.value && (props.tooltip === true || !props.tooltip.original)) {
      if (virtualRef.value !== target) {
        virtualRef.value = target;
      }
    } else {
      virtualRef.value = void 0;
    }
  };

  defineExpose({
    textRef
  });
</script>
