<template>
  <ele-page flex-table="auto">
    <summary-cards
      :items="summaryItems"
      :cols="4"
      clickable
      :active-key="form.status"
      @select="selectStatus"
    />
    <ele-card search-form class="page-section">
      <el-form label-width="72px" @keyup.enter="handleSearch" @submit.prevent="">
        <el-row :gutter="12">
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="类型">
              <el-select v-model="form.kind" clearable placeholder="全部" class="ele-fluid">
                <el-option v-for="o in kindOptions" :key="o.value" :label="o.label" :value="o.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="状态">
              <el-select v-model="form.status" clearable placeholder="全部" class="ele-fluid">
                <el-option v-for="o in statusOptions" :key="o.value" :label="o.label" :value="o.value" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :lg="8" :md="12" :sm="24" :xs="24">
            <el-form-item label="创建时间">
              <el-date-picker
                v-model="form.dates"
                type="daterange"
                value-format="YYYY-MM-DD"
                start-placeholder="开始日期"
                end-placeholder="结束日期"
                range-separator="-"
                class="ele-fluid"
              />
            </el-form-item>
          </el-col>
          <el-col :lg="4" :md="8" :sm="12" :xs="24">
            <el-form-item label-width="0">
              <btn-items
                :wrap="false"
                :items="[
                  { preset: 'search', onClick: () => handleSearch() },
                  { preset: 'reset', onClick: () => handleReset() }
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
        cache-key="merchantDownloadCenter"
      >
        <template #id="{ row }"><span class="mono">{{ row.id }}</span></template>
        <template #status="{ row }">
          <el-tag :type="statusType(row)" effect="light" size="small">{{ statusLabel(row) }}</el-tag>
          <span v-if="row.error" class="error-text">{{ row.error }}</span>
        </template>
        <template #rowCount="{ row }"><span class="mono">{{ row.rowCount || '-' }}</span></template>
        <template #action="{ row }">
          <el-button v-if="row.downloadToken" link type="primary" @click="download(row)">下载</el-button>
          <el-button
            v-else-if="row.status === 'failed'"
            v-permission="'report:export'"
            link
            type="primary"
            @click="retry(row)"
          >重试</el-button>
          <el-button
            v-else-if="row.status === 'pending'"
            v-permission="'report:export'"
            link
            type="danger"
            @click="cancel(row)"
          >取消</el-button>
          <span v-else class="muted-text">{{ row.expired ? '文件已清理' : '-' }}</span>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { computed, onMounted, onUnmounted, ref } from 'vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import {
    cancelExport,
    exportDownloadUrl,
    getExportStats,
    pageExports,
    retryExport
  } from '@/api/merchant/ops';

  defineOptions({ name: 'MerchantDownload' });

  const kindOptions = [
    { value: 'decision_logs', label: '决策日志' },
    { value: 'report_channel', label: '渠道报表' },
    { value: 'report_code', label: '归因码报表' },
    { value: 'report_campaign', label: '活动码报表（已取消）' },
    { value: 'report_path', label: '路径报表' },
    { value: 'report_conversion', label: '转化报表' }
  ];
  const statusOptions = [
    { value: 'pending', label: '排队中' },
    { value: 'running', label: '生成中' },
    { value: 'done', label: '可下载' },
    { value: 'expired', label: '已过期' },
    { value: 'failed', label: '失败' },
    { value: 'cancelled', label: '已取消' }
  ];

  const tableRef = ref(null);
  const stats = ref({ active: 0, ready: 0, expired: 0, failed: 0 });
  const [form, resetFields] = useFormData({ kind: '', status: '', dates: [] });
  let timer;

  const summaryItems = computed(() => [
    { key: 'active', label: '进行中', value: stats.value.active, desc: '排队或正在生成', type: 'primary' },
    { key: 'done', label: '可下载', value: stats.value.ready, desc: '24 小时内有效', type: 'success' },
    { key: 'expired', label: '已过期', value: stats.value.expired, desc: '文件已清理', type: 'info' },
    { key: 'failed', label: '失败', value: stats.value.failed, desc: '可以重试', type: 'danger', alert: stats.value.failed > 0 }
  ]);

  const columns = [
    { prop: 'id', label: '任务编号', width: 100, align: 'center', slot: 'id', formatter: (row) => row.id },
    { prop: 'kindName', label: '类型', width: 120 },
    { prop: 'scopeText', label: '范围', minWidth: 180 },
    {
      prop: 'status',
      label: '状态',
      minWidth: 160,
      slot: 'status',
      formatter: (row) => statusLabel(row)
    },
    { prop: 'rowCount', label: '行数', width: 90, align: 'right', slot: 'rowCount', formatter: (row) => row.rowCount },
    {
      prop: 'fileSize',
      label: '大小',
      width: 100,
      align: 'right',
      formatter: (row) => (row.fileSize ? `${Math.ceil(row.fileSize / 1024)} KB` : '-')
    },
    { prop: 'createdAt', label: '创建时间', minWidth: 160 },
    { prop: 'expiresAt', label: '过期时间', minWidth: 160, formatter: (row) => row.expiresAt || '-' },
    {
      columnKey: 'action',
      label: '操作',
      width: 90,
      align: 'center',
      fixed: 'right',
      slot: 'action',
      hideInPrint: true,
      hideInExport: true
    }
  ];

  const statusLabel = (row) => {
    if (row.expired) return '已过期';
    return { pending: '排队中', running: '生成中', done: '可下载', failed: '失败', cancelled: '已取消' }[row.status] || row.status;
  };
  const statusType = (row) => {
    if (row.expired) return 'info';
    return { pending: 'info', running: 'primary', done: 'success', failed: 'danger', cancelled: 'info' }[row.status] || 'info';
  };

  const where = () => ({
    kind: form.kind,
    status: form.status,
    from: form.dates?.[0],
    to: form.dates?.[1]
  });

  const loadStats = () =>
    getExportStats()
      .then((d) => {
        stats.value = d || stats.value;
        if ((d?.active || 0) === 0) stopPoll();
        else startPoll();
      })
      .catch(() => {});

  const datasource = ({ page, limit, where: w }) =>
    pageExports({ ...w, pageNum: page, pageSize: limit });

  const reload = (page) => {
    tableRef.value?.reload?.({ where: where(), page });
    loadStats();
  };
  const handleSearch = () => reload(1);
  const handleReset = () => {
    resetFields();
    handleSearch();
  };
  const selectStatus = (item) => {
    form.status = form.status === item.key ? '' : item.key;
    handleSearch();
  };

  const download = (row) => window.open(exportDownloadUrl(row.downloadToken), '_blank');
  const retry = (row) => {
    ElMessageBox.confirm('将按原来的条件重新生成一份文件。是否确认重试？', '系统提示', { type: 'warning', draggable: true })
      .then(() => retryExport(row.id))
      .then(() => {
        EleMessage.success({ message: '已重新加入下载中心', plain: true });
        reload();
      })
      .catch((e) => { if (e?.message) EleMessage.error({ message: e.message, plain: true }); });
  };
  const cancel = (row) => {
    ElMessageBox.confirm('取消后该任务不会再生成文件。是否确认取消？', '系统提示', { type: 'warning', draggable: true })
      .then(() => cancelExport(row.id))
      .then(() => {
        EleMessage.success({ message: '已取消', plain: true });
        reload();
      })
      .catch((e) => { if (e?.message) EleMessage.error({ message: e.message, plain: true }); });
  };

  const startPoll = () => {
    if (timer) return;
    timer = setInterval(() => reload(), 5000);
  };
  const stopPoll = () => {
    if (timer) clearInterval(timer);
    timer = null;
  };

  onMounted(loadStats);
  onUnmounted(stopPoll);
</script>

<style scoped>
  .error-text {
    margin-left: 6px;
    color: var(--el-color-danger);
    font-size: 12px;
  }
</style>
