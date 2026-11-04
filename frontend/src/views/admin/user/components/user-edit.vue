<template>
  <ele-modal
    :form="true"
    :width="680"
    :title="isUpdate ? '修改管理员' : '添加管理员'"
    :loading="loading"
    v-bind="modalProps"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="88px"
      @submit.prevent=""
    >
      <el-row :gutter="16">
        <el-col :sm="12" :xs="24">
          <el-form-item label="用户昵称" prop="nickName">
            <el-input
              clearable
              :maxlength="30"
              v-model="form.nickName"
              placeholder="请输入用户昵称"
            />
          </el-form-item>
          <el-form-item label="用户名称" prop="userName">
            <el-input
              clearable
              :maxlength="64"
              :disabled="isUpdate"
              v-model="form.userName"
              placeholder="请输入登录账号"
              autocomplete="off"
            />
          </el-form-item>
          <el-form-item v-if="!isUpdate" label="用户密码" prop="password">
            <el-input
              show-password
              type="password"
              :maxlength="64"
              v-model="form.password"
              placeholder="至少 6 位"
              autocomplete="new-password"
            />
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="角色" prop="roleIds">
            <el-select
              v-model="form.roleIds"
              multiple
              collapse-tags
              collapse-tags-tooltip
              placeholder="请选择角色"
              class="ele-fluid"
            >
              <el-option
                v-for="r in roles"
                :key="r.roleId"
                :label="r.name"
                :value="r.roleId"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="状态" prop="status">
            <el-radio-group v-model="form.status">
              <el-radio value="enabled" label="启用" />
              <el-radio value="disabled" label="停用" />
            </el-radio-group>
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="备注">
        <el-input
          type="textarea"
          :rows="3"
          :maxlength="200"
          v-model="form.remark"
          placeholder="请输入内容"
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
  import { ref, reactive } from 'vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { listRoles, saveAccount } from '@/api/admin/platform';

  const props = defineProps({
    data: Object
  });

  const emit = defineEmits(['done']);

  const { modalProps, closeModal } = useModal();

  const isUpdate = ref(false);
  const loading = ref(false);
  const roles = ref([]);
  const formRef = ref(null);

  const [form, _reset, assignFields] = useFormData({
    userId: void 0,
    userName: '',
    password: '',
    nickName: '',
    roleIds: [],
    status: 'enabled',
    remark: ''
  });

  const rules = reactive({
    userName: [{ required: true, message: '请输入用户名称', trigger: 'blur' }],
    password: [{ required: true, message: '请输入用户密码', trigger: 'blur' }],
    nickName: [{ required: true, message: '请输入用户昵称', trigger: 'blur' }],
    roleIds: [
      {
        required: true,
        type: 'array',
        message: '请选择角色',
        trigger: 'change'
      }
    ],
    status: [{ required: true, message: '请选择状态', trigger: 'change' }]
  });

  const handleSave = () => {
    formRef.value?.validate?.(async (valid) => {
      if (!valid) {
        return;
      }
      loading.value = true;
      try {
        await saveAccount(form.userId || 0, { ...form });
        EleMessage.success({ message: '保存成功', plain: true });
        closeModal();
        emit('done');
      } catch (e) {
        EleMessage.error({ message: e.message, plain: true });
      } finally {
        loading.value = false;
      }
    });
  };

  listRoles()
    .then((rows) => {
      roles.value = rows || [];
    })
    .catch((e) => EleMessage.error({ message: e.message, plain: true }));

  if (props.data) {
    isUpdate.value = true;
    assignFields({
      ...props.data,
      roleIds: props.data.roleIds ? [...props.data.roleIds] : []
    });
  }
</script>
