<template>
  <ele-page flex-table="auto">
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <div style="margin-bottom: 12px">
        <el-button
          v-permission="'system:role:edit'"
          type="primary"
          @click="open()"
          >新建角色</el-button
        >
      </div>
      <el-table :data="rows" border>
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="roleKey" label="标识" min-width="140" />
        <el-table-column label="类型" width="90">
          <template #default="{ row }">{{
            row.builtin ? '内置' : '自定义'
          }}</template>
        </el-table-column>
        <el-table-column label="可见菜单" width="100">
          <template #default="{ row }">{{
            row.roleKey === 'admin' ? '全部' : (row.menuIds || []).length
          }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">{{
            row.status === 'enabled' ? '启用' : '停用'
          }}</template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="140" />
        <el-table-column label="操作" width="140">
          <template #default="{ row }">
            <el-button
              v-permission="'system:role:edit'"
              link
              type="primary"
              @click="open(row)"
              >编辑</el-button
            >
            <el-button
              v-if="!row.builtin"
              v-permission="'system:role:edit'"
              link
              type="danger"
              @click="remove(row)"
              >删除</el-button
            >
          </template>
        </el-table-column>
      </el-table>
    </ele-card>
    <el-dialog
      v-model="visible"
      :title="form.roleId ? '编辑角色' : '新建角色'"
      width="560px"
    >
      <el-form label-width="72px">
        <el-form-item label="名称"
          ><el-input v-model="form.name"
        /></el-form-item>
        <el-form-item label="标识"
          ><el-input v-model="form.roleKey" :disabled="!!form.roleId"
        /></el-form-item>
        <el-form-item label="备注"
          ><el-input v-model="form.remark"
        /></el-form-item>
        <el-form-item label="菜单">
          <el-tree
            ref="treeRef"
            :data="tree"
            show-checkbox
            node-key="menuId"
            default-expand-all
            :props="{ label: 'name', children: 'children' }"
          />
          <div class="hint"
            >超级管理员始终可见全部菜单，勾选只影响其他角色。</div
          >
        </el-form-item>
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
  import { nextTick, reactive, ref } from 'vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage } from 'ele-admin-plus';
  import {
    listMenus,
    listRoles,
    removeRole,
    saveRole
  } from '@/api/admin/platform';

  defineOptions({ name: 'AdminRole' });

  const rows = ref([]);
  const tree = ref([]);
  const flat = ref([]);
  const visible = ref(false);
  const saving = ref(false);
  const treeRef = ref(null);
  const form = reactive({
    roleId: 0,
    name: '',
    roleKey: '',
    remark: '',
    status: 'enabled'
  });

  const load = async () => {
    rows.value = (await listRoles()) || [];
    flat.value = (await listMenus()) || [];
    const byParent = {};
    flat.value.forEach((m) => {
      const pid = m.parentId || 0;
      (byParent[pid] ||= []).push({ ...m, children: [] });
    });
    const attach = (nodes) =>
      nodes.map((n) => {
        n.children = attach(byParent[n.menuId] || []);
        return n;
      });
    tree.value = attach(byParent[0] || []);
  };

  const open = async (row) => {
    Object.assign(form, {
      roleId: row?.roleId || 0,
      name: row?.name || '',
      roleKey: row?.roleKey || '',
      remark: row?.remark || '',
      status: row?.status || 'enabled'
    });
    visible.value = true;
    await nextTick();
    treeRef.value?.setCheckedKeys?.(row?.menuIds || []);
  };

  const submit = async () => {
    saving.value = true;
    try {
      const menuIds = treeRef.value?.getCheckedKeys?.(false) || [];
      const half = treeRef.value?.getHalfCheckedKeys?.() || [];
      await saveRole(form.roleId, { ...form, menuIds: [...menuIds, ...half] });
      EleMessage.success({ message: '已保存', plain: true });
      visible.value = false;
      await load();
    } catch (e) {
      EleMessage.error({ message: e.message, plain: true });
    } finally {
      saving.value = false;
    }
  };

  const remove = (row) => {
    ElMessageBox.confirm(`删除角色「${row.name}」？`, '系统提示', {
      type: 'warning'
    })
      .then(async () => {
        await removeRole(row.roleId);
        EleMessage.success({ message: '已删除', plain: true });
        await load();
      })
      .catch(() => {});
  };

  load().catch((e) => EleMessage.error({ message: e.message, plain: true }));
</script>

<style scoped>
  .hint {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
</style>
