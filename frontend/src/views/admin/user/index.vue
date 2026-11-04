<template>
  <ele-page flex-table="auto">
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        ref="tableRef"
        row-key="userId"
        :columns="columns"
        :datasource="datasource"
        cache-key="adminAccountTable"
      >
        <template #toolbar>
          <el-button
            v-permission="'system:user:edit'"
            type="primary"
            @click="open()"
            >新建管理员</el-button
          >
        </template>
        <template #roles="{ row }">{{
          (row.roleNames || []).join('、')
        }}</template>
        <template #action="{ row }">
          <el-button
            v-permission="'system:user:edit'"
            link
            type="primary"
            @click="open(row)"
            >编辑</el-button
          >
          <el-button
            v-permission="'system:user:edit'"
            link
            type="primary"
            @click="resetPwd(row)"
            >重置密码</el-button
          >
          <el-button
            v-permission="'system:user:edit'"
            link
            type="danger"
            @click="remove(row)"
            >删除</el-button
          >
        </template>
      </ele-pro-table>
    </ele-card>
    <el-dialog
      v-model="visible"
      :title="form.userId ? '编辑管理员' : '新建管理员'"
      width="480px"
    >
      <el-form label-width="72px">
        <el-form-item label="账号"
          ><el-input v-model="form.userName" :disabled="!!form.userId"
        /></el-form-item>
        <el-form-item v-if="!form.userId" label="密码"
          ><el-input v-model="form.password" show-password
        /></el-form-item>
        <el-form-item label="昵称"
          ><el-input v-model="form.nickName"
        /></el-form-item>
        <el-form-item label="角色">
          <el-select v-model="form.roleIds" multiple style="width: 100%">
            <el-option
              v-for="r in roles"
              :key="r.roleId"
              :label="r.name"
              :value="r.roleId"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio value="enabled" label="enabled">启用</el-radio>
            <el-radio value="disabled" label="disabled">停用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="备注"
          ><el-input v-model="form.remark"
        /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit"
          >保存</el-button
        >
      </template>
    </el-dialog>
  </ele-page>
</template>

<script setup>
  import { reactive, ref } from 'vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage } from 'ele-admin-plus';
  import {
    listRoles,
    pageAccounts,
    removeAccount,
    resetAccountPassword,
    saveAccount
  } from '@/api/admin/platform';

  defineOptions({ name: 'AdminAccount' });

  const tableRef = ref(null);
  const visible = ref(false);
  const saving = ref(false);
  const roles = ref([]);
  const form = reactive({
    userId: 0,
    userName: '',
    password: '',
    nickName: '',
    roleIds: [],
    status: 'enabled',
    remark: ''
  });
  const columns = [
    { prop: 'userName', label: '账号', minWidth: 120 },
    { prop: 'nickName', label: '昵称', minWidth: 120 },
    { prop: 'roleNames', label: '角色', minWidth: 160, slot: 'roles' },
    {
      prop: 'status',
      label: '状态',
      width: 90,
      formatter: (row) => (row.status === 'enabled' ? '启用' : '停用')
    },
    { prop: 'createdAt', label: '创建时间', width: 170 },
    { columnKey: 'action', label: '操作', width: 200, slot: 'action' }
  ];

  const datasource = ({ page, limit, where }) =>
    pageAccounts({ ...where, pageNum: page, pageSize: limit });

  const open = async (row) => {
    roles.value = (await listRoles()) || [];
    Object.assign(form, {
      userId: row?.userId || 0,
      userName: row?.userName || '',
      password: '',
      nickName: row?.nickName || '',
      roleIds: row?.roleIds ? [...row.roleIds] : [],
      status: row?.status || 'enabled',
      remark: row?.remark || ''
    });
    visible.value = true;
  };

  const submit = async () => {
    saving.value = true;
    try {
      await saveAccount(form.userId, form);
      EleMessage.success({ message: '已保存', plain: true });
      visible.value = false;
      tableRef.value?.reload?.();
    } catch (e) {
      EleMessage.error({ message: e.message, plain: true });
    } finally {
      saving.value = false;
    }
  };

  const resetPwd = (row) => {
    ElMessageBox.prompt(`为「${row.userName}」设置新密码`, '重置密码', {
      inputType: 'password'
    })
      .then(async ({ value }) => {
        const msg = await resetAccountPassword(row.userId, value);
        EleMessage.success({ message: msg, plain: true });
      })
      .catch(() => {});
  };

  const remove = (row) => {
    ElMessageBox.confirm(`删除管理员「${row.userName}」？`, '系统提示', {
      type: 'warning'
    })
      .then(async () => {
        await removeAccount(row.userId);
        EleMessage.success({ message: '已删除', plain: true });
        tableRef.value?.reload?.();
      })
      .catch(() => {});
  };
</script>
