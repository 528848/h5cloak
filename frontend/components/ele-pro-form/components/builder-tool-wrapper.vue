<template>
  <div class="ele-pro-form-builder-item-tool-wrapper">
    <div
      class="ele-pro-form-builder-item-handle"
      :class="{ 'is-disabled': !handle }"
    >
      <ElIcon v-if="handle" class="ele-pro-form-builder-item-handle-icon">
        <DragOutlined />
      </ElIcon>
      <slot
        name="builderItemHandleContent"
        :item="item"
        :activeItemKey="activeItemKey"
      >
        <div class="ele-pro-form-builder-item-handle-content">
          {{ item.type }}
        </div>
      </slot>
    </div>
    <div class="ele-pro-form-builder-item-tools">
      <slot
        name="builderItemTools"
        :item="item"
        :activeItemKey="activeItemKey"
      ></slot>
    </div>
  </div>
</template>

<script setup>
  import { ElIcon } from 'element-plus';
  import { DragOutlined } from '../../icons/index';

  defineOptions({ name: 'BuilderToolWrapper' });

  defineProps({
    /** 表单项 */
    item: Object,
    /** 编辑模式选中的表单项 */
    activeItemKey: [String, Number],
    /** 是否需要拖拽手柄 */
    handle: Boolean
  });
</script>
