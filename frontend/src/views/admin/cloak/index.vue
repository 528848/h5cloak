<template>
  <ele-page flex-table="auto">
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        ref="tableRef"
        row-key="cloakId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        cache-key="adminCloakTable"
      >
        <template #status="{ row }">
          <el-tag :type="row.status === 'enabled' ? 'success' : 'info'">
            {{ row.status === 'enabled' ? '启用' : '停用' }}
          </el-tag>
        </template>
        <template #action="{ row }">
          <btn-items
            divider
            type="link"
            :items="[
              {
                title: '强制停用',
                permission: 'cloak:strategy:status',
                vIf: () => row.status === 'enabled',
                onClick: () => handleForceDisable(row)
              },
              {
                preset: 'detail',
                permission: 'cloak:strategy:list',
                onClick: () => handleDetail(row)
              }
            ]"
          />
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref } from 'vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import {
    pageAdminCloaks,
    updateAdminCloakStatus
  } from '@/api/admin/cloak';

  defineOptions({ name: 'AdminCloak' });

  const { openModal } = useModal();
  const tableRef = ref(null);

  const columns = [
    { type: 'index', columnKey: 'index', width: 55, align: 'center' },
    { prop: 'cloakName', label: '斗篷名称', minWidth: 160 },
    { prop: 'domain', label: '关联域名', minWidth: 180 },
    { prop: 'merchantName', label: '所属商户', minWidth: 160 },
    { prop: 'todayVisits', label: '今日访问', width: 110, align: 'center' },
    { prop: 'status', label: '状态', width: 100, align: 'center', slot: 'status' },
    { prop: 'updatedAt', label: '更新时间', minWidth: 160, align: 'center' },
    { columnKey: 'action', label: '操作', width: 170, align: 'center', slot: 'action' }
  ];

  const datasource = ({ page, limit, where }) => {
    return pageAdminCloaks({ ...where, pageNum: page, pageSize: limit });
  };

  const reload = () => {
    tableRef.value?.reload?.();
  };

  const handleForceDisable = (row) => {
    ElMessageBox.confirm(
      `是否确认强制停用斗篷"${row.cloakName}"?`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        updateAdminCloakStatus(row.cloakId, 'disabled')
          .then((msg) => {
            loading.close();
            EleMessage.success({ message: msg, plain: true });
            reload();
          })
          .catch((e) => {
            loading.close();
            EleMessage.error({ message: e.message, plain: true });
          });
      })
      .catch(() => {});
  };

  const handleDetail = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/cloak-detail.vue'),
      componentProps: { data: row }
    });
  };
</script>
