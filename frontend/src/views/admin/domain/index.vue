<template>
  <ele-page flex-table="auto">
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        ref="tableRef"
        row-key="domainId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        cache-key="adminDomainTable"
      >
        <template #status="{ row }">
          <el-tag :type="statusType[row.status] || 'info'">
            {{ statusText[row.status] || row.status }}
          </el-tag>
        </template>
        <template #action="{ row }">
          <btn-items
            divider
            type="link"
            :items="[
              {
                title: '强制停用',
                permission: 'cloak:domain:list',
                vIf: () => row.status !== 'disabled',
                onClick: () => handleForceDisable(row)
              },
              {
                preset: 'detail',
                permission: 'cloak:domain:list',
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
    pageAdminDomains,
    updateAdminDomainStatus
  } from '@/api/admin/domain';

  defineOptions({ name: 'AdminDomain' });

  const { openModal } = useModal();
  const tableRef = ref(null);

  const statusText = {
    pending: '待验证',
    bound: '已绑定',
    abnormal: '异常',
    disabled: '已停用'
  };

  const statusType = {
    pending: 'warning',
    bound: 'success',
    abnormal: 'danger',
    disabled: 'info'
  };

  const columns = [
    { type: 'index', columnKey: 'index', width: 55, align: 'center' },
    { prop: 'domain', label: '域名', minWidth: 180 },
    { prop: 'merchantName', label: '所属商户', minWidth: 160 },
    { prop: 'cloakCount', label: '关联斗篷', width: 110, align: 'center' },
    { prop: 'status', label: '状态', width: 110, align: 'center', slot: 'status' },
    { prop: 'lastCheckedAt', label: '最后检测', minWidth: 160, align: 'center' },
    { columnKey: 'action', label: '操作', width: 170, align: 'center', slot: 'action' }
  ];

  const datasource = ({ page, limit, where }) => {
    return pageAdminDomains({ ...where, pageNum: page, pageSize: limit });
  };

  const reload = () => {
    tableRef.value?.reload?.();
  };

  const handleForceDisable = (row) => {
    ElMessageBox.confirm(
      `是否确认强制停用域名"${row.domain}"?`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        updateAdminDomainStatus(row.domainId, 'disabled')
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
      asyncComponent: () => import('./components/domain-detail.vue'),
      componentProps: {
        data: row,
        statusText,
        statusType
      }
    });
  };
</script>
