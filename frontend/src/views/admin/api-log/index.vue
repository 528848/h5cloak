<template>
  <ele-page flex-table="auto">
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        row-key="id"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        cache-key="adminApiLogTable"
      />
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { pageAdminApiLogs } from '@/api/admin/api-log';

  defineOptions({ name: 'AdminApiLog' });

  const columns = [
    { type: 'index', columnKey: 'index', width: 55, align: 'center' },
    { prop: 'requestId', label: '请求 ID', minWidth: 180 },
    { prop: 'merchantName', label: '商户', minWidth: 120 },
    { prop: 'siteName', label: '站点', minWidth: 120 },
    { prop: 'endpoint', label: '接口', minWidth: 160 },
    { prop: 'status', label: '状态码', width: 90, align: 'center' },
    { prop: 'errorCode', label: '错误码', minWidth: 120 },
    { prop: 'latencyMs', label: '耗时(ms)', width: 110, align: 'center' },
    { prop: 'ip', label: 'IP', minWidth: 130 },
    { prop: 'createdAt', label: '调用时间', minWidth: 160, align: 'center' }
  ];

  const datasource = ({ page, limit, where }) => {
    return pageAdminApiLogs({ ...where, pageNum: page, pageSize: limit });
  };
</script>
