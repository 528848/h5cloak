<template>
  <ele-drawer
    :size="1200"
    title="调度日志"
    :body-style="{ padding: '14px 16px' }"
    flex-table="auto"
    v-bind="modalProps"
  >
    <job-log-search
      :data="data"
      style="margin-bottom: -8px"
      @search="(where) => reload(where, 1)"
    />
    <ele-pro-table
      ref="tableRef"
      row-key="jobLogId"
      :columns="columns"
      :datasource="datasource"
      :show-overflow-tooltip="true"
      :highlight-current-row="true"
      v-model:selections="selections"
      :export-config="{ fileName: '调度日志' }"
      cache-key="monitorJobLogTable"
    >
      <template #toolbar>
        <btn-items
          :items="[
            {
              preset: 'del',
              permission: 'monitor:job:remove',
              onClick: () => handleRemove()
            },
            {
              title: '清空',
              icon: CloseCircleOutlined,
              props: { type: 'danger', plain: true },
              permission: 'monitor:job:remove',
              onClick: () => handleRemoveAll()
            },
            {
              preset: 'export',
              permission: 'monitor:job:export',
              onClick: () => handleExport()
            }
          ]"
        />
      </template>
      <template #jobGroup="{ row }">
        <dict-data
          code="sys_job_group"
          type="tag"
          :model-value="row.jobGroup"
        />
      </template>
      <template #status="{ row }">
        <dict-data
          code="sys_common_status"
          type="tag"
          :model-value="row.status"
        />
      </template>
      <template #action="{ row }">
        <btn-items
          :divider="true"
          type="link"
          :items="[{ preset: 'detail', onClick: () => handleDetail(row) }]"
        />
      </template>
    </ele-pro-table>
  </ele-drawer>
</template>

<script setup>
  import { ref, computed } from 'vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { CloseCircleOutlined } from '@/components/icons';
  import { usePermission } from '@/utils/use-permission';
  import { useDictData } from '@/utils/use-dict-data';
  import JobLogSearch from './job-log-search.vue';
  import {
    pageJobLogs,
    removeJobLogs,
    exportJobLogs,
    clearJoblogs
  } from '@/api/monitor/job-log';

  const props = defineProps({
    /** 定时任务 */
    data: Object
  });

  const { modalProps, openModal } = useModal();

  const { hasPermission } = usePermission();

  /** 字典数据 */
  const [statusDicts] = useDictData(['sys_common_status']);

  /** 表格实例 */
  const tableRef = ref(null);

  /** 表格列配置 */
  const columns = computed(() => {
    const cols = [
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
        prop: 'jobName',
        label: '任务名称',
        align: 'center',
        minWidth: 110
      },
      {
        prop: 'jobGroup',
        label: '任务组名',
        align: 'center',
        minWidth: 110,
        slot: 'jobGroup'
      },
      {
        prop: 'invokeTarget',
        label: '调用目标字符串',
        align: 'center',
        minWidth: 140
      },
      {
        prop: 'jobMessage',
        label: '日志信息',
        align: 'center',
        minWidth: 110
      },
      {
        prop: 'status',
        label: '执行状态',
        width: 110,
        align: 'center',
        slot: 'status',
        filters: statusDicts.value.map((d) => {
          return { text: d.dictLabel, value: d.dictValue };
        }),
        filterMultiple: false
      },
      {
        prop: 'createTime',
        label: '执行时间',
        align: 'center',
        minWidth: 110
      }
    ];
    if (hasPermission('monitor:job:query')) {
      cols.push({
        columnKey: 'action',
        label: '操作',
        width: 90,
        align: 'center',
        slot: 'action',
        hideInPrint: true,
        hideInExport: true
      });
    }
    return cols;
  });

  /** 表格选中数据 */
  const selections = ref([]);

  /** 表格数据源 */
  const datasource = ({ pages, where, filters }) => {
    const params = { ...where, ...filters, ...pages };
    if (props.data) {
      params.jobName = props.data.jobName;
      params.jobGroup = props.data.jobGroup;
    }
    return pageJobLogs(params);
  };

  /** 搜索 */
  const reload = (where, page) => {
    tableRef.value?.reload?.({ where, page });
  };

  /** 删除 */
  const handleRemove = () => {
    if (!selections.value.length) {
      EleMessage.error({ message: '请至少选择一条数据', plain: true });
      return;
    }
    const ids = selections.value.map((d) => d.jobLogId);
    ElMessageBox.confirm(
      `是否确认删除调度日志编号为"${ids.join()}"的数据项?`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        removeJobLogs(ids)
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
    tableRef.value?.fetch?.(({ where, filters }) => {
      const params = { ...where, ...filters };
      if (props.data) {
        params.jobName = props.data.jobName;
        params.jobGroup = props.data.jobGroup;
      }
      exportJobLogs(params)
        .then(() => {
          loading.close();
        })
        .catch((e) => {
          loading.close();
          EleMessage.error({ message: e.message, plain: true });
        });
    });
  };

  /** 清空 */
  const handleRemoveAll = () => {
    ElMessageBox.confirm('是否确认清空所有调度日志数据项？', '系统提示', {
      type: 'warning',
      draggable: true
    })
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        clearJoblogs()
          .then(() => {
            loading.close();
            EleMessage.success({ message: '清空成功', plain: true });
            reload();
          })
          .catch((e) => {
            loading.close();
            EleMessage.error({ message: e.message, plain: true });
          });
      })
      .catch(() => {});
  };

  /** 详情 */
  const handleDetail = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./job-log-detail.vue'),
      componentProps: { data: row }
    });
  };
</script>
