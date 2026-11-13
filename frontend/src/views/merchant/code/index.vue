<template>
  <ele-page flex-table="auto">
    <summary-cards
      :items="summaryItems"
      :cols="4"
      clickable
      :active-key="activeStatus"
      @select="handleSummarySelect"
    />
    <code-search
      ref="searchRef"
      v-model:site-id="siteId"
      :sites="sites"
      @search="(where) => reload(where, 1)"
    />
    <ele-card
      class="table-section"
      :body-style="{ paddingTop: '8px' }"
      flex-table="auto"
    >
      <ele-pro-table
        ref="tableRef"
        row-key="codeId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        v-model:selections="selections"
        cache-key="merchantCodeTable"
      >
        <template #toolbar>
          <btn-items
            :items="[
              {
                preset: 'add',
                title: '登记',
                permission: 'attribution:code:add',
                onClick: () => handleAdd()
              },
              {
                title: '批量生成',
                type: 'primary',
                plain: true,
                permission: 'attribution:code:add',
                onClick: () => handleGenerate()
              },
              {
                title: '导入',
                plain: true,
                permission: 'attribution:code:add',
                onClick: () => handleImport()
              },
              {
                preset: 'export',
                permission: 'attribution:code:export',
                onClick: () => handleExport()
              },
              {
                title: '启用',
                type: 'success',
                plain: true,
                disabled: !selections.length,
                permission: 'attribution:code:status',
                onClick: () => handleBatchStatus('enabled')
              },
              {
                title: '停用',
                type: 'warning',
                plain: true,
                disabled: !selections.length,
                permission: 'attribution:code:status',
                onClick: () => handleBatchStatus('disabled')
              },
              {
                preset: 'del',
                disabled: !selections.length,
                permission: 'attribution:code:remove',
                onClick: () => handleBatchRemove()
              }
            ]"
          />
        </template>
        <template #code="{ row }">
          <code-cell :text="row.code" />
        </template>
        <template #status="{ row }">
          <el-tag
            :type="getOptionType(codeStatusOptions, row.effectiveStatus)"
            effect="light"
            size="small"
          >
            {{ getOptionLabel(codeStatusOptions, row.effectiveStatus) }}
          </el-tag>
        </template>
        <template #metrics="{ row }">
          <span class="muted-text">
            访问 {{ row.visitCount }} 人 · 首访 {{ row.firstPassCount }} · 回访
            {{ row.returnCount }} · 转化 {{ row.conversionCount }}
          </span>
        </template>
        <template #expiresAt="{ row }">
          <span v-if="row.expiresAt">{{ row.expiresAt }}</span>
          <span v-else class="muted-text">长期</span>
        </template>
        <template #action="{ row }">
          <el-space :size="6" spacer="|">
            <el-button
              v-permission="'attribution:code:edit'"
              link
              type="primary"
              @click="handleEdit(row)"
              >编辑</el-button
            >
            <el-button
              v-permission="'attribution:link:build'"
              link
              type="primary"
              @click="handleLink(row)"
              >生成链接</el-button
            >
            <el-button
              v-permission="'attribution:code:status'"
              link
              :type="row.status === 'enabled' ? 'info' : 'success'"
              @click="handleStatus(row)"
            >
              {{ row.status === 'enabled' ? '停用' : '启用' }}
            </el-button>
          </el-space>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref, computed, watch, onMounted } from 'vue';
  import CodeCell from '@/components/CodeCell.vue';
  import { useRouter } from 'vue-router';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useCurrentSite } from '@/utils/use-site';
  import {
    pageCodes,
    getCodeStats,
    listCodeChannels,
    updateCodesStatus,
    removeCodes,
    exportCodes
  } from '@/api/merchant/code';
  import CodeSearch from './components/code-search.vue';
  import {
    codeStatusOptions,
    getOptionLabel,
    getOptionType
  } from './constants';

  defineOptions({ name: 'MerchantCode' });

  const router = useRouter();
  const { openModal } = useModal();
  const { sites, siteId, load: loadSites } = useCurrentSite();
  const tableRef = ref(null);
  const searchRef = ref(null);
  const selections = ref([]);
  const channels = ref([]);
  const activeStatus = ref('');
  const stats = ref({
    total: 0,
    enabled: 0,
    disabled: 0,
    expired: 0,
    visits: 0,
    quota: 0,
    used: 0
  });

  const summaryItems = computed(() => [
    {
      key: '',
      label: '全部归因码',
      value: stats.value.total,
      desc: `商户额度 ${stats.value.used} / ${stats.value.quota}`,
      type: 'primary'
    },
    {
      key: 'enabled',
      label: '有效',
      value: stats.value.enabled,
      desc: '访客带码访问可放行',
      type: 'success'
    },
    {
      key: 'disabled',
      label: '已停用',
      value: stats.value.disabled,
      desc: '带该码的访问按无归因处理',
      type: 'info'
    },
    {
      key: 'expired',
      label: '已过期',
      value: stats.value.expired,
      desc: '超过失效时间',
      type: 'warning',
      alert: stats.value.expired > 0
    }
  ]);

  const columns = [
    {
      type: 'selection',
      columnKey: 'selection',
      width: 45,
      align: 'center',
      fixed: 'left'
    },
    {
      prop: 'code',
      label: '码值',
      minWidth: 150,
      slot: 'code',
      formatter: (row) => row.code || '-'
    },
    {
      prop: 'name',
      label: '名称',
      minWidth: 120,
      formatter: (row) => row.name || '-'
    },
    {
      prop: 'channel',
      label: '渠道',
      width: 100,
      formatter: (row) => row.channel || '-'
    },
    {
      prop: 'siteName',
      label: '站点',
      minWidth: 120,
      formatter: (row) => row.siteName || '-'
    },
    {
      prop: 'effectiveStatus',
      label: '状态',
      width: 90,
      align: 'center',
      slot: 'status',
      formatter: (row) => getOptionLabel(codeStatusOptions, row.effectiveStatus)
    },
    { columnKey: 'metrics', label: '访问统计', minWidth: 260, slot: 'metrics' },
    {
      prop: 'expiresAt',
      label: '失效时间',
      minWidth: 150,
      slot: 'expiresAt',
      formatter: (row) => row.expiresAt || '长期'
    },
    {
      prop: 'lastVisitAt',
      label: '最近访问',
      minWidth: 150,
      formatter: (row) => row.lastVisitAt || '-'
    },
    { prop: 'createdAt', label: '登记时间', minWidth: 150 },
    {
      columnKey: 'action',
      label: '操作',
      width: 200,
      align: 'center',
      fixed: 'right',
      slot: 'action',
      hideInPrint: true,
      hideInExport: true
    }
  ];

  const datasource = ({ page, limit, where }) =>
    pageCodes({
      siteId: siteId.value || undefined,
      ...where,
      pageNum: page,
      pageSize: limit
    });

  const loadStats = () => {
    getCodeStats(siteId.value || undefined)
      .then((d) => (stats.value = d))
      .catch(() => {});
  };

  const loadChannels = () => {
    listCodeChannels()
      .then((rows) => (channels.value = rows || []))
      .catch(() => {});
  };

  const reload = (where, page) => {
    tableRef.value?.reload?.({ where, page });
    loadStats();
  };

  const refresh = () => {
    reload(searchRef.value?.where?.());
    loadChannels();
  };

  const handleSummarySelect = (item) => {
    activeStatus.value = item.key;
    searchRef.value?.setStatus(item.key);
  };

  const dialogProps = () => ({
    siteId: siteId.value,
    sites: sites.value,
    channels: channels.value
  });

  const handleAdd = () =>
    openModal({
      custom: true,
      asyncComponent: () => import('./components/code-edit.vue'),
      componentProps: { ...dialogProps(), onDone: refresh }
    });

  const handleEdit = (row) =>
    openModal({
      custom: true,
      asyncComponent: () => import('./components/code-edit.vue'),
      componentProps: { ...dialogProps(), data: row, onDone: refresh }
    });

  const handleGenerate = () =>
    openModal({
      custom: true,
      asyncComponent: () => import('./components/code-generate.vue'),
      componentProps: { ...dialogProps(), onDone: refresh }
    });

  const handleImport = () =>
    openModal({
      custom: true,
      asyncComponent: () => import('./components/code-import.vue'),
      componentProps: { ...dialogProps(), onDone: refresh }
    });

  const handleExport = () => {
    const loading = EleMessage.loading({ message: '导出中..', plain: true });
    exportCodes(searchRef.value?.where?.() || {})
      .then(() => loading.close())
      .catch((e) => {
        loading.close();
        EleMessage.error({ message: e.message, plain: true });
      });
  };

  const handleLink = (row) =>
    router.push({
      path: '/merchant/link',
      query: { siteId: row.siteId, code: row.code }
    });

  const run = (promise) => {
    const loading = EleMessage.loading({ message: '请求中..', plain: true });
    promise
      .then((msg) => {
        loading.close();
        EleMessage.success({ message: msg, plain: true });
        refresh();
      })
      .catch((e) => {
        loading.close();
        EleMessage.error({ message: e.message, plain: true });
      });
  };

  const handleStatus = (row) => {
    const next = row.status === 'enabled' ? 'disabled' : 'enabled';
    ElMessageBox.confirm(
      `是否确认${next === 'enabled' ? '启用' : '停用'}归因码「${row.code}」？`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => run(updateCodesStatus([row.codeId], next)))
      .catch(() => {});
  };

  const handleBatchStatus = (status) => {
    const ids = selections.value.map((r) => r.codeId);
    ElMessageBox.confirm(
      `是否确认${status === 'enabled' ? '启用' : '停用'}选中的 ${ids.length} 个归因码？`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => run(updateCodesStatus(ids, status)))
      .catch(() => {});
  };

  const handleBatchRemove = () => {
    const ids = selections.value.map((r) => r.codeId);
    ElMessageBox.confirm(
      `删除后不可恢复，是否确认删除选中的 ${ids.length} 个归因码？`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => run(removeCodes(ids)))
      .catch(() => {});
  };

  watch(siteId, () => reload(searchRef.value?.where?.(), 1));

  onMounted(() => {
    loadSites().then(() => {
      loadStats();
      loadChannels();
    });
  });
</script>
