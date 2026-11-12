<template>
  <ele-page flex-table="auto">
    <summary-cards
      :items="summaryItems"
      :cols="4"
      clickable
      :active-key="form.kind"
      @select="selectKind"
    />
    <ele-card search-form class="page-section">
      <el-form label-width="72px" @keyup.enter="handleSearch" @submit.prevent="">
        <el-row :gutter="12">
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="关键字">
              <el-input
                v-model.trim="form.keyword"
                clearable
                placeholder="名称或编码"
              />
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="来源">
              <el-select v-model="form.kind" clearable placeholder="全部" class="ele-fluid">
                <el-option
                  v-for="o in kindOptions"
                  :key="o.value"
                  :label="o.label"
                  :value="o.value"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="状态">
              <el-select v-model="form.status" clearable placeholder="全部" class="ele-fluid">
                <el-option label="启用" value="enabled" />
                <el-option label="停用" value="disabled" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label-width="0">
              <btn-items
                :wrap="false"
                :items="[
                  { preset: 'search', onClick: () => handleSearch() },
                  { preset: 'reset', onClick: () => handleReset() }
                ]"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </ele-card>
    <ele-card class="table-section" :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        ref="tableRef"
        row-key="templateId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        cache-key="adminTemplateTableV2"
      >
        <template #toolbar>
          <el-button
            v-permission="'system:template:edit'"
            type="primary"
            @click="goForm()"
          >
            新建模板
          </el-button>
        </template>
        <template #kind="{ row }">
          <el-tag effect="plain" size="small">{{ kindLabel(row.kind) }}</el-tag>
        </template>
        <template #status="{ row }">
          <el-tag :type="row.status === 'enabled' ? 'success' : 'info'" effect="light" size="small">
            {{ row.status === 'enabled' ? '启用' : '停用' }}
          </el-tag>
        </template>
        <template #action="{ row }">
          <el-space :size="8">
            <el-button
              v-permission="'system:template:edit'"
              link
              type="primary"
              @click="goForm(row)"
            >
              编辑
            </el-button>
            <el-button
              v-permission="'system:template:edit'"
              link
              :type="row.status === 'enabled' ? 'warning' : 'primary'"
              @click="toggleStatus(row)"
            >
              {{ row.status === 'enabled' ? '停用' : '启用' }}
            </el-button>
            <el-button
              v-permission="'system:template:edit'"
              link
              type="danger"
              @click="remove(row)"
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
  import { computed, ref, onActivated } from 'vue';
  import { useRouter } from 'vue-router';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import {
    getTemplateStats,
    listTemplates,
    removeTemplate,
    updateTemplateStatus
  } from '@/api/admin/platform';

  defineOptions({ name: 'AdminTemplate' });

  const kindOptions = [
    { value: 'html', label: 'HTML' },
    { value: 'url', label: '网址' },
    { value: 'static', label: '静态项目' }
  ];
  const kindLabel = (v) => kindOptions.find((o) => o.value === v)?.label || 'HTML';

  const router = useRouter();
  const tableRef = ref(null);
  const stats = ref({ total: 0, html: 0, url: 0, static: 0 });
  const [form, resetFields] = useFormData({ keyword: '', kind: '', status: '' });

  const summaryItems = computed(() => [
    { key: '', label: '全部模板', value: stats.value.total, desc: '平台可复用的审核内容', type: 'primary' },
    { key: 'html', label: 'HTML', value: stats.value.html, desc: '任意路径同一份页面', type: 'success' },
    { key: 'url', label: '网址', value: stats.value.url, desc: '按访客路径回源', type: 'warning' },
    { key: 'static', label: '静态项目', value: stats.value.static, desc: '按路径提供文件', type: 'info' }
  ]);

  const columns = [
    { type: 'index', columnKey: 'index', width: 50, align: 'center' },
    { prop: 'code', label: '编码', minWidth: 120 },
    { prop: 'name', label: '名称', minWidth: 140 },
    {
      prop: 'kind',
      label: '来源',
      width: 110,
      align: 'center',
      slot: 'kind',
      formatter: (row) => kindLabel(row.kind)
    },
    {
      prop: 'status',
      label: '状态',
      width: 90,
      align: 'center',
      slot: 'status',
      formatter: (row) => (row.status === 'enabled' ? '启用' : '停用')
    },
    { prop: 'usedCount', label: '引用站点', width: 100, align: 'center' },
    { prop: 'updatedAt', label: '更新时间', minWidth: 160 },
    {
      columnKey: 'action',
      label: '操作',
      width: 180,
      align: 'center',
      fixed: 'right',
      slot: 'action',
      hideInPrint: true,
      hideInExport: true
    }
  ];

  const loadStats = () => {
    getTemplateStats().then((data) => (stats.value = data)).catch(() => {});
  };

  const datasource = ({ page, limit, where }) =>
    listTemplates({ ...where, pageNum: page, pageSize: limit });

  const reload = (where, page) => {
    tableRef.value?.reload?.({ where, page });
    loadStats();
  };
  const handleSearch = () => reload({ ...form }, 1);
  const handleReset = () => {
    resetFields();
    handleSearch();
  };
  const selectKind = (item) => {
    form.kind = item.key;
    handleSearch();
  };
  const goForm = (row) => {
    router.push(row ? `/admin/template/form?id=${row.templateId}` : '/admin/template/form');
  };
  const toggleStatus = (row) => {
    const next = row.status === 'enabled' ? 'disabled' : 'enabled';
    const text = next === 'disabled' ? '停用后，选用它的站点会改为占位页。是否确认停用？' : '是否确认启用该模板？';
    ElMessageBox.confirm(text, '系统提示', { type: 'warning', draggable: true })
      .then(() => updateTemplateStatus(row.templateId, next))
      .then((msg) => {
        EleMessage.success({ message: msg, plain: true });
        reload({ ...form });
      })
      .catch((e) => {
        if (e?.message) EleMessage.error({ message: e.message, plain: true });
      });
  };
  const remove = (row) => {
    ElMessageBox.confirm(`是否确认删除模板「${row.name}」？`, '系统提示', {
      type: 'warning',
      draggable: true
    })
      .then(() => removeTemplate(row.templateId))
      .then((msg) => {
        EleMessage.success({ message: msg, plain: true });
        reload({ ...form });
      })
      .catch((e) => {
        if (e?.message) EleMessage.error({ message: e.message, plain: true });
      });
  };

  loadStats();

  // 列表被页签缓存，从新建/编辑表单页返回时不会重新挂载，这里补一次刷新。
  let skipActivate = true;
  onActivated(() => {
    if (skipActivate) {
      skipActivate = false;
      return;
    }
    reload({ ...form });
  });
</script>
