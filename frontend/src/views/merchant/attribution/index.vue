<template>
  <ele-page>
    <ele-card search-form class="page-section">
      <el-form label-width="72px" @submit.prevent="">
        <el-row :gutter="12">
          <el-col :lg="8" :md="12" :xs="24">
            <el-form-item label="站点">
              <el-select
                v-model="siteId"
                placeholder="请选择站点"
                class="ele-fluid"
              >
                <el-option
                  v-for="s in sites"
                  :key="s.siteId"
                  :label="s.name"
                  :value="s.siteId"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :lg="16" :md="12" :xs="24">
            <div class="site-hint muted-text">
              归因规则按站点生效。访客带着下面任一通过校验的参数访问时放行，否则返回审核内容。
            </div>
          </el-col>
        </el-row>
      </el-form>
    </ele-card>

    <el-empty
      v-if="loaded && !siteId"
      description="请先在「站点管理」创建站点"
      :image-size="90"
    />

    <template v-else-if="siteId">
      <section-card
        title="校验规则"
        subtitle="决定多个参数如何组合、归因码与点击参数如何校验"
        class="page-section"
      >
        <template #extra>
          <el-button
            v-permission="'attribution:rule:edit'"
            type="primary"
            plain
            :loading="savingRule"
            @click="handleSaveRule"
          >
            保存规则
          </el-button>
        </template>
        <el-form
          v-loading="ruleLoading"
          :model="rule"
          label-width="130px"
          :disabled="!canEdit"
          @submit.prevent=""
        >
          <el-row :gutter="16">
            <el-col :span="24">
              <el-form-item label="组合要求">
                <el-radio-group v-model="rule.combineRule" class="stack-group">
                  <el-radio
                    v-for="o in combineOptions"
                    :key="o.value"
                    :value="o.value"
                  >
                    {{ o.label }}<em class="opt-desc">{{ o.desc }}</em>
                  </el-radio>
                </el-radio-group>
              </el-form-item>
            </el-col>
            <el-col :md="12" :xs="24">
              <el-form-item label="校验归因码登记">
                <el-switch v-model="rule.validateCodes" />
                <span class="field-hint">关闭后任何码值都视为有效</span>
              </el-form-item>
            </el-col>
            <el-col :md="12" :xs="24">
              <el-form-item label="校验点击参数格式">
                <el-switch v-model="rule.validateClickFormat" />
                <span class="field-hint">按参数字典的长度与正则</span>
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="允许来源">
                <el-select
                  v-model="rule.allowedSources"
                  multiple
                  filterable
                  allow-create
                  default-first-option
                  placeholder="utm_source 允许值，留空不限制。回车添加"
                  class="ele-fluid"
                >
                  <el-option
                    v-for="s in sourceSuggestions"
                    :key="s"
                    :label="s"
                    :value="s"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :md="8" :xs="24">
              <el-form-item label="点击参数复用上限">
                <el-input-number
                  v-model="rule.clickReuseLimit"
                  :min="0"
                  :max="1000"
                  controls-position="right"
                />
                <span class="field-hint">0 为不限制</span>
              </el-form-item>
            </el-col>
            <el-col :md="8" :xs="24">
              <el-form-item label="复用统计时间窗">
                <el-input-number
                  v-model="rule.clickReuseWindowHours"
                  :min="1"
                  :max="720"
                  controls-position="right"
                />
                <span class="field-hint">小时</span>
              </el-form-item>
            </el-col>
            <el-col :md="8" :xs="24">
              <el-form-item label="超限后">
                <el-select v-model="rule.clickReuseAction" class="ele-fluid">
                  <el-option
                    v-for="o in reuseActionOptions"
                    :key="o.value"
                    :label="o.label"
                    :value="o.value"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="码停用后">
                <el-radio-group
                  v-model="rule.codeRevokeMode"
                  class="stack-group"
                >
                  <el-radio
                    v-for="o in revokeModeOptions"
                    :key="o.value"
                    :value="o.value"
                  >
                    {{ o.label }}<em class="opt-desc">{{ o.desc }}</em>
                  </el-radio>
                </el-radio-group>
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
      </section-card>

      <section-card
        title="参数字典"
        subtitle="平台内置参数可按站点覆盖；自定义参数只对本站点生效"
        class="table-section"
      >
        <template #extra>
          <el-space>
            <el-button link type="primary" :icon="Refresh" @click="loadParams"
              >刷新</el-button
            >
            <el-button
              v-permission="'attribution:rule:edit'"
              type="primary"
              plain
              @click="handleAddParam"
            >
              新增参数
            </el-button>
          </el-space>
        </template>
        <ele-pro-table
          ref="tableRef"
          row-key="paramId"
          :columns="columns"
          :datasource="params"
          :loading="paramsLoading"
          :pagination="false"
          :toolbar="false"
          :show-overflow-tooltip="true"
          size="small"
        >
          <template #name="{ row }">
            <span class="mono">{{ row.name }}</span>
            <span v-if="row.aliases?.length" class="muted-text alias-text">
              别名 {{ row.aliases.join(', ') }}
            </span>
          </template>
          <template #paramType="{ row }">
            <el-tag
              :type="getOptionType(paramTypeOptions, row.paramType)"
              effect="plain"
              size="small"
            >
              {{ getOptionLabel(paramTypeOptions, row.paramType) }}
            </el-tag>
          </template>
          <template #source="{ row }">
            <el-tag
              :type="getOptionType(paramSourceOptions, row.source)"
              effect="light"
              size="small"
            >
              {{ getOptionLabel(paramSourceOptions, row.source) }}
            </el-tag>
          </template>
          <template #format="{ row }">
            <span class="muted-text">{{ row.minLen }}-{{ row.maxLen }} 位</span>
            <span v-if="row.pattern" class="mono pattern-text">{{
              row.pattern
            }}</span>
          </template>
          <template #sufficient="{ row }">
            <el-tag
              :type="row.sufficient ? 'success' : 'info'"
              effect="plain"
              size="small"
            >
              {{ row.sufficient ? '是' : '否' }}
            </el-tag>
          </template>
          <template #status="{ row }">
            <el-tag
              :type="getOptionType(paramStatusOptions, row.status)"
              effect="light"
              size="small"
            >
              {{ getOptionLabel(paramStatusOptions, row.status) }}
            </el-tag>
          </template>
          <template #action="{ row }">
            <el-space :size="6" spacer="|">
              <el-button
                v-permission="'attribution:rule:edit'"
                link
                type="primary"
                @click="handleEditParam(row)"
              >
                {{ row.source === 'builtin' ? '覆盖' : '编辑' }}
              </el-button>
              <el-button
                v-if="row.source !== 'builtin'"
                v-permission="'attribution:rule:edit'"
                link
                :type="row.source === 'override' ? 'warning' : 'danger'"
                @click="handleRemoveParam(row)"
              >
                {{ row.source === 'override' ? '恢复内置' : '删除' }}
              </el-button>
            </el-space>
          </template>
        </ele-pro-table>
      </section-card>
    </template>
  </ele-page>
</template>

<script setup>
  import { ref, reactive, computed, watch, onMounted } from 'vue';
  import { Refresh } from '@element-plus/icons-vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { usePermission } from '@/utils/use-permission';
  import { useCurrentSite } from '@/utils/use-site';
  import {
    listAttributionParams,
    removeAttributionParam,
    getAttributionRule,
    saveAttributionRule
  } from '@/api/merchant/attribution';
  import {
    paramTypeOptions,
    paramSourceOptions,
    paramStatusOptions,
    combineOptions,
    reuseActionOptions,
    revokeModeOptions,
    getOptionLabel,
    getOptionType
  } from './constants';

  defineOptions({ name: 'MerchantAttribution' });

  const { openModal } = useModal();
  const { hasPermission } = usePermission();
  const canEdit = computed(() => hasPermission('attribution:rule:edit'));
  const { sites, siteId, loaded, load: loadSites } = useCurrentSite();

  const sourceSuggestions = [
    'facebook',
    'instagram',
    'google',
    'tiktok',
    'youtube',
    'twitter',
    'snapchat',
    'kwai',
    'linkedin',
    'email',
    'affiliate'
  ];

  const rule = reactive({
    combineRule: 'require_any',
    validateCodes: true,
    validateClickFormat: true,
    allowedSources: [],
    clickReuseLimit: 0,
    clickReuseWindowHours: 24,
    clickReuseAction: 'downgrade',
    codeRevokeMode: 'window'
  });
  const ruleLoading = ref(false);
  const savingRule = ref(false);

  const params = ref([]);
  const paramsLoading = ref(false);
  const tableRef = ref(null);

  const columns = [
    {
      prop: 'name',
      label: '参数名',
      minWidth: 200,
      slot: 'name'
    },
    {
      prop: 'paramType',
      label: '类型',
      width: 100,
      align: 'center',
      slot: 'paramType'
    },
    {
      prop: 'platform',
      label: '平台',
      width: 100,
      formatter: (row) => row.platform || '-'
    },
    { columnKey: 'format', label: '格式', minWidth: 200, slot: 'format' },
    {
      prop: 'sufficient',
      label: '单独放行',
      width: 90,
      align: 'center',
      slot: 'sufficient'
    },
    {
      prop: 'status',
      label: '状态',
      width: 80,
      align: 'center',
      slot: 'status'
    },
    {
      prop: 'source',
      label: '来源',
      width: 90,
      align: 'center',
      slot: 'source'
    },
    {
      prop: 'remark',
      label: '说明',
      minWidth: 160,
      formatter: (row) => row.remark || '-'
    },
    {
      columnKey: 'action',
      label: '操作',
      width: 140,
      align: 'center',
      slot: 'action'
    }
  ];

  const loadRule = () => {
    if (!siteId.value) return;
    ruleLoading.value = true;
    getAttributionRule(siteId.value)
      .then((d) => Object.assign(rule, d))
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => (ruleLoading.value = false));
  };

  const loadParams = () => {
    if (!siteId.value) return;
    paramsLoading.value = true;
    listAttributionParams(siteId.value)
      .then((rows) => (params.value = rows || []))
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => (paramsLoading.value = false));
  };

  const handleSaveRule = () => {
    savingRule.value = true;
    saveAttributionRule({ siteId: siteId.value, ...rule })
      .then((msg) => {
        EleMessage.success({ message: msg, plain: true });
        loadRule();
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => (savingRule.value = false));
  };

  const handleAddParam = () =>
    openModal({
      custom: true,
      asyncComponent: () => import('./components/param-edit.vue'),
      componentProps: { siteId: siteId.value, onDone: loadParams }
    });

  const handleEditParam = (row) =>
    openModal({
      custom: true,
      asyncComponent: () => import('./components/param-edit.vue'),
      componentProps: { siteId: siteId.value, data: row, onDone: loadParams }
    });

  const handleRemoveParam = (row) => {
    const text =
      row.source === 'override'
        ? `恢复参数「${row.name}」的平台内置设置？`
        : `删除自定义参数「${row.name}」？带该参数的访问将不再归因。`;
    ElMessageBox.confirm(text, '系统提示', { type: 'warning', draggable: true })
      .then(() => removeAttributionParam(siteId.value, row.paramId))
      .then((msg) => {
        if (!msg) return;
        EleMessage.success({ message: msg, plain: true });
        loadParams();
      })
      .catch((e) => {
        if (e?.message) EleMessage.error({ message: e.message, plain: true });
      });
  };

  watch(siteId, () => {
    loadRule();
    loadParams();
  });

  onMounted(() => {
    loadSites().then(() => {
      loadRule();
      loadParams();
    });
  });
</script>

<style scoped>
  .site-hint {
    line-height: 34px;
    font-size: 13px;
  }

  .stack-group {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }

  .opt-desc {
    font-style: normal;
    margin-left: 8px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .field-hint {
    margin-left: 8px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .alias-text {
    margin-left: 8px;
    font-size: 12px;
  }

  .pattern-text {
    margin-left: 8px;
    font-size: 12px;
  }
</style>
