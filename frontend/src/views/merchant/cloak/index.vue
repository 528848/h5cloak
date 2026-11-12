<template>
  <ele-page flex-table="auto">
    <summary-cards :items="summaryItems" :cols="4" clickable :active-key="activeKey" @select="selectSummary" />
    <ele-card search-form class="page-section">
      <el-form label-width="72px" @keyup.enter="search" @submit.prevent="">
        <el-row :gutter="12">
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="名称">
              <el-input v-model.trim="form.keyword" clearable placeholder="请输入斗篷名称" />
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="关联域名">
              <el-select v-model="form.domainId" clearable filterable placeholder="全部" class="ele-fluid">
                <el-option v-for="item in domains" :key="item.id" :label="item.domain" :value="item.id" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="模式">
              <el-select v-model="form.mode" clearable placeholder="全部" class="ele-fluid">
                <el-option v-for="item in modeOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="状态">
              <el-select v-model="form.status" clearable placeholder="全部" class="ele-fluid">
                <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label-width="0">
              <btn-items
                :wrap="false"
                :items="[
                  { preset: 'search', onClick: () => search() },
                  { preset: 'reset', onClick: () => reset() }
                ]"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </ele-card>
    <ele-card class="table-section" :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        ref="tableRef"
        row-key="id"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        cache-key="merchantCloakList"
      >
        <template #toolbar>
          <el-button v-permission="'cloak:cloak:add'" type="primary" @click="openEdit()">新建斗篷</el-button>
        </template>
        <template #name="{ row }">
          <el-link type="primary" underline="never" @click="openEdit(row)">{{ row.name }}</el-link>
        </template>
        <template #links="{ row }">
          <div class="link-line"><span>真实</span><em class="mono">{{ row.realUrl || '-' }}</em><el-button v-if="row.realUrl" link type="primary" :icon="DocumentCopy" @click="copy(row.realUrl)" /></div>
          <div class="link-line">
            <span>审核</span>
            <em class="mono">{{ row.reviewMode === 'local_template' && !row.reviewUrl ? '本地模版' : row.reviewUrl || '-' }}</em>
            <el-button v-if="row.reviewUrl" link type="primary" :icon="DocumentCopy" @click="copy(row.reviewUrl)" />
          </div>
          <div class="link-line"><span>短链</span><em class="mono">{{ row.shortLink }}</em><el-button link type="primary" :icon="DocumentCopy" @click="copy(row.shortLink)" /></div>
        </template>
        <template #status="{ row }">
          <el-tag :type="typeOf(statusOptions, row.status)" size="small" effect="light">{{ labelOf(statusOptions, row.status) }}</el-tag>
        </template>
        <template #mode="{ row }">
          <el-tag :type="typeOf(modeOptions, row.mode)" size="small" effect="plain">{{ labelOf(modeOptions, row.mode) }}</el-tag>
        </template>
        <template #hits="{ row }">
          <div>审核 {{ row.reviewHits }} · 今日 {{ row.todayReviewHits }}</div>
          <div>真实 {{ row.realHits }} · 今日 {{ row.todayRealHits }}</div>
        </template>
        <template #action="{ row }">
          <el-space :size="6" spacer="|">
            <el-button v-permission="'cloak:cloak:edit'" link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button v-permission="'cloak:cloak:copy'" link type="primary" @click="copyRow(row)">复制</el-button>
            <el-button v-permission="'cloak:cloak:status'" link type="warning" @click="toggle(row)">
              {{ row.status === 'enabled' ? '停用' : '启用' }}
            </el-button>
            <el-button v-permission="'cloak:cloak:remove'" link type="danger" @click="remove(row)">删除</el-button>
          </el-space>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue';
  import { DocumentCopy } from '@element-plus/icons-vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { copyCloak, getCloakStats, listCloakDomains, pageCloaks, removeCloak, updateCloakStatus } from '@/api/merchant/cloak';
  import { formatCountryCodes } from '@/utils/country-regions';
  import { intermediateOptions, labelOf, modeOptions, statusOptions, typeOf } from './constants';

  defineOptions({ name: 'MerchantCloak' });

  const { openModal } = useModal();
  const tableRef = ref(null);
  const domains = ref([]);
  const stats = ref({ total: 0, enabled: 0, disabled: 0, today: 0 });
  const activeKey = ref('');
  const [form, resetFields] = useFormData({ keyword: '', domainId: undefined, mode: '', status: '' });

  const summaryItems = computed(() => [
    { key: '', label: '全部斗篷', value: stats.value.total, desc: '跳转短链，不是站点', type: 'primary' },
    { key: 'enabled', label: '启用中', value: stats.value.enabled, desc: '短链可以访问', type: 'success' },
    { key: 'disabled', label: '已停用', value: stats.value.disabled, desc: '短链返回 404', type: 'info' },
    { key: 'today', label: '今日跳转', value: stats.value.today, desc: '审核与真实合计', type: 'warning', clickable: false }
  ]);

  const columns = [
    { prop: 'name', label: '名称', minWidth: 140, slot: 'name', fixed: 'left', formatter: (row) => row.name },
    { columnKey: 'links', label: '链接信息', minWidth: 280, slot: 'links', formatter: (row) => row.shortLink },
    { prop: 'status', label: '状态', width: 90, align: 'center', slot: 'status', formatter: (row) => labelOf(statusOptions, row.status) },
    {
      prop: 'targetRegions',
      label: '投放地区',
      minWidth: 160,
      formatter: (row) => (row.targetRegions?.length ? formatCountryCodes(row.targetRegions) : '不限')
    },
    { prop: 'mode', label: '模式', width: 90, align: 'center', slot: 'mode', formatter: (row) => labelOf(modeOptions, row.mode) },
    { prop: 'remark', label: '备注', minWidth: 120, formatter: (row) => row.remark || '-' },
    { columnKey: 'hits', label: '链接数据', minWidth: 180, slot: 'hits', formatter: (row) => `审核 ${row.reviewHits} / 真实 ${row.realHits}` },
    { prop: 'createdAt', label: '创建时间', minWidth: 170 },
    { prop: 'deviceWhitelistText', label: '设备白名单', width: 130, formatter: (row) => row.deviceWhitelistText },
    { prop: 'blockPc', label: '屏蔽PC', width: 90, align: 'center', formatter: (row) => (row.blockPc ? '是' : '否') },
    { prop: 'blockProxy', label: '屏蔽代理', width: 100, align: 'center', formatter: (row) => (row.blockProxy ? '是' : '否') },
    { prop: 'allowedSourcesText', label: '允许访问来源', minWidth: 140 },
    { prop: 'intermediatePage', label: '中间页', width: 110, formatter: (row) => labelOf(intermediateOptions, row.intermediatePage) },
    { columnKey: 'action', label: '操作', width: 220, align: 'center', slot: 'action', fixed: 'right', hideInPrint: true, hideInExport: true }
  ];

  const datasource = ({ page, limit }) =>
    pageCloaks({
      keyword: form.keyword,
      domainId: form.domainId || undefined,
      mode: form.mode,
      status: form.status,
      pageNum: page,
      pageSize: limit
    });

  const loadStats = () => getCloakStats().then((data) => (stats.value = data || stats.value)).catch(() => {});
  const reload = () => {
    tableRef.value?.reload?.({ page: 1 });
    loadStats();
  };
  const search = () => {
    activeKey.value = form.status || '';
    reload();
  };
  const reset = () => {
    resetFields();
    activeKey.value = '';
    reload();
  };
  const selectSummary = (item) => {
    if (item.clickable === false) return;
    activeKey.value = item.key;
    form.status = item.key;
    reload();
  };
  const openEdit = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/cloak-edit.vue'),
      componentProps: { data: row, onDone: () => reload() }
    });
  };
  const copy = async (text) => {
    try {
      await navigator.clipboard.writeText(text);
      EleMessage.success({ message: '已复制', plain: true });
    } catch {
      EleMessage.error({ message: '复制失败', plain: true });
    }
  };
  const copyRow = (row) => {
    ElMessageBox.confirm(`确认复制斗篷「${row.name}」？新斗篷默认停用，并生成新的短链。`, '复制斗篷', { type: 'warning', draggable: true })
      .then(() => copyCloak(row.id))
      .then((msg) => {
        EleMessage.success({ message: msg, plain: true });
        reload();
      })
      .catch((e) => { if (e?.message) EleMessage.error({ message: e.message, plain: true }); });
  };
  const toggle = (row) => {
    const status = row.status === 'enabled' ? 'disabled' : 'enabled';
    const text = status === 'enabled' ? '启用' : '停用';
    ElMessageBox.confirm(`确认${text}斗篷「${row.name}」？`, '系统提示', { type: 'warning', draggable: true })
      .then(() => updateCloakStatus(row.id, status))
      .then((msg) => {
        EleMessage.success({ message: msg, plain: true });
        reload();
      })
      .catch((e) => { if (e?.message) EleMessage.error({ message: e.message, plain: true }); });
  };
  const remove = (row) => {
    ElMessageBox.confirm(`确认删除斗篷「${row.name}」？删除后短链不可再访问。`, '删除斗篷', { type: 'warning', draggable: true })
      .then(() => removeCloak(row.id))
      .then((msg) => {
        EleMessage.success({ message: msg, plain: true });
        reload();
      })
      .catch((e) => { if (e?.message) EleMessage.error({ message: e.message, plain: true }); });
  };

  onMounted(() => {
    loadStats();
    listCloakDomains({ scope: 'all' }).then((rows) => (domains.value = rows || [])).catch(() => {});
  });
</script>

<style scoped>
  .link-line {
    display: flex;
    align-items: center;
    gap: 6px;
    line-height: 22px;
  }

  .link-line span {
    flex: none;
    color: var(--el-text-color-secondary);
  }

  .link-line em {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-style: normal;
  }
</style>
