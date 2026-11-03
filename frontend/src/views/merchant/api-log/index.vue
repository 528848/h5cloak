<template>
  <ele-page flex-table="auto">
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        row-key="logId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        cache-key="merchantApiLogTable"
      />
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { pageMerchantApiLogs } from '@/api/merchant/api-log';

  defineOptions({ name: 'MerchantApiLog' });

  const columns = [
    { type: 'index', columnKey: 'index', width: 55, align: 'center' },
    { prop: 'requestId', label: '请求 ID', minWidth: 180 },
    { prop: 'apiKeyName', label: 'API Key', minWidth: 140 },
    { prop: 'apiName', label: '接口名称', minWidth: 160 },
    { prop: 'method', label: '方法', width: 90, align: 'center' },
    { prop: 'path', label: '请求路径', minWidth: 180 },
    { prop: 'statusCode', label: '状态码', width: 100, align: 'center' },
    { prop: 'latencyMs', label: '耗时(ms)', width: 110, align: 'center' },
    { prop: 'createdAt', label: '调用时间', minWidth: 160, align: 'center' }
  ];

  const datasource = ({ page, limit, where }) => {
    return pageMerchantApiLogs({ ...where, pageNum: page, pageSize: limit });
  };
</script>
