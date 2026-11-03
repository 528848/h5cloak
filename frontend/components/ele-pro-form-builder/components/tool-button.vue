<!-- 操作按钮 -->
<template>
  <ElButton
    v-bind="buttonProps || {}"
    :title="tooltip"
    class="ele-pro-form-builder-tool-button"
    @click="handleClick"
  >
    <template v-if="$slots.default" #default>
      <slot></slot>
    </template>
  </ElButton>
</template>

<script setup>
  import { ElButton } from 'element-plus';

  defineOptions({ name: 'ToolButton' });

  defineProps({
    /** 按钮属性 */
    buttonProps: Object,
    /** 提示信息 */
    tooltip: String
  });

  const emit = defineEmits({
    click: (_event) => true
  });

  /** 点击事件 */
  const handleClick = (e) => {
    emit('click', e);
  };
</script>
