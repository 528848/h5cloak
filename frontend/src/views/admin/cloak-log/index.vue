<template>
  <ele-page flex-table="auto">
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        row-key="logId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        cache-key="adminCloakLogTable"
      >
        <template #finalAction="{ row }">
          <el-tag>{{ actionText[row.finalAction] || row.finalAction }}</el-tag>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { pageAdminCloakLogs } from '@/api/admin/cloak';

  defineOptions({ name: 'AdminCloakLog' });

  const actionText = {
    white: '白页',
    black: '黑页',
    block: '拦截',
    not_found: '404'
  };

  const columns = [
    { type: 'index', columnKey: 'index', width: 55, align: 'center' },
    { prop: 'requestId', label: '请求 ID', minWidth: 180 },
    { prop: 'merchantName', label: '商户', minWidth: 140 },
    { prop: 'domain', label: '域名', minWidth: 180 },
    { prop: 'cloakName', label: '斗篷', minWidth: 150 },
    { prop: 'ipSummary', label: 'IP 摘要', minWidth: 130 },
    {
      prop: 'finalAction',
      label: '最终动作',
      width: 110,
      align: 'center',
      slot: 'finalAction'
    },
    { prop: 'latencyMs', label: '耗时(ms)', width: 110, align: 'center' },
    { prop: 'createdAt', label: '访问时间', minWidth: 160, align: 'center' }
  ];

  const datasource = ({ page, limit, where }) => {
    return pageAdminCloakLogs({ ...where, pageNum: page, pageSize: limit });
  };
</script>
