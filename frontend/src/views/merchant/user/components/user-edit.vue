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
        <el-select
          v-model="form.roleKey"
          placeholder="请选择角色"
          class="ele-fluid"
          filterable
        >
          <el-option
            v-for="o in roles"
            :key="o.roleKey"
            :label="o.name"
            :value="o.roleKey"
          >
            <div class="role-option">
              <span>{{ o.name }}</span>
              <span class="role-option-desc">{{ roleDesc(o) }}</span>
            </div>
          </el-option>
        </el-select>
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
  import {
    addMerchantUser,
    listMerchantRoles,
    updateMerchantUser
  } from '@/api/merchant/user';
  import { roleDesc, roleOptions } from '../constants';

  const props = defineProps({ data: Object });
  const emit = defineEmits(['done']);
  const { modalProps, closeModal } = useModal();
  const isEdit = computed(() => !!props.data?.userId);
  const loading = ref(false);
  const formRef = ref(null);
  const roles = ref([]);

  listMerchantRoles()
    .then((rows) => {
      const list = rows || [];
      const key = props.data?.roleKeys?.[0];
      if (key && !list.some((r) => r.roleKey === key)) {
        list.push({
          roleKey: key,
          name: props.data?.roleNames?.[0] || key,
          remark: '该角色已停用'
        });
      }
      roles.value = list;
    })
    .catch((e) => {
      roles.value = roleOptions.map((o) => ({
        roleKey: o.value,
        name: o.label,
        remark: o.desc,
        builtin: true
      }));
      EleMessage.error({ message: e.message, plain: true });
    });

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
  .role-option {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    width: 100%;
  }
  .role-option-desc {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
</style>
