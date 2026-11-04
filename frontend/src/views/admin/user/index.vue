<template>
  <ele-page>
    <user-search @search="(where) => reload(where, 1)" />
    <ele-card :body-style="{ paddingTop: '8px' }">
      <ele-pro-table
        ref="tableRef"
        row-key="userId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        v-model:selections="selections"
        cache-key="adminAccountTable"
      >
        <template #toolbar>
          <btn-items
            :items="[
              {
                preset: 'add',
                permission: 'system:user:edit',
                onClick: () => handleEdit()
              },
              {
                preset: 'del',
                permission: 'system:user:edit',
                onClick: () => handleRemove()
              }
            ]"
          />
        </template>
        <template #status="{ row }">
          <el-switch
            size="small"
            :model-value="row.status === 'enabled'"
            :disabled="row.userName === 'admin'"
            @change="(checked) => handleStatus(checked, row)"
          />
        </template>
        <template #roles="{ row }">
          <el-tag
            v-for="name in row.roleNames || []"
            :key="name"
            size="small"
            effect="plain"
            style="margin-right: 4px"
          >
            {{ name }}
          </el-tag>
        </template>
        <template #action="{ row }">
          <btn-items
            divider
            type="link"
            :items="[
              {
                preset: 'edit',
                permission: 'system:user:edit',
                onClick: () => handleEdit(row)
              },
              {
                title: '重置密码',
                permission: 'system:user:edit',
                onClick: () => handleResetPwd(row)
              },
              {
                preset: 'del',
                permission: 'system:user:edit',
                props: { disabled: row.userName === 'admin' },
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
  import UserSearch from './components/user-search.vue';
  import {
    pageAccounts,
    removeAccount,
    resetAccountPassword,
    updateAccountStatus
  } from '@/api/admin/platform';

  defineOptions({ name: 'AdminAccount' });

  const { openModal } = useModal();
  const tableRef = ref(null);
  const selections = ref([]);

  const columns = ref([
    { type: 'selection', columnKey: 'selection', width: 50, align: 'center' },
    { type: 'index', columnKey: 'index', width: 50, align: 'center' },
    { prop: 'userName', label: '用户名称', minWidth: 120 },
    { prop: 'nickName', label: '用户昵称', minWidth: 120 },
    {
      prop: 'roleNames',
      label: '角色',
      minWidth: 160,
      slot: 'roles'
    },
    {
      prop: 'status',
      label: '状态',
      width: 90,
      align: 'center',
      slot: 'status'
    },
    { prop: 'createdAt', label: '创建时间', width: 170, align: 'center' },
    {
      columnKey: 'action',
      label: '操作',
      width: 220,
      align: 'center',
      slot: 'action',
      hideInPrint: true,
      hideInExport: true
    }
  ]);

  const datasource = ({ page, limit, where }) =>
    pageAccounts({ ...where, pageNum: page, pageSize: limit });

  const reload = (where, page) => {
    tableRef.value?.reload?.({ where, page });
  };

  const handleEdit = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/user-edit.vue'),
      componentProps: { data: row, onDone: () => reload() }
    });
  };

  const handleStatus = (checked, row) => {
    const status = checked ? 'enabled' : 'disabled';
    const text = checked ? '启用' : '停用';
    ElMessageBox.confirm(`确认要${text}「${row.userName}」吗？`, '系统提示', {
      type: 'warning',
      draggable: true
    })
      .then(async () => {
        const msg = await updateAccountStatus(row.userId, status);
        EleMessage.success({ message: msg, plain: true });
        reload();
      })
      .catch(() => {});
  };

  const handleResetPwd = (row) => {
    ElMessageBox.prompt(`请输入「${row.userName}」的新密码`, '重置密码', {
      inputType: 'password',
      inputPlaceholder: '至少 6 位',
      draggable: true
    })
      .then(async ({ value }) => {
        const msg = await resetAccountPassword(row.userId, value);
        EleMessage.success({ message: msg, plain: true });
      })
      .catch(() => {});
  };

  const handleRemove = (row) => {
    const rows = row == null ? selections.value : [row];
    if (!rows.length) {
      EleMessage.error({ message: '请至少选择一条数据', plain: true });
      return;
    }
    const names = rows.map((d) => d.userName).join('、');
    ElMessageBox.confirm(`是否确认删除用户「${names}」？`, '系统提示', {
      type: 'warning',
      draggable: true
    })
      .then(async () => {
        for (const item of rows) {
          await removeAccount(item.userId);
        }
        EleMessage.success({ message: '删除成功', plain: true });
        selections.value = [];
        reload();
      })
      .catch(() => {});
  };
</script>
