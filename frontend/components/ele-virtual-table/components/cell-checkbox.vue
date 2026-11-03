<!-- 单元格复选框 -->
<template>
  <ElCheckbox
    :modelValue="checked"
    :indeterminate="indeterminate"
    :disabled="disabled"
    :size="size"
    @update:modelValue="handleUpdateModelValue"
  />
</template>

<script setup>
  import { ElCheckbox } from 'element-plus';

  defineOptions({ name: 'CellCheckbox' });

  const props = defineProps({
    /** 是否是选中状态 */
    checked: Boolean,
    /** 是否是半选状态 */
    indeterminate: Boolean,
    /** 是否是禁用状态 */
    disabled: Boolean,
    /** 尺寸 */
    size: String
  });

  const emit = defineEmits({
    /** 选中改变事件 */
    change: (_checked) => true
  });

  /** 选中改变事件 */
  const handleUpdateModelValue = (modelValue) => {
    if (props.checked !== modelValue) {
      emit('change', modelValue);
    }
  };
</script>
