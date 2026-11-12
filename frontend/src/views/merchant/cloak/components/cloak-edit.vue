<!-- 斗篷新建 / 编辑弹窗，结构对齐 V1：常用项在外，高级设置可展开。 -->
<template>
  <ele-modal
    form
    :width="720"
    :height="660"
    :title="title"
    :loading="loading"
    :body-style="modalBodyStyle"
    v-bind="modalProps"
    maxable
  >
    <el-alert
      v-if="showMarketNotice"
      title="斗篷仅适用于海外市场，中国及港澳台不适用"
      type="warning"
      show-icon
      class="cloak-alert"
      @close="dismissMarketNotice"
    />
    <el-form ref="formRef" :model="form" :rules="rules" label-width="108px" @submit.prevent="">
      <el-form-item label="名称" prop="name">
        <el-input v-model.trim="form.name" maxlength="64" show-word-limit placeholder="请输入名称" />
      </el-form-item>
      <el-form-item label="关联域名" prop="domainId">
        <el-select
          v-model="form.domainId"
          :disabled="!!cloakId"
          filterable
          placeholder="请选择已绑定域名"
          class="ele-fluid"
          :loading="domainLoading"
        >
          <el-option v-for="item in domains" :key="item.id" :label="item.domain" :value="item.id" />
        </el-select>
        <div class="field-tip">只用服务端站点上的域名。托管站点和混合站点的域名不能选。</div>
      </el-form-item>
      <el-form-item label="真实链接" required>
        <div class="link-list">
          <div v-for="(_, index) in form.realUrls" :key="linkRowKeys[index]" class="link-list-item">
            <el-form-item :prop="'realUrls.' + index" :rules="realUrlRules" class="link-input-item">
              <el-input v-model.trim="form.realUrls[index]" maxlength="1000" placeholder="请输入真实链接" />
            </el-form-item>
            <button v-if="index === 0" type="button" class="link-action-btn" @click="addRealUrl">
              <el-icon><Plus /></el-icon>
            </button>
            <button v-else type="button" class="link-action-btn" @click="removeRealUrl(index)">
              <el-icon><Minus /></el-icon>
            </button>
          </div>
        </div>
        <div class="field-tip">放行时使用第一条。</div>
      </el-form-item>
      <el-form-item label="审核页模式">
        <el-radio-group v-model="form.reviewMode">
          <el-radio value="external">跳转外链</el-radio>
          <el-radio value="local_template">本地模版</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item :label="form.reviewMode === 'external' ? '审核链接' : '快速审核'" :prop="form.reviewMode === 'external' ? 'reviewUrl' : 'reviewTag'">
        <div class="review-link-row">
          <el-select v-model="form.reviewTag" clearable placeholder="请选择快速审核" class="review-quick-select" @change="applyQuickReview">
            <el-option v-for="item in reviewTagOptions" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
          <el-input
            v-if="form.reviewMode === 'external'"
            v-model.trim="form.reviewUrl"
            maxlength="1000"
            placeholder="请输入审核链接"
          />
        </div>
      </el-form-item>
      <el-form-item label="跳转短链">
        <div class="inline-field">
          <el-radio-group v-model="form.shortLinkType">
            <el-radio value="system">系统链接</el-radio>
            <el-radio value="custom">自定义链接</el-radio>
          </el-radio-group>
          <span v-if="form.shortLinkType === 'system'" class="field-tip">
            {{ form.shortLink ? `当前短链：${form.shortLink}` : '保存后系统自动生成短链' }}
          </span>
        </div>
        <el-input
          v-if="form.shortLinkType === 'custom'"
          v-model.trim="form.shortLinkUrl"
          maxlength="1000"
          placeholder="https://域名/c/编码"
          class="sub-input"
        />
      </el-form-item>
      <el-form-item label="允许访问来源">
        <el-checkbox-group v-model="form.allowedSources">
          <el-checkbox v-for="item in sourceOptions" :key="item.value" :value="item.value" :label="item.label" />
        </el-checkbox-group>
      </el-form-item>
      <el-form-item label="模式">
        <div class="inline-field">
          <el-radio-group v-model="form.mode">
            <el-radio v-for="item in modeOptions" :key="item.value" :value="item.value">{{ item.label }}</el-radio>
          </el-radio-group>
          <span class="field-tip field-tip-warning">绿色和审核固定去审核侧，全开固定去真实链接</span>
        </div>
      </el-form-item>
      <el-form-item label="投放地区">
        <country-regions-select v-model="form.targetRegionPaths" />
        <div class="field-tip">按大洲选择，留空表示不限。</div>
      </el-form-item>

      <div class="advanced-section" :class="{ 'is-open': advancedOpen }">
        <button type="button" class="advanced-toggle" :aria-expanded="advancedOpen" @click="advancedOpen = !advancedOpen">
          <span class="advanced-toggle-text">
            <strong>高级设置</strong>
            <em>{{ advancedOpen ? '收起蜘蛛、设备、访问限制和中间页' : '展开蜘蛛、设备、访问限制和中间页' }}</em>
          </span>
          <el-icon class="advanced-toggle-icon"><ArrowDown /></el-icon>
        </button>
        <el-collapse-transition>
          <div v-show="advancedOpen" class="advanced-panel">
            <el-form-item label="蜘蛛白名单">
              <el-select v-model="form.spiderWhitelist" multiple clearable placeholder="请选择蜘蛛白名单" class="ele-fluid">
                <el-option v-for="item in spiderOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
            <el-form-item label="设备白名单">
              <el-select v-model="form.deviceWhitelist" clearable placeholder="请选择设备白名单" class="ele-fluid">
                <el-option v-for="item in deviceOptions" :key="item.value || 'any'" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
            <el-form-item label="访问限制">
              <div class="switch-group">
                <div class="switch-row"><span>禁止空语言访问</span><el-switch v-model="form.blockEmptyLanguage" /></div>
                <div class="switch-row"><span>屏蔽 PC</span><el-switch v-model="form.blockPc" /></div>
                <div class="switch-row"><span>屏蔽代理</span><el-switch v-model="form.blockProxy" /></div>
              </div>
            </el-form-item>
            <el-form-item label="中间页">
              <el-radio-group v-model="form.intermediatePage">
                <el-radio v-for="item in intermediateOptions" :key="item.value" :value="item.value">{{ item.label }}</el-radio>
              </el-radio-group>
              <el-input
                v-if="form.intermediatePage === 'custom_template'"
                v-model="form.intermediateText"
                type="textarea"
                :rows="3"
                maxlength="500"
                show-word-limit
                placeholder="自定义说明，留空则使用系统文案"
                class="sub-input"
              />
            </el-form-item>
            <el-form-item label="状态">
              <el-radio-group v-model="form.status">
                <el-radio value="enabled">启用</el-radio>
                <el-radio value="disabled">停用</el-radio>
              </el-radio-group>
            </el-form-item>
          </div>
        </el-collapse-transition>
      </div>
      <el-form-item label="备注">
        <el-input v-model="form.remark" type="textarea" :rows="2" maxlength="200" show-word-limit placeholder="请输入备注" />
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
  import { computed, onMounted, reactive, ref, watch } from 'vue';
  import { ArrowDown, Minus, Plus } from '@element-plus/icons-vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { buildRegionPaths } from '@/utils/country-regions';
  import { addCloak, listCloakDomains, updateCloak } from '@/api/merchant/cloak';
  import CountryRegionsSelect from './country-regions-select.vue';
  import { deviceOptions, intermediateOptions, modeOptions, reviewTagOptions, sourceOptions, spiderOptions } from '../constants';

  const props = defineProps({ data: Object });
  const emit = defineEmits(['done']);
  const { modalProps, closeModal } = useModal();

  /** 关闭后写入本地，下次打开弹框不再展示。 */
  const MARKET_NOTICE_KEY = 'cloak.market-notice.dismissed';

  const loading = ref(false);
  const domainLoading = ref(false);
  const showMarketNotice = ref(localStorage.getItem(MARKET_NOTICE_KEY) !== '1');

  const dismissMarketNotice = () => {
    showMarketNotice.value = false;
    localStorage.setItem(MARKET_NOTICE_KEY, '1');
  };
  const domains = ref([]);
  const formRef = ref(null);
  const advancedOpen = ref(false);
  const linkRowKeys = ref([0]);
  const cloakId = ref(null);
  let linkRowSeed = 1;
  const title = computed(() => (cloakId.value ? '编辑斗篷' : '添加斗篷'));
  const modalBodyStyle = { paddingTop: '4px', height: '100%', overflowY: 'auto', scrollbarGutter: 'stable', boxSizing: 'border-box' };

  const [form] = useFormData({
    name: '',
    domainId: undefined,
    realUrls: [''],
    reviewMode: 'external',
    reviewUrl: '',
    reviewTag: '',
    shortLinkType: 'system',
    shortLinkUrl: '',
    shortLink: '',
    allowedSources: ['all'],
    mode: 'cloak',
    targetRegionPaths: [],
    spiderWhitelist: [],
    deviceWhitelist: '',
    blockEmptyLanguage: false,
    blockPc: false,
    blockProxy: false,
    intermediatePage: 'none',
    intermediateText: '',
    status: 'enabled',
    remark: ''
  });

  const realUrlRules = [
    { required: true, message: '请输入真实链接', trigger: 'blur' },
    { type: 'url', message: '链接格式不正确', trigger: 'blur' }
  ];
  const rules = reactive({
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    domainId: [{ required: true, message: '请选择关联域名', trigger: 'change' }],
    reviewUrl: [{ required: true, message: '请输入审核链接', trigger: 'blur' }, { type: 'url', message: '审核链接格式不正确', trigger: 'blur' }],
    reviewTag: [{ required: true, message: '请选择快速审核页', trigger: 'change' }]
  });

  const quickReviewMap = {
    google: 'https://www.google.com',
    facebook: 'https://www.facebook.com',
    tiktok: 'https://www.tiktok.com',
    generic: 'https://example.com/safe-page'
  };

  watch(
    () => [...form.allowedSources],
    (value, oldValue) => {
      if (!value.length) {
        form.allowedSources = ['all'];
        return;
      }
      const hadAll = (oldValue || []).includes('all');
      const specifics = value.filter((item) => item !== 'all');
      let next = value;
      if (value.includes('all') && specifics.length) next = hadAll ? specifics : ['all'];
      else if (!value.includes('all') && !specifics.length) next = ['all'];
      if (next.join(',') !== value.join(',')) form.allowedSources = next;
    }
  );

  const applyQuickReview = (value) => {
    if (form.reviewMode === 'external' && quickReviewMap[value]) form.reviewUrl = quickReviewMap[value];
  };
  const addRealUrl = () => {
    form.realUrls.push('');
    linkRowKeys.value.push(linkRowSeed++);
  };
  const removeRealUrl = (index) => {
    form.realUrls.splice(index, 1);
    linkRowKeys.value.splice(index, 1);
  };

  const handleSave = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) return;
      if (form.shortLinkType === 'custom' && !form.shortLinkUrl) {
        EleMessage.error({ message: '请输入自定义短链', plain: true });
        return;
      }
      loading.value = true;
      const payload = {
        ...form,
        realUrls: form.realUrls.map((item) => item.trim()).filter(Boolean),
        targetRegions: form.targetRegionPaths.map((path) => path[path.length - 1]).filter(Boolean)
      };
      const req = cloakId.value ? updateCloak(cloakId.value, payload) : addCloak(payload).then(() => '创建成功');
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

  onMounted(() => {
    const row = props.data;
    cloakId.value = row?.id || null;
    domainLoading.value = true;
    listCloakDomains({ keepId: cloakId.value || undefined })
      .then((rows) => {
        domains.value = rows || [];
        if (row?.domainId && !domains.value.some((item) => item.id === row.domainId)) {
          domains.value.unshift({ id: row.domainId, domain: row.domain });
        }
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => (domainLoading.value = false));
    if (!row) return;
    const urls = row.realUrls?.length ? [...row.realUrls] : [row.realUrl || ''];
    linkRowKeys.value = urls.map((_, index) => index);
    linkRowSeed = urls.length;
    Object.assign(form, {
      ...row,
      realUrls: urls,
      reviewTag: row.reviewTag || '',
      shortLinkType: row.codeCustom ? 'custom' : 'system',
      shortLinkUrl: row.codeCustom ? row.shortLink : '',
      shortLink: row.shortLink || '',
      targetRegionPaths: buildRegionPaths(row.targetRegions),
      spiderWhitelist: row.spiderWhitelist || [],
      allowedSources: row.allowedSources?.length ? [...row.allowedSources] : ['all']
    });
    advancedOpen.value = !!(row.spiderWhitelist?.length || row.deviceWhitelist || row.blockPc || row.blockProxy || row.blockEmptyLanguage || (row.intermediatePage && row.intermediatePage !== 'none'));
  });
</script>

<style scoped>
  .cloak-alert { margin-bottom: 12px; }
  .field-tip { margin-top: 6px; font-size: 12px; line-height: 1.4; color: var(--el-text-color-secondary); }
  .field-tip-warning { margin-top: 0; color: var(--el-color-warning); }
  .link-list { width: 100%; display: flex; flex-direction: column; gap: 8px; }
  .link-list-item { display: flex; align-items: flex-start; gap: 8px; width: 100%; }
  .link-input-item { flex: 1; margin-bottom: 0; min-width: 0; }
  .link-action-btn {
    flex-shrink: 0; width: 32px; height: 32px; padding: 0; display: inline-flex; align-items: center; justify-content: center;
    border: 1px solid var(--el-border-color); border-radius: 50%; background: var(--el-bg-color); color: var(--el-text-color-secondary); cursor: pointer;
  }
  .link-action-btn:hover { color: var(--el-color-primary); border-color: var(--el-color-primary); }
  .review-link-row { display: flex; gap: 8px; width: 100%; }
  .review-quick-select { width: 148px; flex-shrink: 0; }
  .review-link-row .el-input { flex: 1; }
  .inline-field { display: flex; flex-wrap: wrap; align-items: center; gap: 8px 12px; width: 100%; }
  .sub-input { margin-top: 8px; }
  .advanced-section {
    margin: 4px 0 16px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    background: var(--el-fill-color-blank);
  }

  .advanced-section.is-open {
    border-color: var(--el-color-primary-light-7);
  }

  .advanced-toggle {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    width: 100%;
    margin: 0;
    padding: 10px 14px;
    border: none;
    background: transparent;
    text-align: left;
    cursor: pointer;
  }

  .advanced-toggle:hover {
    background: var(--el-fill-color-light);
  }

  .advanced-toggle-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .advanced-toggle-text strong {
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .advanced-toggle-text em {
    font-style: normal;
    font-size: 12px;
    line-height: 1.4;
    color: var(--el-text-color-secondary);
  }

  .advanced-toggle-icon {
    flex: none;
    font-size: 14px;
    color: var(--el-text-color-secondary);
    transition: transform 0.25s ease;
  }

  .advanced-section.is-open .advanced-toggle-icon {
    transform: rotate(180deg);
    color: var(--el-color-primary);
  }

  .advanced-panel {
    padding: 4px 14px 8px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
  .switch-group { width: 100%; display: flex; flex-direction: column; gap: 10px; }
  .switch-row { display: flex; align-items: center; justify-content: space-between; max-width: 280px; }
</style>
