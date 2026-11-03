<!-- API 斗篷规则编辑弹窗 -->
<template>
  <ele-modal
    :form="true"
    :width="640"
    title="添加规则"
    :loading="loading"
    class="api-key-edit-modal"
    v-bind="modalProps"
  >
    <el-alert
      title="斗篷仅适用于海外市场，中国及港澳台不适用"
      type="warning"
      :closable="false"
      show-icon
      class="api-key-alert"
    />
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="100px"
      class="api-key-form"
      @submit.prevent=""
    >
      <el-form-item label="名称" prop="name">
        <el-input
          v-model="form.name"
          maxlength="50"
          show-word-limit
          placeholder="请输入名称"
        />
      </el-form-item>

      <el-form-item label="Api token">
        <el-input
          v-model="form.apiTokenPreview"
          disabled
          placeholder="创建成功后生效"
        />
        <div class="field-tip">预览值仅供参考，创建成功后以系统生成的 Token 为准</div>
      </el-form-item>

      <el-form-item label="模式" prop="rules.mode">
        <div class="inline-field">
          <el-radio-group v-model="form.rules.mode">
            <el-radio value="cloak">斗篷</el-radio>
            <el-radio value="green">绿色</el-radio>
            <el-radio value="full_open">全开</el-radio>
            <el-radio value="review">审核</el-radio>
          </el-radio-group>
          <span class="field-tip field-tip--warning">
            注：黑名单 IP 所有模式下都会访问审核链接
          </span>
        </div>
      </el-form-item>

      <el-form-item
        v-if="form.rules.mode === 'cloak'"
        label="投放地区"
        prop="targetRegionPaths"
      >
        <country-regions-select v-model="form.targetRegionPaths" />
      </el-form-item>

      <el-form-item label="蜘蛛白名单">
        <el-select
          v-model="form.rules.spiderWhitelist"
          clearable
          placeholder="请选择蜘蛛白名单"
          class="ele-fluid"
        >
          <el-option label="Googlebot" value="googlebot" />
          <el-option label="Bingbot" value="bingbot" />
          <el-option
            label="Facebook External Hit"
            value="facebookexternalhit"
          />
          <el-option label="TikTok Bot" value="tiktokbot" />
        </el-select>
      </el-form-item>

      <el-form-item label="屏蔽PC">
        <el-radio-group v-model="form.rules.blockPc">
          <el-radio :value="false">未开启</el-radio>
          <el-radio :value="true">开启</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="屏蔽代理">
        <el-radio-group v-model="form.rules.blockProxy">
          <el-radio :value="false">未开启</el-radio>
          <el-radio :value="true">开启</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="备注">
        <el-input
          v-model="form.remark"
          type="textarea"
          :rows="3"
          maxlength="200"
          show-word-limit
          placeholder="请输入"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <btn-items
        :items="[
          { preset: 'cancel', onClick: () => handleCancel() },
          { title: '确认', type: 'primary', onClick: () => handleSave() }
        ]"
      />
    </template>
  </ele-modal>
</template>

<script setup>
  import { onMounted, reactive, ref, watch } from 'vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { addApiKey } from '@/api/merchant/api-key';
  import CountryRegionsSelect from '../../cloak/components/country-regions-select.vue';

  const emit = defineEmits(['done']);

  const { modalProps, closeModal } = useModal();

  const loading = ref(false);
  const formRef = ref(null);

  const defaultRules = () => ({
    mode: 'cloak',
    spiderWhitelist: '',
    blockPc: false,
    blockProxy: false
  });

  const [form] = useFormData({
    name: '',
    apiTokenPreview: '',
    remark: '',
    targetRegionPaths: [],
    rules: defaultRules()
  });

  const rules = reactive({
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    'rules.mode': [{ required: true, message: '请选择模式', trigger: 'change' }],
    targetRegionPaths: [
      {
        validator: (_rule, value, callback) => {
          if (form.rules.mode === 'cloak' && (!Array.isArray(value) || value.length === 0)) {
            callback(new Error('请选择投放地区'));
            return;
          }
          callback();
        },
        trigger: 'change'
      }
    ]
  });

  watch(
    () => form.rules.mode,
    (mode) => {
      if (mode !== 'cloak') {
        form.targetRegionPaths = [];
        formRef.value?.clearValidate?.('targetRegionPaths');
      }
    }
  );

  const generateTokenPreview = () => {
    const bytes = new Uint8Array(16);
    crypto.getRandomValues(bytes);
    return Array.from(bytes, (item) => item.toString(16).padStart(2, '0')).join(
      ''
    );
  };

  const handleCancel = () => {
    closeModal();
  };

  const handleSave = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) {
        return;
      }
      loading.value = true;
      const payloadRules = { ...form.rules };
      if (form.rules.mode === 'cloak') {
        payloadRules.targetRegions = form.targetRegionPaths.map(
          (path) => path[path.length - 1]
        );
      } else {
        delete payloadRules.targetRegions;
      }
      addApiKey({
        name: form.name,
        remark: form.remark,
        rules: payloadRules
      })
        .then((res) => {
          loading.value = false;
          const secret = res.data?.secret;
          handleCancel();
          if (secret) {
            emit('done', secret);
          } else {
            emit('done');
            EleMessage.success({ message: res.msg, plain: true });
          }
        })
        .catch((e) => {
          loading.value = false;
          EleMessage.error({ message: e.message, plain: true });
        });
    });
  };

  onMounted(() => {
    form.apiTokenPreview = generateTokenPreview();
  });
</script>

<style scoped>
  .api-key-alert {
    margin-bottom: 16px;
  }

  .api-key-form :deep(.el-form-item) {
    margin-bottom: 18px;
  }

  .inline-field {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px 12px;
    width: 100%;
  }

  .field-tip {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1.4;
  }

  .field-tip--warning {
    color: var(--el-color-warning);
  }
</style>
