<template>
  <ele-page>
    <el-alert
      class="page-section"
      type="info"
      show-icon
      :closable="false"
      title="菜单由平台统一维护。这里查看商户端有哪些页面和权限，分配给角色时在「角色管理」里勾选。"
    />
    <ele-card class="table-section" :body-style="{ paddingTop: '8px' }">
      <ele-pro-table
        ref="tableRef"
        row-key="menuId"
        :columns="columns"
        :datasource="datasource"
        :pagination="false"
        :show-overflow-tooltip="true"
        cache-key="merchantMenuTable"
      >
        <template #toolbar>
          <el-button @click="tableRef?.toggleRowExpansionAll?.(true)">展开全部</el-button>
          <el-button @click="tableRef?.toggleRowExpansionAll?.(false)">折叠全部</el-button>
        </template>
        <template #menuType="{ row }">
          <el-tag v-if="row.menuType === 'M'" size="small">目录</el-tag>
          <el-tag v-else-if="row.menuType === 'C'" size="small" type="success" effect="plain">页面</el-tag>
          <el-tag v-else size="small" type="info" effect="plain">按钮</el-tag>
        </template>
        <template #visible="{ row }">
          <el-tag :type="row.visible ? 'success' : 'info'" size="small" effect="light">
            {{ row.visible ? '显示' : '隐藏' }}
          </el-tag>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref } from 'vue';
  import { toTree } from 'ele-admin-plus';
  import { listMerchantMenus } from '@/api/merchant/user';

  defineOptions({ name: 'MerchantMenu' });

  const tableRef = ref(null);
  const columns = [
    { prop: 'name', label: '菜单名称', minWidth: 180 },
    { prop: 'menuType', label: '类型', width: 90, align: 'center', slot: 'menuType', formatter: (row) => row.menuType },
    { prop: 'path', label: '路由地址', minWidth: 180, formatter: (row) => row.path || '-' },
    { prop: 'perms', label: '权限字符', minWidth: 180, formatter: (row) => row.perms || '-' },
    { prop: 'sort', label: '排序', width: 80, align: 'center' },
    { prop: 'visible', label: '显示', width: 90, align: 'center', slot: 'visible', formatter: (row) => (row.visible ? '显示' : '隐藏') }
  ];
  const datasource = async () => {
    const rows = (await listMerchantMenus()) || [];
    return toTree({
      data: rows.map((m) => ({ ...m, parentId: m.parentId || 0 })),
      idField: 'menuId',
      parentIdField: 'parentId'
    });
  };
</script>
