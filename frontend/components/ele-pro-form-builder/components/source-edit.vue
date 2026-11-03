<!-- 源码编辑弹窗 -->
<template>
  <ElButton
    size="small"
    :icon="CodeOutlined"
    class="ele-pro-form-builder-props-fluid-btn"
    @click="openModal"
  >
    {{ title }}
  </ElButton>
  <EleModal
    :width="800"
    :maxable="true"
    position="center"
    :title="title"
    v-model="visible"
    :closeOnClickModal="false"
    :destroyOnClose="true"
    :bodyStyle="{
      height: '520px',
      minHeight: '100%',
      maxHeight: '100%',
      padding: '8px 16px'
    }"
  >
    <div class="ele-pro-form-builder-code-edit-wrapper">
      <component
        :is="jsonEditerComponent || CodeEditer"
        v-model="jsonContent"
      />
    </div>
    <template #footer>
      <ElButton size="default" @click="handleCancel">取消</ElButton>
      <ElButton type="primary" size="default" @click="handleSave">
        保存
      </ElButton>
    </template>
  </EleModal>
</template>

<script setup>
  import { ref } from 'vue';
  import { ElButton } from 'element-plus';
  import { CodeOutlined } from '../../icons/index';
  import { omit } from '../../utils/common';
  import EleModal from '../../ele-modal/index.vue';
  import CodeEditer from './code-editer.vue';

  defineOptions({ name: 'SourceEdit' });

  const props = defineProps({
    /** 数据 */
    modelValue: Object,
    /** 弹窗标题 */
    title: String,
    /** 需要排除编辑的字段 */
    excludeFields: Array,
    /** JSON 编辑器组件 */
    jsonEditerComponent: [String, Object, Function]
  });

  const emit = defineEmits({
    'update:modelValue': (_data) => true
  });

  /** 弹窗是否打开 */
  const visible = ref(false);

  /** JSON 内容 */
  const jsonContent = ref('');

  /** 打开弹窗 */
  const openModal = () => {
    jsonContent.value = JSON.stringify(
      omit(props.modelValue || {}, props.excludeFields || ['key', 'children']),
      void 0,
      2
    );
    visible.value = true;
  };

  /** 关闭弹窗 */
  const handleCancel = () => {
    visible.value = false;
    jsonContent.value = '';
  };

  /** 保存编辑 */
  const handleSave = () => {
    if (jsonContent.value) {
      let result;
      try {
        result = JSON.parse(jsonContent.value);
      } catch (e) {
        console.error(e);
      }
      if (result) {
        handleCancel();
        emit('update:modelValue', result);
      }
    }
  };
</script>
