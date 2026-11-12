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
      >
        <template #action="{ row }">
          <el-button link type="primary" @click="handleDetail(row)">
            详情
          </el-button>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { useModal } from 'ele-admin-plus';
  import { pageAdminApiLogs } from '@/api/admin/api-log';

  defineOptions({ name: 'AdminApiLog' });

  const { openModal } = useModal();

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
    { prop: 'createdAt', label: '调用时间', minWidth: 160, align: 'center' },
    {
      columnKey: 'action',
      label: '操作',
      width: 80,
      align: 'center',
      fixed: 'right',
      slot: 'action',
      hideInPrint: true,
      hideInExport: true
    }
  ];

  const handleDetail = (row) =>
    openModal({
      custom: true,
      asyncComponent: () =>
        import('@/views/merchant/api-log/components/log-detail.vue'),
      componentProps: { id: row.id, portal: 'admin' }
    });

  const datasource = ({ page, limit, where }) => {
    return pageAdminApiLogs({ ...where, pageNum: page, pageSize: limit });
  };
</script>
