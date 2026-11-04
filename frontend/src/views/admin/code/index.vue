<template>
  <ele-page flex-table="auto">
    <ele-card :body-style="{ paddingBottom: '0' }">
      <el-form inline @submit.prevent="">
        <el-form-item label="关键词">
          <el-input
            v-model="keyword"
            clearable
            placeholder="分销码 / 名称"
            @keyup.enter="reload"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="reload">查询</el-button>
        </el-form-item>
      </el-form>
    </ele-card>
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        ref="tableRef"
        row-key="codeId"
        :columns="columns"
        :datasource="datasource"
        cache-key="adminPlatformCodeTable"
      >
        <template #action="{ row }">
          <el-button
            v-permission="'platform:code:status'"
            link
            :type="row.status === 'enabled' ? 'danger' : 'success'"
            @click="toggle(row)"
          >
            {{ row.status === 'enabled' ? '强制停用' : '恢复启用' }}
          </el-button>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref } from 'vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage } from 'ele-admin-plus';
  import {
    pagePlatformCodes,
    updatePlatformCodeStatus
  } from '@/api/admin/platform';

  defineOptions({ name: 'AdminPlatformCode' });

  const tableRef = ref(null);
  const keyword = ref('');
  const columns = [
    { prop: 'code', label: '分销码', minWidth: 110 },
    { prop: 'name', label: '名称', minWidth: 100 },
    { prop: 'merchantName', label: '商户', minWidth: 100 },
    { prop: 'siteName', label: '站点', minWidth: 120 },
    {
      prop: 'status',
      label: '状态',
      width: 90,
      formatter: (row) => (row.status === 'enabled' ? '启用' : '停用')
    },
    { prop: 'channel', label: '渠道', minWidth: 90 },
    { prop: 'createdAt', label: '创建时间', width: 160 },
    {
      columnKey: 'action',
      label: '操作',
      width: 100,
      slot: 'action',
      fixed: 'right'
    }
  ];

  const datasource = ({ page, limit }) => {
    return pagePlatformCodes({
      keyword: keyword.value,
      pageNum: page,
      pageSize: limit
    });
  };

  const reload = () => tableRef.value?.reload?.({ page: 1 });

  const toggle = (row) => {
    const status = row.status === 'enabled' ? 'disabled' : 'enabled';
    const text = status === 'disabled' ? '强制停用' : '恢复启用';
    ElMessageBox.confirm(`${text}分销码「${row.code}」？`, '系统提示', {
      type: 'warning'
    })
      .then(async () => {
        const msg = await updatePlatformCodeStatus([row.codeId], status);
        EleMessage.success({ message: msg, plain: true });
        tableRef.value?.reload?.();
      })
      .catch(() => {});
  };
</script>
