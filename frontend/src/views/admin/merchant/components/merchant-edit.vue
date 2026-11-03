<!-- 商户编辑弹窗 -->
<template>
  <ele-modal
    :form="true"
    :width="640"
    title="添加商户"
    :loading="loading"
    v-bind="modalProps"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="100px"
      @submit.prevent=""
    >
      <el-row :gutter="16">
        <el-col :sm="12" :xs="24">
          <el-form-item label="商户名称" prop="name">
            <el-input
              clearable
              v-model="form.name"
              placeholder="请输入商户名称"
            />
          </el-form-item>
          <el-form-item label="联系人" prop="contactName">
            <el-input
              clearable
              v-model="form.contactName"
              placeholder="请输入联系人"
            />
          </el-form-item>
          <el-form-item label="联系邮箱" prop="contactEmail">
            <el-input
              clearable
              v-model="form.contactEmail"
              placeholder="请输入联系邮箱"
            />
          </el-form-item>
          <el-form-item label="联系电话" prop="contactPhone">
            <el-input
              clearable
              v-model="form.contactPhone"
              placeholder="请输入联系电话"
            />
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="管理员账号" prop="adminUsername">
            <el-input
              clearable
              v-model="form.adminUsername"
              placeholder="请输入管理员登录账号"
            />
          </el-form-item>
          <el-form-item label="管理员密码" prop="adminPassword">
            <el-input
              show-password
              type="password"
              v-model="form.adminPassword"
              placeholder="请输入初始密码"
              autocomplete="new-password"
            />
          </el-form-item>
          <el-form-item label="域名配额" prop="domainQuota">
            <el-input-number
              :min="0"
              v-model="form.domainQuota"
              controls-position="right"
              class="ele-fluid"
            />
          </el-form-item>
          <el-form-item label="斗篷配额" prop="cloakQuota">
            <el-input-number
              :min="0"
              v-model="form.cloakQuota"
              controls-position="right"
              class="ele-fluid"
            />
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="备注">
        <el-input
          :rows="3"
          type="textarea"
          v-model="form.remark"
          placeholder="请输入备注"
        />
      </el-form-item>
    </el-form>
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
  import { ref, reactive } from 'vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage, useModal, emailReg } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { addMerchant } from '@/api/admin/merchant';

  const emit = defineEmits(['done']);

  const { modalProps, closeModal } = useModal();

  /** 提交状态 */
  const loading = ref(false);

  /** 表单实例 */
  const formRef = ref(null);

  /** 表单数据 */
  const [form] = useFormData({
    name: '',
    contactName: '',
    contactEmail: '',
    contactPhone: '',
    adminUsername: '',
    adminPassword: '',
    domainQuota: 10,
    cloakQuota: 50,
    apiQuota: 10000,
    status: 'enabled',
    remark: ''
  });

  /** 表单验证规则 */
  const rules = reactive({
    name: [
      {
        required: true,
        message: '请输入商户名称',
        type: 'string',
        trigger: 'blur'
      }
    ],
    contactEmail: [
      {
        pattern: emailReg,
        message: '邮箱格式不正确',
        type: 'string',
        trigger: 'blur'
      }
    ],
    adminUsername: [
      {
        required: true,
        message: '请输入管理员账号',
        type: 'string',
        trigger: 'blur'
      }
    ],
    adminPassword: [
      {
        required: true,
        message: '请输入管理员密码',
        type: 'string',
        trigger: 'blur'
      },
      {
        type: 'string',
        trigger: 'blur',
        pattern: /^[\S]{6,18}$/,
        message: '密码必须为6-18位非空白字符'
      }
    ]
  });

  /** 关闭弹窗 */
  const handleCancel = () => {
    closeModal();
  };

  /** 保存 */
  const handleSave = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) {
        return;
      }
      loading.value = true;
      addMerchant(form)
        .then((res) => {
          loading.value = false;
          const loginUrl = res.data?.loginUrl;
          const adminUsername = res.data?.adminUsername || form.adminUsername;
          if (loginUrl && adminUsername) {
            ElMessageBox.alert(
              `登录地址：${loginUrl}\n登录账号：${adminUsername}\n初始密码：创建时设置的密码`,
              '商户创建成功',
              { confirmButtonText: '知道了', type: 'success' }
            );
          } else {
            EleMessage.success({ message: res.msg, plain: true });
          }
          emit('done');
          handleCancel();
        })
        .catch((e) => {
          loading.value = false;
          EleMessage.error({ message: e.message, plain: true });
        });
    });
  };
</script>
