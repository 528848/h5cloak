<template>
  <ele-page>
    <summary-cards
      :items="kpis"
      :cols="4"
      clickable
      :active-key="activeKey"
      @select="selectKpi"
    />
    <ele-card search-form class="page-section">
      <el-form label-width="72px" @keyup.enter="search" @submit.prevent="">
        <el-row :gutter="12">
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="状态">
              <el-select v-model="form.status" clearable placeholder="全部" class="ele-fluid">
                <el-option
                  v-for="item in alertStatusOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="类型">
              <el-select v-model="form.type" clearable placeholder="全部" class="ele-fluid">
                <el-option v-for="item in defaults" :key="item.type" :label="item.name" :value="item.type" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="级别">
              <el-select v-model="form.severity" clearable placeholder="全部" class="ele-fluid">
                <el-option
                  v-for="item in severityOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
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
    <ele-card class="table-section" :body-style="{ paddingTop: '8px' }">
      <ele-pro-table
        ref="tableRef"
        row-key="id"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        cache-key="merchantAlertTable"
        :page-size="20"
      >
        <template #severity="{ row }">
          <el-tag :type="severityType(row.severity)" size="small" effect="dark">
            {{ severityLabel(row.severity) }}
          </el-tag>
        </template>
        <template #title="{ row }">
          <el-link type="primary" underline="never" @click="showDetail(row)">{{ row.title }}</el-link>
          <div class="alert-msg">{{ row.message }}</div>
        </template>
        <template #status="{ row }">
          <el-tag :type="statusType(row.status)" size="small" effect="light">
            {{ statusLabel(row.status) }}
          </el-tag>
        </template>
        <template #notified="{ row }">
          <el-space :size="4" wrap>
            <el-tag
              v-for="(value, key) in row.notified || {}"
              :key="key"
              size="small"
              :type="notifyType(value)"
              effect="plain"
            >
              {{ channelLabel(key) }}
            </el-tag>
          </el-space>
        </template>
        <template #action="{ row }">
          <el-space :size="6" spacer="|">
            <el-button link type="primary" @click="showDetail(row)">详情</el-button>
            <el-button
              v-if="row.status === 'open'"
              v-permission="'alert:edit'"
              link
              type="warning"
              @click="resolve(row)"
            >关闭</el-button>
          </el-space>
        </template>
      </ele-pro-table>
    </ele-card>

    <ele-modal v-model="detailVisible" title="告警详情" :width="640">
      <el-descriptions v-if="detail" :column="1" border class="detail-table">
        <el-descriptions-item label="类型">{{ detail.typeName || '-' }}</el-descriptions-item>
        <el-descriptions-item label="站点">{{ detail.siteName || '全部' }}</el-descriptions-item>
        <el-descriptions-item label="目标">
          <span class="mono">{{ detail.targetKey || '-' }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="级别">{{ severityLabel(detail.severity) }}</el-descriptions-item>
        <el-descriptions-item label="状态">{{ statusLabel(detail.status) }}</el-descriptions-item>
        <el-descriptions-item label="首次触发">{{ detail.firedAt || '-' }}</el-descriptions-item>
        <el-descriptions-item label="最近出现">{{ detail.lastSeenAt || '-' }}</el-descriptions-item>
        <el-descriptions-item v-if="detail.resolvedAt" label="恢复时间">{{ detail.resolvedAt }}</el-descriptions-item>
        <el-descriptions-item label="说明">{{ detail.message || '-' }}</el-descriptions-item>
        <el-descriptions-item label="通知结果">{{ notifiedText(detail.notified) }}</el-descriptions-item>
      </el-descriptions>
      <pre v-if="detail" class="code-block mono">{{ JSON.stringify(detail.detail || {}, null, 2) }}</pre>
    </ele-modal>
  </ele-page>
</template>

<script setup>
  import { computed, onMounted, ref } from 'vue';
  import { useRoute } from 'vue-router';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { getAlertRules, getAlertStats, pageAlerts, resolveAlert } from '@/api/merchant/ops';
  import {
    alertStatusOptions,
    channelLabel,
    severityLabel,
    severityOptions,
    severityType,
    statusLabel,
    statusType
  } from './constants';

  defineOptions({ name: 'MerchantAlert' });

  const route = useRoute();
  const tableRef = ref(null);
  const stats = ref({});
  const defaults = ref([]);
  const activeKey = ref('open');
  const detail = ref(null);
  const detailVisible = ref(false);
  const [form, resetFields] = useFormData({ status: 'open', type: '', severity: '' });

  const kpis = computed(() => [
    {
      key: 'open',
      label: '未恢复',
      value: stats.value.open || 0,
      desc: '仍需处理',
      type: 'danger',
      alert: (stats.value.open || 0) > 0
    },
    {
      key: 'critical',
      label: '严重',
      value: stats.value['open:critical'] || 0,
      desc: '未恢复中的严重告警',
      type: 'danger',
      alert: (stats.value['open:critical'] || 0) > 0
    },
    {
      key: 'warning',
      label: '警告',
      value: stats.value['open:warning'] || 0,
      desc: '未恢复中的警告',
      type: 'warning'
    },
    {
      key: 'resolved',
      label: '已恢复',
      value: stats.value.resolved || 0,
      desc: '问题消失或已手动关闭',
      type: 'success'
    }
  ]);

  const columns = [
    {
      prop: 'severity',
      label: '级别',
      width: 90,
      align: 'center',
      slot: 'severity',
      formatter: (row) => severityLabel(row.severity)
    },
    {
      prop: 'title',
      label: '告警',
      minWidth: 280,
      slot: 'title',
      formatter: (row) => row.title
    },
    { prop: 'siteName', label: '站点', minWidth: 140, formatter: (row) => row.siteName || '全部' },
    {
      prop: 'status',
      label: '状态',
      width: 100,
      align: 'center',
      slot: 'status',
      formatter: (row) => statusLabel(row.status)
    },
    { prop: 'firedAt', label: '触发时间', minWidth: 170 },
    { prop: 'lastSeenAt', label: '最近出现', minWidth: 170 },
    {
      prop: 'notified',
      label: '通知',
      minWidth: 180,
      slot: 'notified',
      formatter: (row) => notifiedText(row.notified)
    },
    {
      columnKey: 'action',
      label: '操作',
      width: 120,
      align: 'center',
      slot: 'action',
      fixed: 'right',
      hideInPrint: true,
      hideInExport: true
    }
  ];

  const datasource = ({ page, limit }) =>
    pageAlerts({ ...form, pageNum: page, pageSize: limit });

  const loadStats = () =>
    getAlertStats()
      .then((data) => (stats.value = data || {}))
      .catch(() => {});

  const reload = () => {
    tableRef.value?.reload?.({ page: 1 });
    loadStats();
  };

  const syncActive = () => {
    if (form.status === 'open' && !form.type && !form.severity) activeKey.value = 'open';
    else if (form.status === 'resolved' && !form.type && !form.severity) activeKey.value = 'resolved';
    else if (form.status === 'open' && !form.type && (form.severity === 'critical' || form.severity === 'warning')) {
      activeKey.value = form.severity;
    } else activeKey.value = '';
  };

  const search = () => {
    syncActive();
    reload();
  };

  const reset = () => {
    resetFields();
    form.status = '';
    activeKey.value = '';
    reload();
  };

  const selectKpi = (item) => {
    activeKey.value = item.key;
    form.type = '';
    if (item.key === 'open') {
      form.status = 'open';
      form.severity = '';
    } else if (item.key === 'resolved') {
      form.status = 'resolved';
      form.severity = '';
    } else {
      form.status = 'open';
      form.severity = item.key;
    }
    reload();
  };

  const notifyType = (value) => {
    const text = String(value || '');
    if (text === 'ok') return 'success';
    if (text.startsWith('skipped')) return 'info';
    return 'danger';
  };

  const notifiedText = (notified) => {
    const text = Object.entries(notified || {})
      .map(([key, value]) => `${channelLabel(key)}：${value}`)
      .join('；');
    return text || '-';
  };

  const showDetail = (row) => {
    detail.value = row;
    detailVisible.value = true;
  };

  const resolve = (row) => {
    ElMessageBox.confirm(
      '确定手动关闭该告警？若问题仍存在且过了冷却时间会再次触发。',
      '关闭告警',
      { type: 'warning', draggable: true }
    )
      .then(() => resolveAlert(row.id))
      .then(() => {
        EleMessage.success({ message: '已关闭', plain: true });
        reload();
      })
      .catch(() => {});
  };

  onMounted(() => {
    loadStats();
    getAlertRules()
      .then((data) => (defaults.value = data?.defaults || []))
      .catch(() => {});
    if (route.query.id) {
      form.status = '';
      form.type = '';
      form.severity = '';
      activeKey.value = '';
      reload();
    }
  });
</script>

<style scoped>
  .alert-msg {
    margin-top: 4px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
    white-space: normal;
  }

  .code-block {
    margin: 12px 0 0;
    padding: 10px 12px;
    border-radius: 8px;
    background: var(--el-fill-color-light);
    font-size: 12px;
    max-height: 220px;
    overflow: auto;
  }
</style>
