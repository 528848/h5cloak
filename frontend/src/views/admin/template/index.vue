<template>
  <ele-page flex-table="auto">
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        ref="tableRef"
        row-key="templateId"
        :columns="columns"
        :datasource="datasource"
        :pagination="false"
        cache-key="adminTemplateTable"
      >
        <template #toolbar>
          <el-button
            v-permission="'system:template:edit'"
            type="primary"
            @click="open()"
            >新建模板</el-button
          >
        </template>
        <template #action="{ row }">
          <el-button
            v-permission="'system:template:edit'"
            link
            type="primary"
            @click="open(row)"
            >编辑</el-button
          >
          <el-button
            v-permission="'system:template:edit'"
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
      :title="form.templateId ? '编辑模板' : '新建模板'"
      width="720px"
      destroy-on-close
    >
      <el-form label-width="72px">
        <el-form-item label="编码"
          ><el-input v-model="form.code"
        /></el-form-item>
        <el-form-item label="名称"
          ><el-input v-model="form.name"
        /></el-form-item>
        <el-form-item label="说明"
          ><el-input v-model="form.description"
        /></el-form-item>
        <el-form-item label="HTML"
          ><el-input v-model="form.html" type="textarea" :rows="14"
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
    getTemplate,
    listTemplates,
    removeTemplate,
    saveTemplate
  } from '@/api/admin/platform';

  defineOptions({ name: 'AdminTemplate' });

  const tableRef = ref(null);
  const visible = ref(false);
  const saving = ref(false);
  const form = reactive({
    templateId: 0,
    code: '',
    name: '',
    description: '',
    html: ''
  });
  const columns = [
    { prop: 'code', label: '编码', minWidth: 120 },
    { prop: 'name', label: '名称', minWidth: 140 },
    { prop: 'description', label: '说明', minWidth: 180 },
    { prop: 'createdAt', label: '创建时间', width: 170 },
    { columnKey: 'action', label: '操作', width: 140, slot: 'action' }
  ];

  const datasource = async () => {
    const rows = (await listTemplates()) || [];
    return { rows, total: rows.length };
  };

  const open = async (row) => {
    Object.assign(form, {
      templateId: 0,
      code: '',
      name: '',
      description: '',
      html: '<!doctype html><title>审核</title><p>内容审核中</p>'
    });
    if (row) {
      const data = await getTemplate(row.templateId);
      Object.assign(form, data);
    }
    visible.value = true;
  };

  const submit = async () => {
    saving.value = true;
    try {
      await saveTemplate(form.templateId, form);
      EleMessage.success({ message: '已保存', plain: true });
      visible.value = false;
      tableRef.value?.reload?.();
    } catch (e) {
      EleMessage.error({ message: e.message, plain: true });
    } finally {
      saving.value = false;
    }
  };

  const remove = (row) => {
    ElMessageBox.confirm(`删除模板「${row.name}」？`, '系统提示', {
      type: 'warning'
    })
      .then(async () => {
        await removeTemplate(row.templateId);
        EleMessage.success({ message: '已删除', plain: true });
        tableRef.value?.reload?.();
      })
      .catch(() => {});
  };
</script>
