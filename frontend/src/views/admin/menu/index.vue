<template>
  <ele-page>
    <menu-search @search="(where) => reload(where)" />
    <ele-card :body-style="{ paddingTop: '8px' }">
      <ele-pro-table
        ref="tableRef"
        row-key="menuId"
        :columns="columns"
        :datasource="datasource"
        :pagination="false"
        :default-expand-all="false"
        :show-overflow-tooltip="true"
        cache-key="adminMenuTable"
      >
        <template #toolbar>
          <btn-items
            :items="[
              { preset: 'expand', onClick: () => handleExpandAll() },
              { preset: 'fold', onClick: () => handleFoldAll() }
            ]"
          />
        </template>
        <template #menuName="{ row }">
          <menu-icon
            v-if="row.icon"
            :icon="row.icon"
            :component-props="{ size: 15 }"
            :component-style="{ marginRight: '4px', verticalAlign: '-2px' }"
          />
          <span>{{ row.name }}</span>
        </template>
        <template #menuType="{ row }">
          <el-tag
            v-if="row.menuType === 'M'"
            size="small"
            :disable-transitions="true"
            >目录</el-tag
          >
          <el-tag
            v-else-if="row.menuType === 'C'"
            size="small"
            type="success"
            :disable-transitions="true"
            >菜单</el-tag
          >
          <el-tag v-else size="small" type="info" :disable-transitions="true"
            >按钮</el-tag
          >
        </template>
        <template #visible="{ row }">
          <el-tag
            size="small"
            :type="row.visible ? 'success' : 'info'"
            :disable-transitions="true"
          >
            {{ row.visible ? '显示' : '隐藏' }}
          </el-tag>
        </template>
        <template #action="{ row }">
          <btn-items
            type="link"
            :items="[
              {
                preset: 'edit',
                permission: 'system:menu:edit',
                onClick: () => handleEdit(row)
              }
            ]"
          />
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref } from 'vue';
  import { toTree } from 'ele-admin-plus';
  import MenuIcon from '@/components/IconSelect/components/menu-icon.vue';
  import MenuSearch from './components/menu-search.vue';
  import { useModal } from 'ele-admin-plus';
  import { listMenus } from '@/api/admin/platform';

  defineOptions({ name: 'AdminMenu' });

  const { openModal } = useModal();
  const tableRef = ref(null);

  const columns = ref([
    { type: 'index', columnKey: 'index', width: 50, align: 'center' },
    { prop: 'name', label: '菜单名称', slot: 'menuName', minWidth: 180 },
    {
      prop: 'menuType',
      label: '类型',
      width: 90,
      align: 'center',
      slot: 'menuType'
    },
    { prop: 'path', label: '路由地址', minWidth: 160 },
    { prop: 'perms', label: '权限字符', minWidth: 180 },
    { prop: 'sort', label: '排序', width: 80, align: 'center' },
    {
      prop: 'visible',
      label: '显示',
      width: 90,
      align: 'center',
      slot: 'visible'
    },
    {
      columnKey: 'action',
      label: '操作',
      width: 90,
      align: 'center',
      slot: 'action'
    }
  ]);

  const match = (row, where) => {
    if (where?.name && !`${row.name}`.includes(where.name)) {
      return false;
    }
    if (where?.visible === '1' && !row.visible) {
      return false;
    }
    if (where?.visible === '0' && row.visible) {
      return false;
    }
    return true;
  };

  const datasource = async ({ where }) => {
    const rows = (await listMenus()) || [];
    let data = rows.map((m) => ({ ...m, parentId: m.parentId || 0 }));
    if (where?.name || (where?.visible != null && where.visible !== '')) {
      const keep = new Set();
      const byId = Object.fromEntries(data.map((m) => [m.menuId, m]));
      data.forEach((m) => {
        if (!match(m, where)) {
          return;
        }
        let cur = m;
        while (cur) {
          keep.add(cur.menuId);
          cur = byId[cur.parentId];
        }
      });
      data = data.filter((m) => keep.has(m.menuId));
    }
    return toTree({ data, idField: 'menuId', parentIdField: 'parentId' });
  };

  const reload = (where) => tableRef.value?.reload?.({ where });

  const handleEdit = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/menu-edit.vue'),
      componentProps: { data: row, onDone: () => reload() }
    });
  };

  const handleExpandAll = () => {
    tableRef.value?.toggleRowExpansionAll?.(true);
  };
  const handleFoldAll = () => {
    tableRef.value?.toggleRowExpansionAll?.(false);
  };
</script>
