<template>
  <ele-page flex-table="auto">
    <access-list-search @search="(where) => reload(where, 1)" />
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        ref="tableRef"
        row-key="listId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        cache-key="merchantAccessListTable"
      >
        <template #toolbar>
          <btn-items
            :items="[
              {
                preset: 'add',
                permission: 'cloak:accessList:add',
                onClick: () => handleAdd()
              }
            ]"
          />
        </template>
        <template #listType="{ row }">
          <el-tag
            :type="row.listType === 'white' ? 'success' : 'danger'"
            size="small"
          >
            {{ row.listType === 'white' ? '白名单' : '黑名单' }}
          </el-tag>
        </template>
        <template #targetType="{ row }">
          {{ targetTypeText[row.targetType] || row.targetType }}
        </template>
        <template #scope="{ row }">
          {{ formatScope(row) }}
        </template>
        <template #status="{ row }">
          <el-tag
            :type="row.status === 'enabled' ? 'success' : 'info'"
            size="small"
          >
            {{ row.status === 'enabled' ? '启用' : '停用' }}
          </el-tag>
        </template>
        <template #action="{ row }">
          <btn-items
            divider
            type="link"
            :items="[
              {
                preset: 'edit',
                permission: 'cloak:accessList:edit',
                onClick: () => handleEdit(row)
              },
              {
                title: row.status === 'enabled' ? '停用' : '启用',
                permission: 'cloak:accessList:edit',
                onClick: () => handleStatus(row)
              },
              {
                preset: 'del',
                permission: 'cloak:accessList:remove',
                onClick: () => handleRemove(row)
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
    pageAccessLists,
    removeAccessList,
    updateAccessListStatus
  } from '@/api/merchant/access-list';
  import AccessListSearch from './components/access-list-search.vue';

  defineOptions({ name: 'MerchantAccessList' });

  const { openModal } = useModal();

  const tableRef = ref(null);

  const targetTypeText = {
    ip: 'IP',
    ua: 'User-Agent',
    referer: 'Referer'
  };

  const columns = [
    { type: 'index', columnKey: 'index', width: 55, align: 'center' },
    {
      prop: 'listType',
      label: '名单类型',
      width: 110,
      align: 'center',
      slot: 'listType'
    },
    {
      prop: 'targetType',
      label: '目标类型',
      width: 120,
      align: 'center',
      slot: 'targetType'
    },
    { prop: 'targetValue', label: '匹配值', minWidth: 180 },
    {
      prop: 'scope',
      label: '作用范围',
      minWidth: 140,
      align: 'center',
      slot: 'scope'
    },
    {
      prop: 'status',
      label: '状态',
      width: 100,
      align: 'center',
      slot: 'status'
    },
    { prop: 'remark', label: '备注', minWidth: 160 },
    {
      columnKey: 'action',
      label: '操作',
      width: 180,
      align: 'center',
      slot: 'action',
      fixed: 'right'
    }
  ];

  const formatScope = (row) => {
    if (row.cloakName) {
      return `斗篷专属 / ${row.cloakName}`;
    }
    return row.scope === 'cloak' ? '斗篷专属' : '商户全局';
  };

  const datasource = ({ page, limit, where }) => {
    return pageAccessLists({ ...where, pageNum: page, pageSize: limit });
  };

  const reload = (where, page) => {
    tableRef.value?.reload?.({ where, page });
  };

  const handleAdd = () => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/access-list-edit.vue'),
      componentProps: { onDone: () => reload() }
    });
  };

  const handleEdit = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/access-list-edit.vue'),
      componentProps: { data: row, onDone: () => reload() }
    });
  };

  const handleStatus = (row) => {
    const nextStatus = row.status === 'enabled' ? 'disabled' : 'enabled';
    const actionText = nextStatus === 'enabled' ? '启用' : '停用';
    ElMessageBox.confirm(`是否确认${actionText}该名单?`, '系统提示', {
      type: 'warning',
      draggable: true
    })
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        updateAccessListStatus(row.listId, nextStatus)
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

  const handleRemove = (row) => {
    ElMessageBox.confirm('是否确认删除该名单?', '系统提示', {
      type: 'warning',
      draggable: true
    })
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        removeAccessList(row.listId)
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
</script>
