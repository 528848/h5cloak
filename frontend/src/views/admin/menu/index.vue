<template>
  <ele-page flex-table="auto">
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <el-table
        :data="rows"
        border
        row-key="menuId"
        default-expand-all
        :tree-props="{ children: 'children' }"
      >
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column label="类型" width="70">
          <template #default="{ row }">{{
            { M: '目录', C: '页面', F: '按钮' }[row.menuType] || row.menuType
          }}</template>
        </el-table-column>
        <el-table-column prop="path" label="路径" min-width="160" />
        <el-table-column prop="perms" label="权限" min-width="180" />
        <el-table-column prop="sort" label="排序" width="80" />
        <el-table-column label="显示" width="80">
          <template #default="{ row }">{{
            row.visible ? '是' : '否'
          }}</template>
        </el-table-column>
        <el-table-column label="操作" width="90">
          <template #default="{ row }">
            <el-button
              v-permission="'system:menu:edit'"
              link
              type="primary"
              @click="open(row)"
              >编辑</el-button
            >
          </template>
        </el-table-column>
      </el-table>
    </ele-card>
    <el-dialog v-model="visible" title="编辑菜单" width="420px">
      <el-form label-width="72px">
        <el-form-item label="名称"
          ><el-input v-model="form.name"
        /></el-form-item>
        <el-form-item label="图标"
          ><el-input v-model="form.icon"
        /></el-form-item>
        <el-form-item label="排序"
          ><el-input-number v-model="form.sort" :min="0"
        /></el-form-item>
        <el-form-item label="显示"
          ><el-switch v-model="form.visible"
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
  import { EleMessage } from 'ele-admin-plus';
  import { listMenus, saveMenu } from '@/api/admin/platform';

  defineOptions({ name: 'AdminMenu' });

  const rows = ref([]);
  const visible = ref(false);
  const saving = ref(false);
  const form = reactive({
    menuId: 0,
    name: '',
    icon: '',
    sort: 0,
    visible: true
  });

  const load = async () => {
    const flat = (await listMenus()) || [];
    const byParent = {};
    flat.forEach((m) => {
      (byParent[m.parentId || 0] ||= []).push({ ...m });
    });
    const attach = (nodes) =>
      nodes.map((n) => {
        const kids = attach(byParent[n.menuId] || []);
        return kids.length ? { ...n, children: kids } : n;
      });
    rows.value = attach(byParent[0] || []);
  };

  const open = (row) => {
    Object.assign(form, {
      menuId: row.menuId,
      name: row.name,
      icon: row.icon,
      sort: row.sort,
      visible: row.visible
    });
    visible.value = true;
  };

  const submit = async () => {
    saving.value = true;
    try {
      const msg = await saveMenu(form.menuId, form);
      EleMessage.success({ message: msg, plain: true });
      visible.value = false;
      await load();
    } catch (e) {
      EleMessage.error({ message: e.message, plain: true });
    } finally {
      saving.value = false;
    }
  };

  load().catch((e) => EleMessage.error({ message: e.message, plain: true }));
</script>
