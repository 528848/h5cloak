<!-- 复杂值编辑 -->
<template>
  <ElInput
    :rows="4"
    size="small"
    type="textarea"
    v-model="inputValue"
    :placeholder="placeholder"
    @change="handleInputChange"
  />
</template>

<script setup>
  import { ref, watch } from 'vue';
  import { ElInput } from 'element-plus';

  defineOptions({ name: 'JsonInput' });

  const props = defineProps({
    /** 绑定值 */
    modelValue: [String, Boolean, Number, Object, Array],
    /** 提示文本 */
    placeholder: String
  });

  const emit = defineEmits({
    'update:modelValue': (_data) => true
  });

  /** 输入框值 */
  const inputValue = ref('');

  /** 转 json 字符串 */
  const stringifyJSON = (data) => {
    if (data == null) {
      return '';
    }
    return JSON.stringify(data, void 0, 2);
  };

  /** 更新绑定值 */
  const updateModelValue = (json) => {
    if (stringifyJSON(props.modelValue) !== json) {
      try {
        const isEmpty = json == null || json === '' || json.trim() === '';
        const value = isEmpty ? void 0 : JSON.parse(json);
        emit('update:modelValue', value);
      } catch (e) {
        console.error(e);
      }
    }
  };

  /** 更新数据 */
  const handleInputChange = (value) => {
    if (value != null) {
      updateModelValue(value);
    }
  };

  /** 同步值 */
  watch(
    () => props.modelValue,
    (value) => {
      const json = stringifyJSON(value);
      if (inputValue.value !== json) {
        inputValue.value = json;
      }
    },
    {
      deep: true,
      immediate: true
    }
  );
</script>
