<template>
  <ele-modal
    :form="true"
    :width="560"
    :title="isEdit ? '编辑账号' : '新增账号'"
    :loading="loading"
    v-bind="modalProps"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="96px"
      @submit.prevent=""
    >
      <el-form-item label="登录账号" prop="userName">
        <el-input
          v-model.trim="form.userName"
          :disabled="isEdit"
          clearable
          placeholder="3-32 位字母数字"
          autocomplete="off"
        />
      </el-form-item>
      <el-form-item v-if="!isEdit" label="初始密码" prop="password">
        <el-input
          v-model="form.password"
          type="password"
          show-password
          placeholder="8-64 位，须包含字母和数字"
          autocomplete="new-password"
        />
      </el-form-item>
      <el-form-item label="昵称" prop="nickName">
        <el-input
          v-model.trim="form.nickName"
          clearable
          maxlength="30"
          placeholder="请输入昵称"
        />
      </el-form-item>
      <el-form-item label="角色" prop="roleKey">
        <el-radio-group v-model="form.roleKey" class="role-group">
          <el-radio v-for="o in roleOptions" :key="o.value" :value="o.value">
            {{ o.label }}<em class="role-desc">{{ o.desc }}</em>
          </el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="邮箱" prop="email">
        <el-input
          v-model.trim="form.email"
          clearable
          placeholder="请输入邮箱"
        />
      </el-form-item>
      <el-form-item label="手机号">
        <el-input
          v-model.trim="form.phonenumber"
          clearable
          placeholder="请输入手机号"
        />
      </el-form-item>
      <el-form-item label="备注">
        <el-input
          v-model="form.remark"
          type="textarea"
          :rows="2"
          maxlength="200"
          show-word-limit
          placeholder="请输入备注"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <btn-items
        :items="[
          { preset: 'cancel', onClick: () => closeModal() },
          { preset: 'save', onClick: () => handleSave() }
        ]"
      />
    </template>
  </ele-modal>
</template>

<script setup>
  import { ref, reactive, computed } from 'vue';
  import { EleMessage, useModal, emailReg } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { addMerchantUser, updateMerchantUser } from '@/api/merchant/user';
  import { roleOptions } from '../constants';

  const props = defineProps({ data: Object });
  const emit = defineEmits(['done']);
  const { modalProps, closeModal } = useModal();
  const isEdit = computed(() => !!props.data?.userId);
  const loading = ref(false);
  const formRef = ref(null);

  const [form] = useFormData({
    userName: props.data?.userName ?? '',
    password: '',
    nickName: props.data?.nickName ?? '',
    roleKey: props.data?.roleKeys?.[0] ?? 'merchant_operator',
    email: props.data?.email ?? '',
    phonenumber: props.data?.phonenumber ?? '',
    remark: props.data?.remark ?? ''
  });

  const rules = reactive({
    userName: [
      { required: true, message: '请输入登录账号', trigger: 'blur' },
      {
        pattern: /^[A-Za-z0-9_.@-]{3,32}$/,
        message: '3-32 位，仅字母、数字、_ . - @',
        trigger: 'blur'
      }
    ],
    password: [
      { required: true, message: '请输入初始密码', trigger: 'blur' },
      {
        pattern: /^(?=.*[A-Za-z])(?=.*\d).{8,64}$/,
        message: '8-64 位，须包含字母和数字',
        trigger: 'blur'
      }
    ],
    roleKey: [{ required: true, message: '请选择角色', trigger: 'change' }],
    email: [{ pattern: emailReg, message: '邮箱格式不正确', trigger: 'blur' }]
  });

  const handleSave = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) return;
      loading.value = true;
      const req = isEdit.value
        ? updateMerchantUser(props.data.userId, form)
        : addMerchantUser(form).then((r) => r.msg);
      req
        .then((msg) => {
          loading.value = false;
          EleMessage.success({ message: msg, plain: true });
          emit('done');
          closeModal();
        })
        .catch((e) => {
          loading.value = false;
          EleMessage.error({ message: e.message, plain: true });
        });
    });
  };
</script>

<style scoped>
  .role-group {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
  .role-desc {
    font-style: normal;
    margin-left: 8px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
</style>
