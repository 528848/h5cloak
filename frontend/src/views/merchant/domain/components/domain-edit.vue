<!-- 域名编辑弹窗 -->
<template>
  <ele-modal
    :form="true"
    :width="640"
    title="添加域名"
    :loading="loading"
    :body-style="{ paddingTop: '8px' }"
    v-bind="modalProps"
  >
    <el-steps :active="activeStep" align-center class="domain-steps">
      <el-step title="输入域名" />
      <el-step title="托管域名" description="修改DNS信息" />
      <el-step title="验证域名" />
    </el-steps>

    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-position="top"
      class="domain-form"
      @submit.prevent=""
    >
      <div v-show="activeStep === 0" class="domain-step-panel">
        <el-form-item label="域名" prop="domain">
          <el-input
            v-model="form.domain"
            clearable
            maxlength="255"
            placeholder="请输入需要配置的域名(不带协议),建议输入二级域名"
          />
        </el-form-item>
      </div>

      <div v-show="activeStep === 1" class="domain-step-panel">
        <template v-if="dnsConfig?.bindMode === 'nameserver'">
          <div class="domain-dns-section">
            <div class="domain-dns-section-title">设置主机记录</div>
            <ele-text type="danger" size="sm" class="domain-dns-warning">
              注意：修改域名 DNS 服务器地址（Nameservers），请确保此域名未在其他场景中使用。
            </ele-text>
            <div class="domain-dns-subtitle">记录类型(VALUE)</div>
            <div
              v-for="item in dnsConfig.entryNameservers"
              :key="item"
              class="domain-dns-ns-row"
            >
              <span class="domain-dns-value">{{ item }}</span>
              <ele-copyable :text="item" />
            </div>
          </div>
        </template>
        <template v-else>
          <el-form-item label="解析类型" prop="verifyType">
            <el-radio-group v-model="form.verifyType">
              <el-radio value="CNAME">CNAME</el-radio>
              <el-radio value="TXT">TXT</el-radio>
            </el-radio-group>
          </el-form-item>
          <div class="domain-dns-section">
            <div class="domain-dns-section-title">{{ dnsGuideTitle }}</div>
            <ele-text type="danger" size="sm" class="domain-dns-warning">
              请将域名解析到斗篷系统入口服务器，流量才会进入判定服务。
            </ele-text>
            <div class="domain-dns-guide">
              <div class="domain-dns-row">
                <span class="domain-dns-label">记录类型</span>
                <span>{{ form.verifyType }}</span>
              </div>
              <div class="domain-dns-row">
                <span class="domain-dns-label">主机记录</span>
                <span>@ 或 www</span>
              </div>
              <div class="domain-dns-row">
                <span class="domain-dns-label">记录值</span>
                <span class="domain-dns-value">{{ displayRecordValue }}</span>
                <ele-copyable
                  v-if="displayRecordValue"
                  :text="displayRecordValue"
                />
              </div>
              <template v-if="form.verifyType === 'CNAME' && dnsConfig?.entryIps?.length">
                <div class="domain-dns-row">
                  <span class="domain-dns-label">入口 IP</span>
                  <span class="domain-dns-value">
                    {{ dnsConfig.entryIps.join(' / ') }}
                  </span>
                </div>
              </template>
            </div>
          </div>
        </template>
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
      </div>

      <div v-show="activeStep === 2" class="domain-step-panel">
        <el-result
          v-if="verifyResult"
          icon="success"
          title="域名验证成功"
          :sub-title="`域名 ${form.domain} 已完成绑定`"
        />
          <template v-else>
            <div class="domain-dns-section">
              <div class="domain-dns-section-title">请确认 DNS 解析已生效</div>
              <div class="domain-dns-guide">
                <div class="domain-dns-row">
                  <span class="domain-dns-label">域名</span>
                  <span>{{ form.domain }}</span>
                </div>
                <template v-if="verifyInfo?.bindMode === 'nameserver'">
                  <div class="domain-dns-row">
                    <span class="domain-dns-label">托管方式</span>
                    <span>Nameserver</span>
                  </div>
                  <div
                    v-for="item in verifyInfo.entryNameservers"
                    :key="item"
                    class="domain-dns-ns-row"
                  >
                    <span class="domain-dns-value">{{ item }}</span>
                    <ele-copyable :text="item" />
                  </div>
                </template>
                <template v-else>
                  <div class="domain-dns-row">
                    <span class="domain-dns-label">解析类型</span>
                    <span>{{ verifyInfo?.verifyType || form.verifyType }}</span>
                  </div>
                  <div class="domain-dns-row">
                    <span class="domain-dns-label">记录值</span>
                    <span class="domain-dns-value">{{ verifyInfo?.verifyValue }}</span>
                    <ele-copyable
                      v-if="verifyInfo?.verifyValue"
                      :text="verifyInfo.verifyValue"
                    />
                  </div>
                </template>
              </div>
            </div>
          <ele-text type="placeholder" style="display: block; margin-top: 12px">
            DNS 修改后可能需要几分钟生效，确认配置无误后点击「验证域名」。
          </ele-text>
        </template>
      </div>
    </el-form>

    <template #footer>
      <btn-items
        :items="footerButtons"
      />
    </template>
  </ele-modal>
</template>

<script setup>
  import { computed, reactive, ref, watch } from 'vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import {
    addMerchantDomain,
    getMerchantDomainDnsConfig,
    verifyMerchantDomain
  } from '@/api/merchant/domain';

  const emit = defineEmits(['done']);

  const { modalProps, closeModal } = useModal();

  const loading = ref(false);
  const activeStep = ref(0);
  const formRef = ref(null);
  const verifyInfo = ref(null);
  const verifyResult = ref(null);
  const dnsConfig = ref(null);

  const [form] = useFormData({
    domain: '',
    verifyType: 'CNAME',
    remark: ''
  });

  const rules = reactive({
    domain: [
      { required: true, message: '请输入域名', trigger: 'blur' },
      {
        pattern: /^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/,
        message: '域名格式不正确',
        trigger: 'blur'
      }
    ],
    verifyType: [
      { required: true, message: '请选择解析类型', trigger: 'change' }
    ]
  });

  const dnsGuideTitle = computed(() => {
    if (form.verifyType === 'TXT') {
      return '请添加 TXT 解析记录';
    }
    return '请添加 CNAME 解析记录';
  });

  const displayRecordValue = computed(() => {
    if (verifyInfo.value?.verifyValue) {
      return verifyInfo.value.verifyValue;
    }
    if (form.verifyType === 'CNAME') {
      return dnsConfig.value?.entryHost || '';
    }
    return '保存后系统自动生成';
  });

  const loadDnsConfig = () => {
    getMerchantDomainDnsConfig()
      .then((data) => {
        dnsConfig.value = data;
      })
      .catch((e) => {
        EleMessage.error({ message: e.message, plain: true });
      });
  };

  watch(activeStep, (step) => {
    if (step === 1 && !dnsConfig.value) {
      loadDnsConfig();
    }
  });

  const footerButtons = computed(() => {
    if (activeStep.value === 0) {
      return [
        { preset: 'cancel', onClick: () => handleCancel() },
        { title: '下一步', onClick: () => handleNext() }
      ];
    }
    if (activeStep.value === 1) {
      return [
        { title: '上一步', onClick: () => handlePrev() },
        { title: '下一步', onClick: () => handleCreate() }
      ];
    }
    if (verifyResult.value) {
      return [{ title: '完成', onClick: () => handleFinish() }];
    }
    return [
      { title: '上一步', onClick: () => handlePrev() },
      { title: '验证域名', onClick: () => handleVerify() }
    ];
  });

  const validateFields = async (fields) => {
    if (!formRef.value) {
      return false;
    }
    try {
      await formRef.value.validateField(fields);
      return true;
    } catch {
      return false;
    }
  };

  const handleNext = async () => {
    const valid = await validateFields(['domain']);
    if (!valid) {
      return;
    }
    activeStep.value = 1;
  };

  const handlePrev = () => {
    if (activeStep.value === 2 && verifyResult.value) {
      return;
    }
    activeStep.value = Math.max(0, activeStep.value - 1);
  };

  const handleCreate = async () => {
    const valid = await validateFields(['verifyType']);
    if (!valid) {
      return;
    }
    if (verifyInfo.value?.domainId) {
      activeStep.value = 2;
      return;
    }
    loading.value = true;
    addMerchantDomain({
      domain: form.domain,
      verifyType: form.verifyType,
      remark: form.remark
    })
      .then((data) => {
        loading.value = false;
        verifyInfo.value = data.data;
        activeStep.value = 2;
        EleMessage.success({ message: data.msg, plain: true });
      })
      .catch((e) => {
        loading.value = false;
        EleMessage.error({ message: e.message, plain: true });
      });
  };

  const handleVerify = () => {
    if (!verifyInfo.value?.domainId) {
      EleMessage.error({ message: '请先完成域名配置', plain: true });
      return;
    }
    loading.value = true;
    verifyMerchantDomain(verifyInfo.value.domainId)
      .then((data) => {
        loading.value = false;
        verifyResult.value = data.data;
        EleMessage.success({ message: data.msg, plain: true });
      })
      .catch((e) => {
        loading.value = false;
        EleMessage.error({ message: e.message, plain: true });
      });
  };

  const handleCancel = () => {
    if (verifyInfo.value) {
      emit('done');
    }
    closeModal();
  };

  const handleFinish = () => {
    emit('done');
    closeModal();
  };
</script>

<style scoped>
  .domain-steps {
    margin-bottom: 24px;
  }

  .domain-steps :deep(.el-step__title) {
    font-size: 14px;
  }

  .domain-steps :deep(.el-step__description) {
    font-size: 12px;
  }

  .domain-form {
    min-height: 220px;
  }

  .domain-step-panel {
    padding: 0 8px;
  }

  .domain-form :deep(.el-form-item__label) {
    font-weight: 500;
    padding-bottom: 4px;
  }

  .domain-dns-alert {
    margin-bottom: 16px;
  }

  .domain-dns-section {
    margin-bottom: 16px;
    padding: 16px;
    border-radius: 8px;
    background: var(--el-fill-color-light);
  }

  .domain-dns-section-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--el-color-primary);
    margin-bottom: 8px;
  }

  .domain-dns-warning {
    display: block;
    margin-bottom: 12px;
    line-height: 1.6;
  }

  .domain-dns-subtitle {
    font-size: 13px;
    color: var(--el-text-color-secondary);
    margin-bottom: 8px;
  }

  .domain-dns-ns-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 0;
  }

  .domain-dns-guide {
    line-height: 1.8;
    font-size: 13px;
  }

  .domain-dns-row {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .domain-dns-label {
    min-width: 72px;
    color: var(--el-text-color-secondary);
  }

  .domain-dns-value {
    color: var(--el-text-color-primary);
    word-break: break-all;
  }
</style>
