<!-- 分段器 -->
<template>
  <ElTabs
    ref="tabRef"
    :modelValue="modelValue"
    :stretch="block"
    class="ele-segmented"
    :class="[
      { 'is-block': block },
      { 'is-large': size === 'large' },
      { 'is-small': size === 'small' },
      { 'is-disabled': disabled },
      { 'is-single': items && items.length === 1 },
      { 'is-empty': isEmptyActive }
    ]"
    @update:modelValue="handleUpdateModelValue"
  >
    <ElTabPane
      v-for="(item, index) in items"
      :key="`${index}-${item.value}-${item.label}`"
      :name="item.value"
      :disabled="disabled || item.disabled"
    >
      <template #label>
        <slot name="label" :item="item">
          <ElIcon
            v-if="item.icon"
            v-bind="item.iconProps || {}"
            class="ele-segmented-item-icon"
          >
            <component :is="item.icon" :style="item.iconStyle" />
          </ElIcon>
          <div v-if="item.label" class="ele-segmented-item-label">
            {{ item.label }}
          </div>
        </slot>
      </template>
    </ElTabPane>
  </ElTabs>
</template>

<script setup>
  import { ref, computed } from 'vue';
  import { ElIcon, ElTabs, ElTabPane } from 'element-plus';
  import { useFormValidate } from '../ele-basic-select/util';
  import { segmentedProps, segmentedEmits } from './props';

  defineOptions({ name: 'EleSegmented' });

  const props = defineProps(segmentedProps);

  const emit = defineEmits(segmentedEmits);

  const { validateChange } = useFormValidate();

  /** 选项卡 */
  const tabRef = ref(null);

  /** 是否没有选中 */
  const isEmptyActive = computed(() => {
    return (
      props.modelValue == null ||
      !props.items ||
      !props.items.some((d) => d.value === props.modelValue)
    );
  });

  /** 选中改变事件 */
  const handleUpdateModelValue = (value) => {
    if (props.modelValue !== value) {
      emit('update:modelValue', value);
      if (props.validateEvent) {
        validateChange();
      }
      emit('change', value);
    }
  };

  /** 获取容器节点 */
  const getWrapEl = () => {
    const tabEl = tabRef.value ? tabRef.value.$el : void 0;
    const headerEl = tabEl ? tabEl.querySelector('.el-tabs__header') : void 0;
    return headerEl ? headerEl.querySelector('.el-tabs__nav') : void 0;
  };

  /** 更新选中指示器位置 */
  const updateActiveBar = () => {
    const el = getWrapEl();
    if (el) {
      const bar = el.querySelector('.el-tabs__active-bar');
      if (bar) {
        bar.style.width = '0px';
      }
    }
  };

  defineExpose({
    tabRef,
    updateActiveBar
  });
</script>
