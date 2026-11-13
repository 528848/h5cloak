<template>
  <ele-page flex-table="auto">
    <summary-cards
      :items="summaryItems"
      :cols="4"
      clickable
      :active-key="activeStatus"
      @select="handleSummarySelect"
    />
    <site-search ref="searchRef" @search="(where) => reload(where, 1)" />
    <ele-card
      class="table-section"
      :body-style="{ paddingTop: '8px' }"
      flex-table="auto"
    >
      <ele-pro-table
        ref="tableRef"
        row-key="siteId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        cache-key="merchantSiteTable"
      >
        <template #toolbar>
          <btn-items
            :items="[
              {
                preset: 'add',
                title: '新建站点',
                permission: 'site:site:add',
                onClick: () => handleAdd()
              }
            ]"
          />
        </template>
        <template #name="{ row }">
          <el-link type="primary" underline="never" @click="handleEdit(row)">
            {{ row.name }}
          </el-link>
        </template>
        <template #siteType="{ row }">
          <el-tag
            :type="getOptionType(siteTypeOptions, row.siteType)"
            effect="plain"
            size="small"
          >
            {{ getOptionLabel(siteTypeOptions, row.siteType) }}
          </el-tag>
        </template>
        <template #status="{ row }">
          <el-tag
            :type="getOptionType(siteStatusOptions, row.status)"
            effect="light"
            size="small"
          >
            {{ getOptionLabel(siteStatusOptions, row.status) }}
          </el-tag>
          <el-tag
            v-if="row.platformHold"
            type="danger"
            effect="plain"
            size="small"
            style="margin-left: 6px"
          >
            平台停用
          </el-tag>
        </template>
        <template #domains="{ row }">
          {{ row.boundDomainCount || 0 }} / {{ row.domainCount || 0 }}
        </template>
        <template #origin="{ row }">
          <el-tag
            v-if="row.unhealthyOriginCount > 0"
            type="danger"
            effect="light"
            size="small"
          >
            {{ row.unhealthyOriginCount }} 个不可达
          </el-tag>
          <el-tag
            v-else-if="row.originCount > 0"
            type="success"
            effect="plain"
            size="small"
          >
            正常
          </el-tag>
          <span v-else class="muted-text">未配置</span>
        </template>
        <template #publishableKey="{ row }">
          <code-cell :text="row.publishableKey" />
        </template>
        <template #action="{ row }">
          <el-space :size="6" spacer="|">
            <el-button link type="primary" @click="handleEdit(row)"
              >配置</el-button
            >
            <el-button
              v-permission="'site:site:status'"
              link
              :type="row.status === 'enabled' ? 'info' : 'success'"
              @click="handleStatus(row)"
            >
              {{ row.status === 'enabled' ? '停用' : '启用' }}
            </el-button>
            <el-button
              v-permission="'site:site:remove'"
              link
              type="danger"
              @click="handleRemove(row)"
            >
              删除
            </el-button>
          </el-space>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref, computed, onMounted, onActivated } from 'vue';
  import CodeCell from '@/components/CodeCell.vue';
  import { useRouter } from 'vue-router';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage } from 'ele-admin-plus';
  import {
    pageSites,
    getSiteStats,
    updateSiteStatus,
    removeSite
  } from '@/api/merchant/site';
  import SiteSearch from './components/site-search.vue';
  import {
    siteStatusOptions,
    siteTypeOptions,
    getOptionLabel,
    getOptionType
  } from './constants';

  defineOptions({ name: 'MerchantSite' });

  const router = useRouter();
  const tableRef = ref(null);
  const searchRef = ref(null);
  const activeStatus = ref('');
  const stats = ref({
    total: 0,
    enabled: 0,
    disabled: 0,
    domainAbnormal: 0
  });

  const summaryItems = computed(() => [
    {
      key: '',
      label: '全部站点',
      value: stats.value.total,
      desc: '当前商户已创建的站点',
      type: 'primary'
    },
    {
      key: 'enabled',
      label: '启用中',
      value: stats.value.enabled,
      desc: '可进行访问归因',
      type: 'success'
    },
    {
      key: 'disabled',
      label: '已停用',
      value: stats.value.disabled,
      desc: '停止判定与回源',
      type: 'info'
    },
    {
      key: 'domainAbnormal',
      label: '域名异常',
      value: stats.value.domainAbnormal,
      desc: '解析与期望不一致',
      type: 'danger',
      alert: stats.value.domainAbnormal > 0,
      clickable: false
    }
  ]);

  const columns = [
    {
      type: 'index',
      columnKey: 'index',
      width: 50,
      align: 'center',
      fixed: 'left'
    },
    {
      prop: 'name',
      label: '站点名称',
      minWidth: 160,
      slot: 'name',
      formatter: (row) => row.name
    },
    {
      prop: 'siteType',
      label: '接入方式',
      width: 110,
      align: 'center',
      slot: 'siteType',
      formatter: (row) => getOptionLabel(siteTypeOptions, row.siteType)
    },
    {
      prop: 'status',
      label: '状态',
      width: 150,
      align: 'center',
      slot: 'status',
      formatter: (row) => getOptionLabel(siteStatusOptions, row.status)
    },
    {
      columnKey: 'domains',
      label: '域名（已绑定/全部）',
      width: 160,
      align: 'center',
      slot: 'domains',
      formatter: (row) =>
        `${row.boundDomainCount || 0} / ${row.domainCount || 0}`
    },
    {
      columnKey: 'origin',
      label: '源站',
      width: 120,
      align: 'center',
      slot: 'origin'
    },
    {
      prop: 'publishableKey',
      label: '公钥',
      minWidth: 180,
      slot: 'publishableKey',
      formatter: (row) => row.publishableKey
    },
    { prop: 'createdAt', label: '创建时间', minWidth: 160 },
    {
      columnKey: 'action',
      label: '操作',
      width: 180,
      align: 'center',
      fixed: 'right',
      slot: 'action',
      hideInPrint: true,
      hideInExport: true
    }
  ];

  const datasource = ({ page, limit, where }) =>
    pageSites({ ...where, pageNum: page, pageSize: limit });

  const loadStats = () => {
    getSiteStats()
      .then((data) => (stats.value = data))
      .catch(() => {});
  };

  const reload = (where, page) => {
    tableRef.value?.reload?.({ where, page });
    loadStats();
  };

  const handleSummarySelect = (item) => {
    if (item.clickable === false) return;
    activeStatus.value = item.key;
    searchRef.value?.setStatus(item.key);
  };

  const handleAdd = () => router.push('/merchant/site/form');

  const handleEdit = (row) =>
    router.push({ path: '/merchant/site/form', query: { id: row.siteId } });

  const run = (promise) => {
    const loading = EleMessage.loading({ message: '请求中..', plain: true });
    promise
      .then((msg) => {
        loading.close();
        EleMessage.success({ message: msg, plain: true });
        reload();
      })
      .catch((e) => {
        loading.close();
        EleMessage.error({ message: e.message, plain: true });
      });
  };

  const handleStatus = (row) => {
    const next = row.status === 'enabled' ? 'disabled' : 'enabled';
    const text = next === 'enabled' ? '启用' : '停用';
    ElMessageBox.confirm(`是否确认${text}站点「${row.name}」？`, '系统提示', {
      type: 'warning',
      draggable: true
    })
      .then(() => run(updateSiteStatus(row.siteId, next)))
      .catch(() => {});
  };

  const handleRemove = (row) => {
    ElMessageBox.confirm(
      `删除后域名与源站配置一并清除，是否确认删除站点「${row.name}」？`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => run(removeSite(row.siteId)))
      .catch(() => {});
  };

  onMounted(loadStats);

  // 列表被页签缓存，从新建/编辑页返回时不会重新挂载，这里补一次刷新。
  let skipActivate = true;
  onActivated(() => {
    if (skipActivate) {
      skipActivate = false;
      return;
    }
    reload();
  });
</script>
