<template>
  <ele-page>
    <!-- 搜索表单 -->
    <post-search @search="(where) => reload(where, 1)" />
    <ele-card :body-style="{ paddingTop: '8px' }">
      <!-- 表格 -->
      <ele-pro-table
        ref="tableRef"
        row-key="postId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        v-model:selections="selections"
        :export-config="{ fileName: '岗位数据' }"
        cache-key="systemPostTable"
      >
        <template #toolbar>
          <btn-items
            :items="[
              {
                preset: 'add',
                permission: 'system:post:add',
                onClick: () => handleEdit()
              },
              {
                preset: 'del',
                permission: 'system:post:remove',
                onClick: () => handleRemove()
              },
              {
                preset: 'export',
                permission: 'system:post:export',
                onClick: () => handleExport()
              }
            ]"
          />
        </template>
        <template #status="{ row }">
          <dict-data
            code="sys_normal_disable"
            type="tag"
            :model-value="row.status"
          />
        </template>
        <template #action="{ row }">
          <btn-items
            :divider="true"
            type="link"
            :items="[
              {
                preset: 'edit',
                permission: 'system:post:edit',
                onClick: () => handleEdit(row)
              },
              {
                preset: 'del',
                permission: 'system:post:remove',
                onClick: () => handleRemove(row)
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
  import { ElMessageBox } from 'element-plus';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useDictData } from '@/utils/use-dict-data';
  import PostSearch from './components/post-search.vue';
  import { pagePosts, removePosts, exportPosts } from '@/api/system/post';

  defineOptions({ name: 'SystemPost' });

  const { openModal } = useModal();

  /** 字典数据 */
  const [statusDicts] = useDictData(['sys_normal_disable']);

  /** 表格实例 */
  const tableRef = ref(null);

  /** 表格列配置 */
  const columns = ref([
    {
      type: 'selection',
      columnKey: 'selection',
      width: 50,
      align: 'center',
      showOverflowTooltip: false
    },
    {
      type: 'index',
      columnKey: 'index',
      width: 50,
      align: 'center'
    },
    {
      prop: 'postCode',
      label: '岗位编码',
      align: 'center',
      minWidth: 110
    },
    {
      prop: 'postName',
      label: '岗位名称',
      align: 'center',
      minWidth: 110
    },
    {
      prop: 'postSort',
      label: '岗位排序',
      align: 'center',
      minWidth: 110
    },
    {
      prop: 'status',
      label: '状态',
      width: 90,
      align: 'center',
      minWidth: 110,
      slot: 'status',
      formatter: (row) =>
        statusDicts.value.find((d) => d.dictValue == row.status)?.dictLabel
    },
    {
      prop: 'createTime',
      label: '创建时间',
      align: 'center',
      width: 180
    },
    {
      columnKey: 'action',
      label: '操作',
      width: 156,
      align: 'center',
      slot: 'action',
      hideInPrint: true,
      hideInExport: true
    }
  ]);

  /** 表格选中数据 */
  const selections = ref([]);

  /** 表格数据源 */
  const datasource = ({ pages, where, orders }) => {
    return pagePosts({ ...where, ...orders, ...pages });
  };

  /** 搜索 */
  const reload = (where, page) => {
    tableRef.value?.reload?.({ where, page });
  };

  /** 打开编辑弹窗 */
  const handleEdit = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/post-edit.vue'),
      componentProps: { data: row, onDone: () => reload() }
    });
  };

  /** 删除 */
  const handleRemove = (row) => {
    const rows = row == null ? selections.value : [row];
    if (!rows.length) {
      EleMessage.error({ message: '请至少选择一条数据', plain: true });
      return;
    }
    ElMessageBox.confirm(
      `是否确认删除岗位编码为"${rows.map((d) => d.postCode).join()}"的数据项?`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        removePosts(rows.map((d) => d.postId))
          .then(() => {
            loading.close();
            EleMessage.success({ message: '删除成功', plain: true });
            reload();
          })
          .catch((e) => {
            loading.close();
            EleMessage.error({ message: e.message, plain: true });
          });
      })
      .catch(() => {});
  };

  /** 导出数据 */
  const handleExport = () => {
    const loading = EleMessage.loading({ message: '请求中..', plain: true });
    tableRef.value?.fetch?.(({ where, orders }) => {
      exportPosts({ ...where, ...orders })
        .then(() => {
          loading.close();
        })
        .catch((e) => {
          loading.close();
          EleMessage.error({ message: e.message, plain: true });
        });
    });
  };
</script>
