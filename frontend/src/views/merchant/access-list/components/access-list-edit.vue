<!-- 黑白名单编辑弹窗 -->
<template>
  <ele-modal
    :form="true"
    :width="520"
    :title="isUpdate ? '修改名单' : '添加名单'"
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
      <el-form-item label="名单类型" prop="listType">
        <el-radio-group v-model="form.listType">
          <el-radio value="black">黑名单</el-radio>
          <el-radio value="white">白名单</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="目标类型" prop="targetType">
        <el-select
          v-model="form.targetType"
          class="ele-fluid"
          placeholder="请选择"
        >
          <el-option label="IP" value="ip" />
          <el-option label="User-Agent" value="ua" />
          <el-option label="Referer" value="referer" />
        </el-select>
      </el-form-item>
      <el-form-item label="匹配值" prop="targetValue">
        <el-input
          v-model="form.targetValue"
          maxlength="255"
          show-word-limit
          :placeholder="targetValuePlaceholder"
        />
      </el-form-item>
      <el-form-item label="作用范围" prop="scope">
        <div class="inline-field">
          <el-radio-group v-model="form.scope">
            <el-radio value="merchant">商户全局</el-radio>
            <el-radio value="cloak">
              <span class="scope-radio-label">
                斗篷专属
                <el-tooltip
                  content="仅对所选斗篷生效，不影响商户下其他斗篷"
                  placement="top"
                >
                  <el-icon class="scope-tip-icon"><QuestionFilled /></el-icon>
                </el-tooltip>
              </span>
            </el-radio>
          </el-radio-group>
        </div>
      </el-form-item>
      <el-form-item
        v-if="form.scope === 'cloak'"
        label="关联斗篷"
        prop="cloakId"
      >
        <el-select
          v-model="form.cloakId"
          class="ele-fluid"
          placeholder="请选择斗篷"
          :loading="cloakLoading"
        >
          <el-option
            v-for="item in cloakOptions"
            :key="item.cloakId"
            :label="item.cloakName"
            :value="item.cloakId"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="状态" prop="status">
        <el-radio-group v-model="form.status">
          <el-radio value="enabled">启用</el-radio>
          <el-radio value="disabled">停用</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="备注">
        <el-input
          v-model="form.remark"
          type="textarea"
          :rows="3"
          maxlength="255"
          show-word-limit
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
  import { onMounted, ref, reactive, watch, computed } from 'vue';
  import { QuestionFilled } from '@element-plus/icons-vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { pageMerchantCloaks } from '@/api/merchant/cloak';
  import { addAccessList, updateAccessList } from '@/api/merchant/access-list';

  const props = defineProps({
    data: Object
  });

  const emit = defineEmits(['done']);

  const { modalProps, closeModal } = useModal();

  const loading = ref(false);
  const cloakLoading = ref(false);
  const cloakOptions = ref([]);
  const formRef = ref(null);
  const isUpdate = ref(false);

  const [form, resetFields] = useFormData({
    listId: void 0,
    listType: 'black',
    targetType: 'ip',
    targetValue: '',
    scope: 'merchant',
    cloakId: void 0,
    status: 'enabled',
    remark: ''
  });

  const isValidIpv4 = (value) => {
    const pattern =
      /^(?:(?:25[0-5]|2[0-4]\d|1\d{2}|[1-9]?\d)\.){3}(?:25[0-5]|2[0-4]\d|1\d{2}|[1-9]?\d)$/;
    return pattern.test(String(value).trim());
  };

  const targetValuePlaceholder = computed(() => {
    if (form.targetType === 'ip') {
      return '请输入 IP 地址，如 192.168.1.1';
    }
    if (form.targetType === 'ua') {
      return '请输入 User-Agent 关键字';
    }
    if (form.targetType === 'referer') {
      return '请输入 Referer 域名或关键字';
    }
    return '请输入匹配值';
  });

  const rules = reactive({
    listType: [
      { required: true, message: '请选择名单类型', trigger: 'change' }
    ],
    targetType: [
      { required: true, message: '请选择目标类型', trigger: 'change' }
    ],
    targetValue: [
      { required: true, message: '请输入匹配值', trigger: 'blur' },
      {
        validator: (_rule, value, callback) => {
          if (!value) {
            callback();
            return;
          }
          if (form.targetType === 'ip' && !isValidIpv4(value)) {
            callback(new Error('IP 格式不正确'));
            return;
          }
          callback();
        },
        trigger: ['blur', 'change']
      }
    ],
    scope: [{ required: true, message: '请选择作用范围', trigger: 'change' }],
    cloakId: [
      {
        validator: (_rule, value, callback) => {
          if (form.scope === 'cloak' && !value) {
            callback(new Error('请选择关联斗篷'));
            return;
          }
          callback();
        },
        trigger: 'change'
      }
    ],
    status: [{ required: true, message: '请选择状态', trigger: 'change' }]
  });

  watch(
    () => form.scope,
    (value) => {
      if (value === 'merchant') {
        form.cloakId = void 0;
      }
    }
  );

  watch(
    () => form.targetType,
    () => {
      formRef.value?.validateField?.('targetValue').catch(() => {});
    }
  );

  const loadCloaks = () => {
    cloakLoading.value = true;
    pageMerchantCloaks({ pageNum: 1, pageSize: 200 })
      .then((data) => {
        cloakLoading.value = false;
        cloakOptions.value = data.rows || [];
      })
      .catch((e) => {
        cloakLoading.value = false;
        EleMessage.error({ message: e.message, plain: true });
      });
  };

  const assignForm = () => {
    resetFields();
    isUpdate.value = !!props.data?.listId;
    if (!isUpdate.value) {
      return;
    }
    Object.assign(form, {
      listId: props.data.listId,
      listType: props.data.listType,
      targetType: props.data.targetType,
      targetValue: props.data.targetValue,
      scope: props.data.cloakId ? 'cloak' : props.data.scope || 'merchant',
      cloakId: props.data.cloakId || void 0,
      status: props.data.status || 'enabled',
      remark: props.data.remark || ''
    });
  };

  const buildPayload = () => ({
    listType: form.listType,
    targetType: form.targetType,
    targetValue: form.targetValue,
    scope: form.scope,
    cloakId: form.scope === 'cloak' ? form.cloakId : null,
    status: form.status,
    remark: form.remark || null
  });

  const handleCancel = () => {
    closeModal();
  };

  const handleSave = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) {
        return;
      }
      loading.value = true;
      const request = isUpdate.value
        ? updateAccessList(form.listId, buildPayload())
        : addAccessList(buildPayload());
      request
        .then((res) => {
          loading.value = false;
          EleMessage.success({ message: res.msg || '保存成功', plain: true });
          emit('done');
          handleCancel();
        })
        .catch((e) => {
          loading.value = false;
          EleMessage.error({ message: e.message, plain: true });
        });
    });
  };

  onMounted(() => {
    assignForm();
    loadCloaks();
  });
</script>

<style scoped>
  .inline-field {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px 12px;
    width: 100%;
  }

  .scope-radio-label {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }

  .scope-tip-icon {
    font-size: 14px;
    color: var(--el-text-color-secondary);
    cursor: help;
    vertical-align: -2px;
  }

  .scope-tip-icon:hover {
    color: var(--el-color-primary);
  }
</style>
