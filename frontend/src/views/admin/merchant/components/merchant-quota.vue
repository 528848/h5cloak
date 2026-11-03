<!-- 额度与功能开关 -->
<template>
  <ele-modal
    :form="true"
    :width="640"
    title="额度与功能"
    :loading="loading"
    v-bind="modalProps"
  >
    <el-form ref="formRef" :model="form" label-width="110px" @submit.prevent="">
      <el-divider content-position="left">使用额度</el-divider>
      <el-row :gutter="16">
        <el-col :sm="12" :xs="24">
          <el-form-item label="判定次数/月">
            <el-input-number
              v-model="form.quota.decisionQuota"
              :min="0"
              :step="10000"
              controls-position="right"
              class="ele-fluid"
            />
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="站点数">
            <el-input-number
              v-model="form.quota.siteQuota"
              :min="1"
              :max="1000"
              controls-position="right"
              class="ele-fluid"
            />
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="域名数">
            <el-input-number
              v-model="form.quota.domainQuota"
              :min="1"
              :max="10000"
              controls-position="right"
              class="ele-fluid"
            />
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="分销码数">
            <el-input-number
              v-model="form.quota.codeQuota"
              :min="0"
              :max="1000000"
              controls-position="right"
              class="ele-fluid"
            />
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="子账号数">
            <el-input-number
              v-model="form.quota.userQuota"
              :min="1"
              :max="500"
              controls-position="right"
              class="ele-fluid"
            />
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="日志保留天数">
            <el-input-number
              v-model="form.quota.logRetentionDays"
              :min="7"
              :max="365"
              controls-position="right"
              class="ele-fluid"
            />
          </el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="达到额度后">
            <el-radio-group v-model="form.quota.quotaPolicy">
              <el-radio
                v-for="o in quotaPolicyOptions"
                :key="o.value"
                :value="o.value"
                >{{ o.label }}</el-radio
              >
            </el-radio-group>
          </el-form-item>
        </el-col>
      </el-row>

      <el-divider content-position="left">功能开关</el-divider>
      <div class="feature-list">
        <div v-for="f in featureOptions" :key="f.key" class="feature-row">
          <div class="feature-text">
            <span>{{ f.label }}</span>
            <em>{{ f.desc }}</em>
          </div>
          <el-switch v-model="form.features[f.key]" />
        </div>
      </div>
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
  import { ref } from 'vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { updateMerchantQuota } from '@/api/admin/merchant';
  import { quotaPolicyOptions, featureOptions } from '../constants';

  const props = defineProps({ data: { type: Object, required: true } });
  const emit = defineEmits(['done']);
  const { modalProps, closeModal } = useModal();
  const loading = ref(false);
  const formRef = ref(null);

  const [form] = useFormData({
    quota: { ...props.data.quota },
    features: { ...props.data.features }
  });

  const handleSave = () => {
    loading.value = true;
    updateMerchantQuota(props.data.merchantId, form.quota, form.features)
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
  };
</script>

<style scoped>
  .feature-list {
    display: flex;
    flex-direction: column;
  }
  .feature-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 0;
    border-bottom: 1px dashed var(--el-border-color-lighter);
  }
  .feature-row:last-child {
    border-bottom: none;
  }
  .feature-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .feature-text > span {
    font-size: 13px;
    color: var(--el-text-color-primary);
  }
  .feature-text > em {
    font-style: normal;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
</style>
