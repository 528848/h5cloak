<!-- 重置商户管理员密码 -->
<template>
  <ele-modal
    :form="true"
    :width="440"
    title="重置管理员密码"
    :loading="loading"
    v-bind="modalProps"
  >
    <el-alert
      type="warning"
      show-icon
      :closable="false"
      style="margin-bottom: 16px"
    >
      将重置商户「{{ data.merchantName }}」管理员
      <b class="mono">{{ data.adminUsername || '-' }}</b>
      的密码，其所有登录会话会被强制下线。
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
      <el-form-item label="确认密码" prop="confirm">
        <el-input
          v-model="form.confirm"
          type="password"
          show-password
          placeholder="再次输入新密码"
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
  import { ElMessageBox } from 'element-plus';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { resetMerchantAdminPassword } from '@/api/admin/merchant';

  const props = defineProps({ data: { type: Object, required: true } });
  const { modalProps, closeModal } = useModal();
  const loading = ref(false);
  const formRef = ref(null);
  const [form] = useFormData({ password: '', confirm: '' });

  const rules = reactive({
    password: [
      { required: true, message: '请输入新密码', trigger: 'blur' },
      {
        pattern: /^(?=.*[A-Za-z])(?=.*\d).{8,64}$/,
        message: '8-64 位，须包含字母和数字',
        trigger: 'blur'
      }
    ],
    confirm: [
      { required: true, message: '请再次输入新密码', trigger: 'blur' },
      {
        validator: (_, v, cb) =>
          v === form.password ? cb() : cb(new Error('两次输入的密码不一致')),
        trigger: 'blur'
      }
    ]
  });

  const handleSave = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) return;
      loading.value = true;
      resetMerchantAdminPassword(props.data.merchantId, form.password)
        .then((res) => {
          loading.value = false;
          closeModal();
          ElMessageBox.alert(
            `管理员账号：${res.data.adminUsername}\n新密码：${form.password}\n请通过安全渠道告知商户。`,
            '密码已重置',
            { confirmButtonText: '知道了', type: 'success' }
          );
        })
        .catch((e) => {
          loading.value = false;
          EleMessage.error({ message: e.message, plain: true });
        });
    });
  };
</script>
