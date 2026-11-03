<!-- 代码编辑器 -->
<template>
  <ElInput
    size="default"
    type="textarea"
    :modelValue="modelValue"
    @update:modelValue="updateModelValue"
    class="ele-pro-form-builder-code-editer"
  />
</template>

<script setup>
  import { ElInput } from 'element-plus';

  defineOptions({ name: 'CodeEditer' });

  defineProps({
    /** 绑定值 */
    modelValue: String
  });

  const emit = defineEmits({
    'update:modelValue': (_value) => true
  });

  /** 更新绑定值 */
  const updateModelValue = (value) => {
    emit('update:modelValue', value);
  };
</script>
