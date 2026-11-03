<!-- 编辑弹窗 -->
<template>
  <ele-modal
    :width="720"
    title="创建表"
    :body-style="{ paddingTop: '6px' }"
    :maxable="true"
    :loading="loading"
    v-bind="modalProps"
  >
    <div style="margin-bottom: 6px">创建表语句（支持多个建表语句）：</div>
    <el-input
      :rows="12"
      type="textarea"
      v-model="sql"
      placeholder="请输入创建表语句"
    />
    <template #footer>
      <btn-items
        :items="[
          { preset: 'cancel', onClick: () => handleCancel() },
          { preset: 'save', onClick: () => handleSave() }
        ]"
      />
    </template>
  </ele-modal>
</template>

<script setup>
  import { ref } from 'vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { createTable } from '@/api/tool/gen';

  const emit = defineEmits(['done']);

  const { modalProps, closeModal } = useModal();

  /** 创建表语句 */
  const sql = ref('');

  /** 提交状态 */
  const loading = ref(false);

  /** 关闭弹窗 */
  const handleCancel = () => {
    closeModal();
  };

  /** 保存编辑 */
  const handleSave = () => {
    if (!sql.value) {
      EleMessage.error({ message: '请输入创建表语句', plain: true });
      return;
    }
    loading.value = true;
    createTable(sql.value)
      .then((msg) => {
        loading.value = false;
        EleMessage.success({ message: msg, plain: true });
        emit('done');
        handleCancel();
      })
      .catch((e) => {
        loading.value = false;
        EleMessage.error({ message: e.message, plain: true });
      });
  };
</script>
