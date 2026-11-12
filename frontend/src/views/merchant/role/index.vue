<template>
  <ele-page>
    <ele-card search-form class="page-section">
      <el-form label-width="72px" @keyup.enter="reload" @submit.prevent="">
        <el-row :gutter="12">
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="角色名称">
              <el-input v-model.trim="keyword" clearable placeholder="请输入角色名称" />
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label-width="0">
              <btn-items
                :wrap="false"
                :items="[
                  { preset: 'search', onClick: () => reload() },
                  { preset: 'reset', onClick: () => reset() }
                ]"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </ele-card>
    <ele-card class="table-section" :body-style="{ paddingTop: '8px' }">
      <ele-pro-table
        ref="tableRef"
        row-key="roleId"
        :columns="columns"
        :datasource="datasource"
        :pagination="false"
        :show-overflow-tooltip="true"
        cache-key="merchantRoleTable"
      >
        <template #toolbar>
          <el-button v-permission="'merchant:role:edit'" type="primary" @click="open()">添加角色</el-button>
        </template>
        <template #kind="{ row }">
          <el-tag v-if="row.builtin" size="small" effect="plain">内置</el-tag>
          <el-tag v-else size="small" type="success" effect="plain">自定义</el-tag>
        </template>
        <template #status="{ row }">
          <el-switch
            size="small"
            :model-value="row.status === 'enabled'"
            :disabled="row.builtin"
            @change="(checked) => changeStatus(checked, row)"
          />
        </template>
        <template #action="{ row }">
          <el-button
            v-permission="'merchant:role:edit'"
            link
            type="primary"
            :disabled="row.builtin"
            @click="open(row)"
          >编辑</el-button>
          <el-button
            v-permission="'merchant:role:edit'"
            link
            type="danger"
            :disabled="row.builtin"
            @click="remove(row)"
          >删除</el-button>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref } from 'vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { listManageRoles, removeMerchantRole, saveMerchantRole } from '@/api/merchant/user';

  defineOptions({ name: 'MerchantRole' });

  const { openModal } = useModal();
  const tableRef = ref(null);
  const keyword = ref('');
  const columns = [
    { prop: 'name', label: '角色名称', minWidth: 140 },
    { prop: 'roleKey', label: '权限字符', minWidth: 160 },
    { prop: 'builtin', label: '类型', width: 90, align: 'center', slot: 'kind', formatter: (row) => (row.builtin ? '内置' : '自定义') },
    { prop: 'status', label: '状态', width: 90, align: 'center', slot: 'status', formatter: (row) => (row.status === 'enabled' ? '启用' : '停用') },
    { prop: 'remark', label: '备注', minWidth: 160, formatter: (row) => row.remark || '-' },
    { columnKey: 'action', label: '操作', width: 120, align: 'center', slot: 'action', hideInPrint: true, hideInExport: true }
  ];

  const datasource = async () => {
    const rows = (await listManageRoles()) || [];
    return keyword.value ? rows.filter((r) => r.name.includes(keyword.value)) : rows;
  };
  const reload = () => tableRef.value?.reload?.();
  const reset = () => {
    keyword.value = '';
    reload();
  };
  const changeStatus = (checked, row) => {
    const status = checked ? 'enabled' : 'disabled';
    ElMessageBox.confirm(`确认要${checked ? '启用' : '停用'}角色「${row.name}」吗？`, '系统提示', {
      type: 'warning',
      draggable: true
    })
      .then(() => saveMerchantRole(row.roleId, { ...row, status }))
      .then(() => {
        EleMessage.success({ message: '状态已更新', plain: true });
        reload();
      })
      .catch((e) => {
        if (e?.message) {
          EleMessage.error({ message: e.message, plain: true });
        }
      });
  };
  const open = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/role-edit.vue'),
      componentProps: { data: row, onDone: () => reload() }
    });
  };
  const remove = (row) => {
    ElMessageBox.confirm(`是否确认删除角色「${row.name}」？`, '系统提示', { type: 'warning', draggable: true })
      .then(() => removeMerchantRole(row.roleId))
      .then((msg) => {
        EleMessage.success({ message: msg, plain: true });
        reload();
      })
      .catch((e) => { if (e?.message) EleMessage.error({ message: e.message, plain: true }); });
  };
</script>
