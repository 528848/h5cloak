<!-- eslint-disable vue/no-v-html -->
<template>
  <ElTooltip
    v-bind="rootProps"
    ref="tooltipRef"
    @update:visible="handleUpdateVisible"
    @before-show="handlePopBeforeShow"
    @before-hide="handlePopBeforeHide"
    @show="handlePopShow"
    @hide="handlePopHide"
    @open="handlePopOpen"
    @close="handlePopClose"
  >
    <slot v-if="$slots.default"></slot>
    <template #content>
      <slot name="body">
        <div
          v-if="rawContent && content != null && content != ''"
          v-html="content"
          class="ele-tooltip-body"
          :style="bodyStyle"
        ></div>
        <div v-else class="ele-tooltip-body" :style="bodyStyle">
          <slot name="content">{{ content }}</slot>
        </div>
      </slot>
    </template>
  </ElTooltip>
</template>

<script setup>
  import { ref, computed } from 'vue';
  import { ElTooltip } from 'element-plus';
  import { omit } from '../utils/common';
  import { tooltipProps, tooltipEmits } from './props';

  defineOptions({ name: 'EleTooltip' });

  const props = defineProps(tooltipProps);

  const emit = defineEmits(tooltipEmits);

  /** 根组件实例 */
  const tooltipRef = ref();

  /** 根组件属性 */
  const rootProps = computed(() => {
    const options = omit(props, [
      'content',
      'bodyStyle',
      'bg',
      'arrowBg',
      'width',
      'isPopover',
      'onUpdate:visible',
      'onBeforeShow',
      'onBeforeHide',
      'onBefore-show',
      'onBefore-hide',
      'onShow',
      'onHide',
      'onOpen',
      'onClose'
    ]);
    const classes = ['ele-popper'];
    if (props.isPopover && props.effect === 'light') {
      classes.push('ele-popover');
    } else {
      classes.push('ele-tooltip');
    }
    if (typeof props.popperClass === 'string' && props.popperClass) {
      classes.push(props.popperClass);
    }
    options.popperClass = classes.join(' ');

    const style = {};
    if (props.bg || props.arrowBg) {
      options.effect = 'dark';
      if (props.bg) {
        style['--ele-tooltip-bg'] = props.bg;
      }
      if (props.arrowBg) {
        style['--ele-tooltip-arrow-bg'] = props.arrowBg;
      }
    }
    const { width, popperStyle } = props;
    if (width != null) {
      style.width = typeof width === 'string' ? width : `${width}px`;
    }
    if (!popperStyle) {
      options.popperStyle = style;
    } else if (Array.isArray(popperStyle)) {
      options.popperStyle = [style, ...popperStyle];
    } else {
      options.popperStyle = [style, popperStyle];
    }
    return options;
  });

  /** 气泡实例 */
  const popperRef = computed(() => tooltipRef.value?.popperRef);

  /** 内容实例 */
  const contentRef = computed(() => tooltipRef.value?.contentRef);

  /** 验证当前焦点是否在气泡内 */
  const isFocusInsideContent = () => {
    tooltipRef.value && tooltipRef.value.isFocusInsideContent();
  };

  /** 更新气泡 */
  const updatePopper = () => {
    tooltipRef.value && tooltipRef.value.updatePopper();
  };

  /** 打开气泡 */
  const handleOpen = (opt) => {
    tooltipRef.value && tooltipRef.value.onOpen(opt);
  };

  /** 关闭气泡 */
  const handleClose = (opt) => {
    tooltipRef.value && tooltipRef.value.onClose(opt);
  };

  /** 关闭气泡 */
  const hide = (opt) => {
    tooltipRef.value && tooltipRef.value.hide(opt);
  };

  /** 更新打开状态 */
  const handleUpdateVisible = (visible) => {
    emit('update:visible', visible);
  };

  /** 开始打开事件 */
  const handlePopBeforeShow = (e) => {
    emit('before-show', e);
  };

  /** 开始关闭事件 */
  const handlePopBeforeHide = (e) => {
    emit('before-hide', e);
  };

  /** 打开事件 */
  const handlePopShow = (e) => {
    emit('show', e);
  };

  /** 关闭事件 */
  const handlePopHide = (e) => {
    emit('hide', e);
  };

  /** 打开事件 */
  const handlePopOpen = (e) => {
    emit('open', e);
  };

  /** 关闭事件 */
  const handlePopClose = (e) => {
    emit('close', e);
  };

  defineExpose({
    tooltipRef,
    popperRef,
    contentRef,
    isFocusInsideContent,
    updatePopper,
    handleOpen,
    handleClose,
    onOpen: handleOpen,
    onClose: handleClose,
    hide
  });
</script>
