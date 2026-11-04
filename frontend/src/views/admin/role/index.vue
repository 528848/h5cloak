<template>
  <ele-page>
    <role-search @search="(where) => reload(where)" />
    <ele-card :body-style="{ paddingTop: '8px' }">
      <ele-pro-table
        ref="tableRef"
        row-key="roleId"
        :columns="columns"
        :datasource="datasource"
        :pagination="false"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        v-model:selections="selections"
        cache-key="adminRoleTable"
      >
        <template #toolbar>
          <btn-items
            :items="[
              {
                preset: 'add',
                permission: 'system:role:edit',
                onClick: () => handleEdit()
              },
              {
                preset: 'del',
                permission: 'system:role:edit',
                onClick: () => handleRemove()
              }
            ]"
          />
        </template>
        <template #status="{ row }">
          <el-switch
            size="small"
            :model-value="row.status === 'enabled'"
            :disabled="row.builtin"
            @change="(checked) => handleStatus(checked, row)"
          />
        </template>
        <template #action="{ row }">
          <btn-items
            divider
            type="link"
            :items="[
              {
                preset: 'edit',
                permission: 'system:role:edit',
                onClick: () => handleEdit(row)
              },
              {
                preset: 'del',
                permission: 'system:role:edit',
                props: { disabled: row.builtin },
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
  import RoleSearch from './components/role-search.vue';
  import { listRoles, removeRole, saveRole } from '@/api/admin/platform';

  defineOptions({ name: 'AdminRole' });

  const { openModal } = useModal();
  const tableRef = ref(null);
  const selections = ref([]);
  const whereRef = ref({});

  const columns = ref([
    {
      type: 'selection',
      columnKey: 'selection',
      width: 50,
      align: 'center',
      selectable: (row) => !row.builtin
    },
    { type: 'index', columnKey: 'index', width: 50, align: 'center' },
    { prop: 'name', label: '角色名称', minWidth: 140 },
    { prop: 'roleKey', label: '权限字符', minWidth: 140 },
    {
      prop: 'builtin',
      label: '类型',
      width: 90,
      align: 'center',
      formatter: (row) => (row.builtin ? '内置' : '自定义')
    },
    {
      columnKey: 'menus',
      label: '菜单',
      width: 90,
      align: 'center',
      formatter: (row) =>
        row.roleKey === 'admin' ? '全部' : (row.menuIds || []).length
    },
    {
      prop: 'status',
      label: '状态',
      width: 90,
      align: 'center',
      slot: 'status'
    },
    { prop: 'remark', label: '备注', minWidth: 160 },
    {
      columnKey: 'action',
      label: '操作',
      width: 140,
      align: 'center',
      slot: 'action'
    }
  ]);

  const datasource = async ({ where }) => {
    whereRef.value = where || {};
    const rows = (await listRoles()) || [];
    return rows.filter((r) => {
      if (where?.name && !`${r.name}`.includes(where.name)) {
        return false;
      }
      if (where?.roleKey && !`${r.roleKey}`.includes(where.roleKey)) {
        return false;
      }
      if (where?.status && r.status !== where.status) {
        return false;
      }
      return true;
    });
  };

  const reload = (where) => tableRef.value?.reload?.({ where });

  const handleEdit = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/role-edit.vue'),
      componentProps: { data: row, onDone: () => reload(whereRef.value) }
    });
  };

  const handleStatus = (checked, row) => {
    const status = checked ? 'enabled' : 'disabled';
    ElMessageBox.confirm(
      `确认要${checked ? '启用' : '停用'}角色「${row.name}」吗？`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(async () => {
        await saveRole(row.roleId, { ...row, status });
        EleMessage.success({ message: '状态已更新', plain: true });
        reload(whereRef.value);
      })
      .catch(() => {});
  };

  const handleRemove = (row) => {
    const rows = row == null ? selections.value : [row];
    if (!rows.length) {
      EleMessage.error({ message: '请至少选择一条数据', plain: true });
      return;
    }
    ElMessageBox.confirm(
      `是否确认删除角色「${rows.map((d) => d.name).join('、')}」？`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(async () => {
        for (const item of rows) {
          await removeRole(item.roleId);
        }
        EleMessage.success({ message: '删除成功', plain: true });
        selections.value = [];
        reload(whereRef.value);
      })
      .catch(() => {});
  };
</script>
