<template>
  <ele-modal
    :form="true"
    :width="440"
    title="重置密码"
    :loading="loading"
    v-bind="modalProps"
  >
    <el-alert
      type="warning"
      show-icon
      :closable="false"
      style="margin-bottom: 16px"
    >
      重置账号
      <b class="mono">{{ data.userName }}</b> 的密码后，该账号会被强制下线。
    </el-alert>
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="80px"
      @submit.prevent=""
    >
      <el-form-item label="新密码" prop="password">
        <el-input
          v-model="form.password"
          type="password"
          show-password
          placeholder="8-64 位，须包含字母和数字"
          autocomplete="new-password"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <btn-items
        :items="[
          { preset: 'cancel', onClick: () => closeModal() },
          { preset: 'save', title: '确认重置', onClick: () => handleSave() }
        ]"
      />
    </template>
  </ele-modal>
</template>

<script setup>
  import { ref, reactive } from 'vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { resetMerchantUserPassword } from '@/api/merchant/user';

  const props = defineProps({ data: { type: Object, required: true } });
  const { modalProps, closeModal } = useModal();
  const loading = ref(false);
  const formRef = ref(null);
  const [form] = useFormData({ password: '' });
  const rules = reactive({
    password: [
      { required: true, message: '请输入新密码', trigger: 'blur' },
      {
        pattern: /^(?=.*[A-Za-z])(?=.*\d).{8,64}$/,
        message: '8-64 位，须包含字母和数字',
        trigger: 'blur'
      }
    ]
  });

  const handleSave = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) return;
      loading.value = true;
      resetMerchantUserPassword(props.data.userId, form.password)
        .then((msg) => {
          loading.value = false;
          EleMessage.success({ message: msg, plain: true });
          closeModal();
        })
        .catch((e) => {
          loading.value = false;
          EleMessage.error({ message: e.message, plain: true });
        });
    });
  };
</script>
