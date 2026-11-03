<template>
  <ele-page flex-table="auto">
    <ele-card
      class="table-section"
      :body-style="{ paddingTop: '8px' }"
      flex-table="auto"
    >
      <ele-pro-table
        ref="tableRef"
        row-key="paramId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :pagination="false"
        cache-key="adminParamTable"
      >
        <template #toolbar>
          <btn-items
            :items="[
              {
                preset: 'add',
                title: '新增内置参数',
                permission: 'attribution:param:edit',
                onClick: () => handleEdit()
              }
            ]"
          />
        </template>
        <template #name="{ row }">
          <span class="mono">{{ row.name }}</span>
          <span v-if="row.aliases?.length" class="muted-text alias-text">
            别名 {{ row.aliases.join(', ') }}
          </span>
        </template>
        <template #paramType="{ row }">
          <el-tag
            :type="getOptionType(paramTypeOptions, row.paramType)"
            effect="plain"
            size="small"
          >
            {{ getOptionLabel(paramTypeOptions, row.paramType) }}
          </el-tag>
        </template>
        <template #format="{ row }">
          <span class="muted-text">{{ row.minLen }}-{{ row.maxLen }} 位</span>
          <span v-if="row.pattern" class="mono pattern-text">{{
            row.pattern
          }}</span>
        </template>
        <template #sufficient="{ row }">
          <el-tag
            :type="row.sufficient ? 'success' : 'info'"
            effect="plain"
            size="small"
          >
            {{ row.sufficient ? '是' : '否' }}
          </el-tag>
        </template>
        <template #status="{ row }">
          <el-tag
            :type="getOptionType(paramStatusOptions, row.status)"
            effect="light"
            size="small"
          >
            {{ getOptionLabel(paramStatusOptions, row.status) }}
          </el-tag>
        </template>
        <template #action="{ row }">
          <el-space :size="6" spacer="|">
            <el-button
              v-permission="'attribution:param:edit'"
              link
              type="primary"
              @click="handleEdit(row)"
              >编辑</el-button
            >
            <el-button
              v-permission="'attribution:param:edit'"
              link
              type="danger"
              @click="handleRemove(row)"
              >删除</el-button
            >
          </el-space>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref } from 'vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { listBuiltinParams, removeBuiltinParam } from '@/api/admin/param';
  import {
    paramTypeOptions,
    paramStatusOptions,
    getOptionLabel,
    getOptionType
  } from '@/views/merchant/attribution/constants';

  defineOptions({ name: 'AdminParam' });

  const { openModal } = useModal();
  const tableRef = ref(null);

  const columns = [
    { type: 'index', columnKey: 'index', width: 50, align: 'center' },
    { prop: 'name', label: '参数名', minWidth: 200, slot: 'name' },
    {
      prop: 'paramType',
      label: '类型',
      width: 100,
      align: 'center',
      slot: 'paramType'
    },
    {
      prop: 'platform',
      label: '平台',
      width: 110,
      formatter: (row) => row.platform || '-'
    },
    { columnKey: 'format', label: '格式', minWidth: 220, slot: 'format' },
    {
      prop: 'sufficient',
      label: '单独放行',
      width: 90,
      align: 'center',
      slot: 'sufficient'
    },
    {
      prop: 'status',
      label: '状态',
      width: 80,
      align: 'center',
      slot: 'status'
    },
    { prop: 'sort', label: '排序', width: 70, align: 'center' },
    {
      prop: 'remark',
      label: '说明',
      minWidth: 180,
      formatter: (row) => row.remark || '-'
    },
    { prop: 'updatedAt', label: '更新时间', minWidth: 150 },
    {
      columnKey: 'action',
      label: '操作',
      width: 120,
      align: 'center',
      slot: 'action',
      hideInPrint: true,
      hideInExport: true
    }
  ];

  const datasource = () => listBuiltinParams();

  const reload = () => tableRef.value?.reload?.();

  const handleEdit = (row) =>
    openModal({
      custom: true,
      asyncComponent: () =>
        import('@/views/merchant/attribution/components/param-edit.vue'),
      componentProps: { mode: 'admin', data: row, onDone: reload }
    });

  const handleRemove = (row) => {
    ElMessageBox.confirm(
      `删除内置参数「${row.name}」后，所有站点都不再识别该参数。是否继续？`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => removeBuiltinParam(row.paramId))
      .then((msg) => {
        if (!msg) return;
        EleMessage.success({ message: msg, plain: true });
        reload();
      })
      .catch((e) => {
        if (e?.message) EleMessage.error({ message: e.message, plain: true });
      });
  };
</script>

<style scoped>
  .alias-text,
  .pattern-text {
    margin-left: 8px;
    font-size: 12px;
  }
</style>
