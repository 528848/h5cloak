<template>
  <ElScrollbar class="ele-icon-select-body">
    <div
      class="ele-icon-select-grid"
      :class="{ 'is-loose': tooltip === 'static' }"
      :style="gridStyle"
    >
      <div
        v-for="(d, i) in data"
        :key="`${i}-${d}`"
        class="ele-icon-select-item"
        :class="{ 'is-active': icon && d === icon }"
        :title="!tooltip || tooltip === 'static' ? d : void 0"
        :style="itemStyle"
        @click="handleItemClick(d)"
        @mouseover="(e) => handleItemHover(d, e)"
      >
        <slot name="icon" :icon="d" :prefix="false"></slot>
        <div v-if="tooltip === 'static'" class="ele-icon-select-item-name">
          {{ d }}
        </div>
      </div>
    </div>
    <div v-if="!data || !data.length" class="ele-icon-select-empty">
      <ElEmpty :imageSize="60" v-bind="emptyProps || {}" />
    </div>
    <EleTooltip
      v-if="tooltip === true"
      placement="top"
      :offset="6"
      :teleported="popperType !== 'popper'"
      v-bind="tooltipProps || {}"
      :visible="tooltipVisible"
      :content="tooltipContent"
      :virtualRef="virtualRef"
      :virtualTriggering="true"
      @update:visible="updateTooltipVisible"
    />
  </ElScrollbar>
</template>

<script setup>
  import { ref, watch } from 'vue';
  import { ElEmpty, ElScrollbar } from 'element-plus';
  import EleTooltip from '../../ele-tooltip/index.vue';

  defineOptions({ name: 'IconGrid' });

  const props = defineProps({
    /** 选中的图标 */
    icon: String,
    /** 图标数据 */
    data: {
      type: Array,
      required: true
    },
    /** 空组件属性 */
    emptyProps: Object,
    /** 是否显示提示 */
    tooltip: [Boolean, String],
    /** 提示属性 */
    tooltipProps: Object,
    /** 气泡是否展开 */
    popperVisible: Boolean,
    /** 网格样式 */
    gridStyle: Object,
    /** 图标样式 */
    itemStyle: Object,
    /** 下拉组件类型 */
    popperType: String
  });

  const emit = defineEmits({
    select: (_icon) => true
  });

  /** 提示组件是否显示 */
  const tooltipVisible = ref(false);

  /** 提示文本 */
  const tooltipContent = ref('');

  /** 提示单例目标 */
  const virtualRef = ref();

  /** 更新提示显示状态 */
  const updateTooltipVisible = (visible) => {
    tooltipVisible.value = visible;
  };

  /** 关闭提示 */
  const hideTooltip = () => {
    updateTooltipVisible(false);
  };

  /** 打开提示 */
  const handleItemHover = (icon, e) => {
    if (
      props.tooltip === true &&
      (props.popperVisible || props.popperType === 'default') &&
      icon
    ) {
      virtualRef.value = e.currentTarget;
      tooltipContent.value = icon;
      tooltipVisible.value = true;
    }
  };

  /** 点击事件 */
  const handleItemClick = (icon) => {
    emit('select', icon);
  };

  watch(
    () => props.popperVisible,
    (visible) => {
      if (!visible) {
        hideTooltip();
      }
    }
  );

  defineExpose({
    hideTooltip
  });
</script>
