<template>
  <ele-page flex-table="auto">
    <user-search @search="(where) => reload(where, 1)" />
    <ele-card
      class="table-section"
      :body-style="{ paddingTop: '8px' }"
      flex-table="auto"
    >
      <ele-pro-table
        ref="tableRef"
        row-key="userId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        cache-key="merchantUserTable"
      >
        <template #toolbar>
          <btn-items
            :items="[
              {
                preset: 'add',
                permission: 'merchant:user:add',
                onClick: () => handleAdd()
              }
            ]"
          />
        </template>
        <template #userName="{ row }">
          <span class="mono">{{ row.userName }}</span>
          <el-tag
            v-if="row.userId === currentUserId"
            size="small"
            effect="plain"
            style="margin-left: 6px"
            >本人</el-tag
          >
        </template>
        <template #role="{ row }">
          <el-tag effect="light" size="small">
            {{ row.roleNames?.[0] || getOptionLabel(roleOptions, row.roleKeys[0]) }}
          </el-tag>
        </template>
        <template #status="{ row }">
          <el-tag
            :type="getOptionType(userStatusOptions, row.status)"
            effect="light"
            size="small"
          >
            {{ getOptionLabel(userStatusOptions, row.status) }}
          </el-tag>
        </template>
        <template #action="{ row }">
          <el-space :size="6" spacer="|">
            <el-button
              v-permission="'merchant:user:edit'"
              link
              type="primary"
              @click="handleEdit(row)"
              >编辑</el-button
            >
            <el-button
              v-permission="'merchant:user:resetPwd'"
              link
              type="warning"
              @click="handleResetPwd(row)"
              >重置密码</el-button
            >
            <el-button
              v-if="row.userId !== currentUserId"
              v-permission="'merchant:user:status'"
              link
              :type="row.status === 'enabled' ? 'info' : 'success'"
              @click="handleStatus(row)"
            >
              {{ row.status === 'enabled' ? '停用' : '启用' }}
            </el-button>
            <el-button
              v-if="row.userId !== currentUserId"
              v-permission="'merchant:user:remove'"
              link
              type="danger"
              @click="handleRemove(row)"
            >
              删除
            </el-button>
          </el-space>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref, computed } from 'vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useUserStore } from '@/store/modules/user';
  import {
    pageMerchantUsers,
    updateMerchantUserStatus,
    removeMerchantUser
  } from '@/api/merchant/user';
  import UserSearch from './components/user-search.vue';
  import {
    roleOptions,
    userStatusOptions,
    getOptionLabel,
    getOptionType
  } from './constants';

  defineOptions({ name: 'MerchantUser' });

  const { openModal } = useModal();
  const userStore = useUserStore();
  const currentUserId = computed(() => userStore.info?.userId);
  const tableRef = ref(null);

  const columns = [
    {
      type: 'index',
      columnKey: 'index',
      width: 50,
      align: 'center',
      fixed: 'left'
    },
    {
      prop: 'userName',
      label: '登录账号',
      minWidth: 160,
      slot: 'userName',
      formatter: (row) => row.userName
    },
    { prop: 'nickName', label: '昵称', minWidth: 120 },
    {
      columnKey: 'role',
      label: '角色',
      minWidth: 120,
      align: 'center',
      slot: 'role',
      formatter: (row) =>
        row.roleNames?.[0] || getOptionLabel(roleOptions, row.roleKeys[0])
    },
    {
      prop: 'status',
      label: '状态',
      width: 80,
      align: 'center',
      slot: 'status',
      formatter: (row) => getOptionLabel(userStatusOptions, row.status)
    },
    {
      prop: 'email',
      label: '邮箱',
      minWidth: 160,
      formatter: (row) => row.email || '-'
    },
    {
      prop: 'lastLoginAt',
      label: '最近登录',
      minWidth: 150,
      formatter: (row) => row.lastLoginAt || '-'
    },
    { prop: 'createdAt', label: '创建时间', minWidth: 150 },
    {
      columnKey: 'action',
      label: '操作',
      width: 230,
      align: 'center',
      fixed: 'right',
      slot: 'action',
      hideInPrint: true,
      hideInExport: true
    }
  ];

  const datasource = ({ page, limit, where }) =>
    pageMerchantUsers({ ...where, pageNum: page, pageSize: limit });

  const reload = (where, page) => tableRef.value?.reload?.({ where, page });

  const handleAdd = () =>
    openModal({
      custom: true,
      asyncComponent: () => import('./components/user-edit.vue'),
      componentProps: { onDone: () => reload() }
    });

  const handleEdit = (row) =>
    openModal({
      custom: true,
      asyncComponent: () => import('./components/user-edit.vue'),
      componentProps: { data: row, onDone: () => reload() }
    });

  const handleResetPwd = (row) =>
    openModal({
      custom: true,
      asyncComponent: () => import('./components/user-reset-password.vue'),
      componentProps: { data: row }
    });

  const run = (promise) => {
    const loading = EleMessage.loading({ message: '请求中..', plain: true });
    promise
      .then((msg) => {
        loading.close();
        EleMessage.success({ message: msg, plain: true });
        reload();
      })
      .catch((e) => {
        loading.close();
        EleMessage.error({ message: e.message, plain: true });
      });
  };

  const handleStatus = (row) => {
    const next = row.status === 'enabled' ? 'disabled' : 'enabled';
    ElMessageBox.confirm(
      `是否确认${next === 'enabled' ? '启用' : '停用'}账号「${row.userName}」？`,
      '系统提示',
      {
        type: 'warning',
        draggable: true
      }
    )
      .then(() => run(updateMerchantUserStatus(row.userId, next)))
      .catch(() => {});
  };

  const handleRemove = (row) => {
    ElMessageBox.confirm(
      `删除后不可恢复，是否确认删除账号「${row.userName}」？`,
      '系统提示',
      {
        type: 'warning',
        draggable: true
      }
    )
      .then(() => run(removeMerchantUser(row.userId)))
      .catch(() => {});
  };
</script>
