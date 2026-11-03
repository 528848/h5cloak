<!-- 数据编辑代码模式 -->
<template>
  <div class="ele-pro-form-builder-code-edit-wrapper">
    <component :is="codeEditerComponent || CodeEditer" v-model="codeContent" />
  </div>
</template>

<script setup>
  import { ref, onMounted } from 'vue';
  import CodeEditer from './code-editer.vue';

  defineOptions({ name: 'OptionsCode' });

  const props = defineProps({
    /** 选项数据代码 */
    data: [Array, String],
    /** 顶部提示内容 */
    codeTips: String,
    /** 默认提示示例代码 */
    codePlaceholder: String,
    /** 代码字符串前缀 */
    codePrefix: String,
    /** 代码编辑器组件 */
    codeEditerComponent: [String, Object, Function]
  });

  /** 代码内容 */
  const codeContent = ref('');

  /** 获取数据结果 */
  const getResult = () => {
    const code = codeContent.value;
    if (code == null || !code) {
      return;
    }
    return `${props.codePrefix}${code}`;
  };

  /** 解析数据 */
  onMounted(() => {
    if (props.data == null || typeof props.data !== 'string') {
      codeContent.value = props.codePlaceholder ?? '';
      return;
    }
    const data = props.data.trim();
    const codePrefix = props.codePrefix;
    if (data.startsWith(codePrefix)) {
      codeContent.value = data.slice(codePrefix.length);
      return;
    }
    codeContent.value = (data || props.codePlaceholder) ?? '';
  });

  defineExpose({
    getResult
  });
</script>
