<!-- 斗篷编辑弹窗 -->
<template>
  <ele-modal
    :form="true"
    :width="720"
    :height="660"
    position="top"
    :top="48"
    :title="modalTitle"
    :loading="loading"
    class="cloak-edit-modal"
    :body-style="modalBodyStyle"
    v-bind="modalProps"
  >
    <el-alert
      title="斗篷仅适用于海外市场，中国及港澳台不适用"
      type="warning"
      :closable="false"
      show-icon
      class="cloak-alert"
    />
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="100px"
      class="cloak-form"
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

      <el-form-item label="关联域名" prop="domainId">
        <el-select
          v-model="form.domainId"
          placeholder="请选择已绑定域名"
          class="ele-fluid"
          :loading="domainLoading"
        >
          <el-option
            v-for="item in domainOptions"
            :key="item.domainId"
            :label="item.domain"
            :value="item.domainId"
          />
        </el-select>
      </el-form-item>

      <el-form-item label="真实链接" required>
        <div class="link-list">
          <div
            v-for="(item, index) in form.rules.realUrls"
            :key="linkRowKeys[index]"
            class="link-list-item"
          >
            <el-form-item
              :prop="'rules.realUrls.' + index"
              :rules="realUrlRules"
              class="link-input-item"
            >
              <el-input
                v-model="form.rules.realUrls[index]"
                maxlength="1000"
                show-word-limit
                placeholder="请输入真实链接"
              />
            </el-form-item>
            <button
              v-if="index === 0"
              type="button"
              class="link-action-btn"
              @click="addRealUrl"
            >
              <el-icon><Plus /></el-icon>
            </button>
            <button
              v-else
              type="button"
              class="link-action-btn"
              @click="removeRealUrl(index)"
            >
              <el-icon><Minus /></el-icon>
            </button>
          </div>
        </div>
      </el-form-item>

      <el-form-item label="审核页模式" prop="rules.reviewPageMode">
        <el-radio-group v-model="form.rules.reviewPageMode">
          <el-radio value="external">跳转外链</el-radio>
          <el-radio value="local_template">本地模版</el-radio>
        </el-radio-group>
      </el-form-item>

      <el-form-item label="审核链接" prop="whiteUrl">
        <div class="review-link-row">
          <el-select
            v-model="form.rules.reviewQuickTag"
            clearable
            placeholder="请选择快速审核"
            class="review-quick-select"
            @change="handleQuickReview"
          >
            <el-option label="Google" value="google" />
            <el-option label="Facebook" value="facebook" />
            <el-option label="TikTok" value="tiktok" />
            <el-option label="通用安全页" value="generic" />
          </el-select>
          <el-input
            v-model="form.whiteUrl"
            maxlength="1000"
            show-word-limit
            placeholder="请输入审核链接"
          />
        </div>
      </el-form-item>

      <el-form-item label="跳转短链" prop="rules.shortLinkType">
        <div class="inline-field">
          <el-radio-group v-model="form.rules.shortLinkType">
            <el-radio value="system">系统链接</el-radio>
            <el-radio value="custom">自定义链接</el-radio>
          </el-radio-group>
          <span
            v-if="form.rules.shortLinkType === 'system'"
            class="field-tip"
          >
            {{
              form.rules.shortLinkUrl
                ? `当前短链：${form.rules.shortLinkUrl}`
                : '保存后系统自动生成短链'
            }}
          </span>
        </div>
        <el-input
          v-if="form.rules.shortLinkType === 'custom'"
          v-model="form.rules.shortLinkUrl"
          maxlength="1000"
          show-word-limit
          placeholder="请输入自定义短链"
          class="sub-input"
        />
      </el-form-item>

      <el-form-item label="允许访问来源" prop="rules.allowedSources">
        <el-checkbox-group v-model="form.rules.allowedSources">
          <el-checkbox value="facebook">facebook</el-checkbox>
          <el-checkbox value="google">google</el-checkbox>
          <el-checkbox value="tiktok">tiktok</el-checkbox>
          <el-checkbox value="kwai">kwai</el-checkbox>
          <el-checkbox value="all">不限制</el-checkbox>
        </el-checkbox-group>
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

      <el-form-item label="投放地区" prop="targetRegionPaths">
        <country-regions-select v-model="form.targetRegionPaths" />
      </el-form-item>

      <div class="advanced-section">
        <button
          type="button"
          class="advanced-toggle"
          :class="{ 'is-expanded': advancedOpen }"
          @click="advancedOpen = !advancedOpen"
        >
          <span>高级设置</span>
          <el-icon class="advanced-toggle-icon"><ArrowDown /></el-icon>
        </button>
        <el-collapse-transition>
          <div v-show="advancedOpen" class="advanced-panel">
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

            <el-form-item label="设备白名单">
              <el-select
                v-model="form.rules.deviceWhitelist"
                clearable
                placeholder="请选择设备白名单"
                class="ele-fluid"
              >
                <el-option label="仅移动端" value="mobile" />
                <el-option label="仅平板" value="tablet" />
                <el-option label="仅 PC" value="pc" />
                <el-option label="移动端 + 平板" value="mobile,tablet" />
              </el-select>
            </el-form-item>

            <el-form-item label="访问限制" class="switch-group-item">
              <div class="switch-group">
                <div class="switch-row">
                  <span>禁止空语言访问</span>
                  <el-switch v-model="form.rules.blockEmptyLanguage" />
                </div>
                <div class="switch-row">
                  <span>屏蔽 PC</span>
                  <el-switch v-model="form.rules.blockPc" />
                </div>
                <div class="switch-row">
                  <span>屏蔽代理</span>
                  <el-switch v-model="form.rules.blockProxy" />
                </div>
              </div>
            </el-form-item>

            <el-form-item label="中间页">
              <el-radio-group v-model="form.rules.intermediatePage">
                <el-radio value="none">不使用</el-radio>
                <el-radio value="system_template">系统模版</el-radio>
                <el-radio value="custom_template">自定义模版</el-radio>
              </el-radio-group>
            </el-form-item>

            <el-form-item label="状态" prop="status" class="advanced-last-item">
              <el-radio-group v-model="form.status">
                <el-radio value="enabled">启用</el-radio>
                <el-radio value="disabled">停用</el-radio>
              </el-radio-group>
            </el-form-item>
          </div>
        </el-collapse-transition>
      </div>

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
          { preset: 'cancel', onClick: () => handleCancel() },
          { preset: 'save', onClick: () => handleSave() }
        ]"
      />
    </template>
  </ele-modal>
</template>

<script setup>
  import { computed, onMounted, ref, reactive, watch } from 'vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { Plus, Minus, ArrowDown } from '@element-plus/icons-vue';
  import { pageMerchantDomains } from '@/api/merchant/domain';
  import { addMerchantCloak, updateMerchantCloak } from '@/api/merchant/cloak';
  import CountryRegionsSelect from './country-regions-select.vue';
  import { buildRegionPaths } from '@/utils/country-regions';

  const props = defineProps({
    data: Object,
    copy: Boolean
  });

  const emit = defineEmits(['done']);

  const { modalProps, closeModal } = useModal();

  const loading = ref(false);
  const domainLoading = ref(false);
  const domainOptions = ref([]);
  const formRef = ref(null);
  const advancedOpen = ref(false);
  const linkRowKeys = ref([0]);
  const cloakId = ref(null);
  let linkRowSeed = 1;

  const modalTitle = computed(() => {
    if (props.copy) {
      return '复制斗篷';
    }
    if (cloakId.value) {
      return '编辑斗篷';
    }
    return '添加斗篷';
  });

  const modalBodyStyle = {
    paddingTop: '4px',
    height: '100%',
    overflowY: 'auto',
    scrollbarGutter: 'stable',
    boxSizing: 'border-box'
  };

  const defaultRules = () => ({
    reviewPageMode: 'external',
    realUrls: [''],
    shortLinkType: 'system',
    shortLinkUrl: '',
    allowedSources: ['all'],
    mode: 'cloak',
    spiderWhitelist: '',
    deviceWhitelist: '',
    blockEmptyLanguage: false,
    blockPc: false,
    blockProxy: false,
    intermediatePage: 'none',
    reviewQuickTag: ''
  });

  const [form] = useFormData({
    name: '',
    domainId: void 0,
    whiteUrl: '',
    blackUrl: '',
    status: 'disabled',
    remark: '',
    targetRegionPaths: [],
    rules: defaultRules()
  });

  const realUrlRules = [
    { required: true, message: '请输入真实链接', trigger: 'blur' },
    { type: 'url', message: '链接格式不正确', trigger: 'blur' }
  ];

  const rules = reactive({
    name: [
      { required: true, message: '请输入名称', trigger: 'blur' },
      { max: 50, message: '名称不能超过 50 个字符', trigger: 'blur' }
    ],
    domainId: [
      { required: true, message: '请选择关联域名', type: 'number', trigger: 'change' }
    ],
    whiteUrl: [
      { required: true, message: '请输入审核链接', trigger: 'blur' },
      { type: 'url', message: '审核链接格式不正确', trigger: 'blur' }
    ],
    'rules.reviewPageMode': [
      { required: true, message: '请选择审核页模式', trigger: 'change' }
    ],
    'rules.shortLinkType': [
      { required: true, message: '请选择跳转短链类型', trigger: 'change' }
    ],
    'rules.mode': [{ required: true, message: '请选择模式', trigger: 'change' }],
    targetRegionPaths: [
      {
        required: true,
        type: 'array',
        min: 1,
        message: '请选择投放地区',
        trigger: 'change'
      }
    ]
  });

  watch(
    () => [...form.rules.allowedSources],
    (value, oldValue) => {
      if (!Array.isArray(value) || value.length === 0) {
        form.rules.allowedSources = ['all'];
        return;
      }

      const hadAll = Array.isArray(oldValue) && oldValue.includes('all');
      const hasAll = value.includes('all');
      const specifics = value.filter((item) => item !== 'all');

      let next = value;

      if (hasAll && specifics.length > 0) {
        next = hadAll ? specifics : ['all'];
      } else if (!hasAll && specifics.length === 0) {
        next = ['all'];
      }

      if (next.join(',') !== value.join(',')) {
        form.rules.allowedSources = next;
      }
    }
  );

  const quickReviewMap = {
    google: 'https://www.google.com',
    facebook: 'https://www.facebook.com',
    tiktok: 'https://www.tiktok.com',
    generic: 'https://example.com/safe-page'
  };

  const handleQuickReview = (value) => {
    if (value && quickReviewMap[value]) {
      form.whiteUrl = quickReviewMap[value];
    }
  };

  const addRealUrl = () => {
    form.rules.realUrls.push('');
    linkRowKeys.value.push(linkRowSeed++);
  };

  const removeRealUrl = (index) => {
    if (form.rules.realUrls.length <= 1) {
      return;
    }
    form.rules.realUrls.splice(index, 1);
    linkRowKeys.value.splice(index, 1);
  };

  const loadDomains = () => {
    domainLoading.value = true;
    pageMerchantDomains({ status: 'bound', pageNum: 1, pageSize: 200 })
      .then((data) => {
        domainLoading.value = false;
        domainOptions.value = data.rows || [];
        if (!domainOptions.value.length) {
          EleMessage.warning({
            message: '暂无已绑定域名，请先添加并校验域名',
            plain: true
          });
        }
      })
      .catch((e) => {
        domainLoading.value = false;
        EleMessage.error({ message: e.message, plain: true });
      });
  };

  const buildPayload = () => {
    const realUrls = form.rules.realUrls.filter((item) => !!item);
    const targetRegions = form.targetRegionPaths.map(
      (path) => path[path.length - 1]
    );
    const rules = { ...form.rules, realUrls, targetRegions };

    if (rules.shortLinkType === 'system' && !props.copy && cloakId.value) {
      delete rules.shortLinkUrl;
    }
    if (props.copy && rules.shortLinkType === 'system') {
      rules.shortLinkUrl = '';
    }

    return {
      name: form.name,
      domainId: form.domainId,
      whiteUrl: form.whiteUrl,
      blackUrl: realUrls[0],
      status: form.status,
      remark: form.remark,
      defaultAction: 'block',
      rules
    };
  };

  const assignFromRow = (row, copy = false) => {
    const rowRules = row.rules || {};
    const realUrls = row.realUrls?.length ? [...row.realUrls] : [row.blackUrl || ''];
    linkRowKeys.value = realUrls.map((_, index) => index);
    linkRowSeed = realUrls.length;

    cloakId.value = copy ? null : row.cloakId;
    form.name = copy
      ? `${row.cloakName || ''}-副本`.slice(0, 50)
      : row.cloakName || '';
    form.domainId = row.domainId;
    form.whiteUrl = row.whiteUrl || '';
    form.blackUrl = row.blackUrl || '';
    form.status = copy ? 'disabled' : row.status || 'disabled';
    form.remark = row.remark || '';
    form.targetRegionPaths = buildRegionPaths(row.targetRegions || []);
    form.rules = {
      ...defaultRules(),
      ...rowRules,
      realUrls,
      shortLinkUrl:
        copy && (rowRules.shortLinkType || 'system') === 'system'
          ? ''
          : rowRules.shortLinkUrl || ''
    };
  };

  const handleCancel = () => {
    closeModal();
  };

  const handleSave = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) {
        return;
      }
      if (form.rules.shortLinkType === 'custom' && !form.rules.shortLinkUrl) {
        EleMessage.error({ message: '请输入自定义短链', plain: true });
        return;
      }
      loading.value = true;
      const request = cloakId.value
        ? updateMerchantCloak(cloakId.value, buildPayload())
        : addMerchantCloak(buildPayload());
      request
        .then((res) => {
          loading.value = false;
          const shortLink = res.data?.shortLinkUrl;
          EleMessage.success({
            message: shortLink ? `${res.msg}，短链：${shortLink}` : res.msg,
            plain: true
          });
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
    loadDomains();
    if (props.data) {
      assignFromRow(props.data, !!props.copy);
    }
  });
</script>

<style scoped>
  .cloak-alert {
    margin-bottom: 12px;
  }

  .cloak-form :deep(.el-form-item) {
    margin-bottom: 16px;
  }

  .link-list {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .link-list-item {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    width: 100%;
    position: relative;
    min-height: 32px;
  }

  .link-input-item {
    flex: 1;
    margin-bottom: 0;
    min-width: 0;
  }

  .link-input-item :deep(.el-form-item__error) {
    position: absolute;
    top: 100%;
    left: 0;
    padding-top: 2px;
    line-height: 1;
  }

  .link-action-btn {
    flex-shrink: 0;
    width: 32px;
    height: 32px;
    margin-top: 1px;
    padding: 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: 1px solid var(--el-border-color);
    border-radius: 50%;
    background: #fff;
    color: var(--el-text-color-secondary);
    cursor: pointer;
    transition: color 0.2s, border-color 0.2s;
  }

  .link-action-btn:hover {
    color: var(--el-color-primary);
    border-color: var(--el-color-primary);
  }

  .review-link-row {
    display: flex;
    gap: 8px;
    width: 100%;
  }

  .review-quick-select {
    width: 148px;
    flex-shrink: 0;
  }

  .review-link-row .el-input {
    flex: 1;
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

  .sub-input {
    margin-top: 8px;
  }

  .advanced-section {
    margin-top: 4px;
  }

  .advanced-toggle {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    margin: 0 0 4px 100px;
    padding: 0;
    border: none;
    background: transparent;
    font-size: 14px;
    line-height: 22px;
    color: var(--el-text-color-regular);
    cursor: pointer;
    transition: color 0.2s;
  }

  .advanced-toggle:hover {
    color: var(--el-color-primary);
  }

  .advanced-toggle-icon {
    font-size: 12px;
    transition: transform 0.25s ease;
  }

  .advanced-toggle.is-expanded .advanced-toggle-icon {
    transform: rotate(180deg);
  }

  .advanced-panel {
    padding-top: 8px;
  }

  .advanced-last-item {
    margin-bottom: 0;
  }

  .switch-group {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .switch-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    max-width: 280px;
    font-size: 14px;
    color: var(--el-text-color-regular);
  }

  .switch-group-item :deep(.el-form-item__content) {
    line-height: normal;
  }
</style>
