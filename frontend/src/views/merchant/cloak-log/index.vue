<template>
  <ele-page flex-table="auto">
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        row-key="logId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        cache-key="merchantCloakLogTable"
      >
        <template #finalAction="{ row }">
          <el-tag>{{ actionText[row.finalAction] || row.finalAction }}</el-tag>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { pageMerchantCloakLogs } from '@/api/merchant/cloak';

  defineOptions({ name: 'MerchantCloakLog' });

  const actionText = {
    white: '白页',
    black: '黑页',
    block: '拦截',
    not_found: '404'
  };

  const columns = [
    { type: 'index', columnKey: 'index', width: 55, align: 'center' },
    { prop: 'requestId', label: '请求 ID', minWidth: 180 },
    { prop: 'domain', label: '域名', minWidth: 180 },
    { prop: 'cloakName', label: '斗篷', minWidth: 150 },
    { prop: 'ipSummary', label: 'IP 摘要', minWidth: 130 },
    { prop: 'country', label: '国家', width: 100, align: 'center' },
    { prop: 'deviceType', label: '设备', width: 100, align: 'center' },
    { prop: 'finalAction', label: '最终动作', width: 110, align: 'center', slot: 'finalAction' },
    { prop: 'createdAt', label: '访问时间', minWidth: 160, align: 'center' }
  ];

  const datasource = ({ page, limit, where }) => {
    return pageMerchantCloakLogs({ ...where, pageNum: page, pageSize: limit });
  };
</script>
