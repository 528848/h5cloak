<!-- 修改密码弹窗 -->
<template>
  <ele-modal
    :form="true"
    :width="420"
    :title="t('layout.header.password')"
    :loading="loading"
    v-bind="modalProps"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      :label-width="locale === 'en' ? '120px' : '80px'"
      @submit.prevent=""
    >
      <el-form-item :label="t('password.oldPassword')" prop="oldPassword">
        <el-input
          show-password
          type="password"
          v-model="form.oldPassword"
          :placeholder="t('password.oldPasswordPlaceholder')"
        />
      </el-form-item>
      <el-form-item :label="t('password.newPassword')" prop="newPassword">
        <el-input
          show-password
          type="password"
          v-model="form.newPassword"
          :placeholder="t('password.newPasswordPlaceholder')"
        />
      </el-form-item>
      <el-form-item :label="t('password.confirm')" prop="confirmPassword">
        <el-input
          show-password
          type="password"
          v-model="form.confirmPassword"
          :placeholder="t('password.confirmPlaceholder')"
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
  import { ref, reactive, computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { updatePassword } from '@/api/layout';

  const { modalProps, closeModal } = useModal();

  const { t, locale } = useI18n();

  /** 提交状态 */
  const loading = ref(false);

  /** 表单组件 */
  const formRef = ref(null);

  /** 表单数据 */
  const form = reactive({
    oldPassword: '',
    newPassword: '',
    confirmPassword: ''
  });

  /** 表单验证规则 */
  const rules = computed(() => {
    return {
      oldPassword: [
        {
          required: true,
          message: t('password.oldPasswordPlaceholder'),
          type: 'string',
          trigger: 'blur'
        }
      ],
      newPassword: [
        {
          required: true,
          message: t('password.newPasswordPlaceholder'),
          type: 'string',
          trigger: 'blur'
        },
        {
          type: 'string',
          trigger: 'blur',
          pattern: /^[\S]{5,18}$/,
          message: t('password.passwordMessage')
        }
      ],
      confirmPassword: [
        {
          required: true,
          message: t('password.confirmPlaceholder'),
          type: 'string',
          trigger: 'blur'
        },
        {
          type: 'string',
          trigger: 'blur',
          validator: (_rule, value, callback) => {
            if (value && value !== form.newPassword) {
              return callback(new Error(t('password.uniformityMessage')));
            }
            callback();
          }
        }
      ]
    };
  });

  /** 关闭弹窗 */
  const handleCancel = () => {
    closeModal();
  };

  /** 保存修改 */
  const handleSave = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) {
        return;
      }
      loading.value = true;
      updatePassword(form)
        .then(() => {
          loading.value = false;
          EleMessage.success({ message: '修改成功', plain: true });
          handleCancel();
        })
        .catch((e) => {
          loading.value = false;
          EleMessage.error({ message: e.message, plain: true });
        });
    });
  };
</script>
