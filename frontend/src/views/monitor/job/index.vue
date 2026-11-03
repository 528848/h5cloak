<template>
  <ele-page>
    <!-- 搜索表单 -->
    <job-search @search="(where) => reload(where, 1)" />
    <ele-card :body-style="{ paddingTop: '8px' }">
      <!-- 表格 -->
      <ele-pro-table
        ref="tableRef"
        row-key="jobId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        v-model:selections="selections"
        :export-config="{ fileName: '定时任务' }"
        cache-key="monitorJobTable"
      >
        <template #toolbar>
          <btn-items
            :items="[
              {
                preset: 'add',
                permission: 'monitor:job:add',
                onClick: () => handleEdit()
              },
              {
                preset: 'del',
                permission: 'monitor:job:remove',
                props: { class: 'hidden-sm-and-down' },
                onClick: () => handleRemove()
              },
              {
                preset: 'export',
                permission: 'monitor:job:export',
                onClick: () => handleExport()
              },
              {
                title: '日志',
                icon: LogOutlined,
                permission: 'monitor:job:query',
                props: { type: 'default' },
                onClick: () => handleOpenLog()
              }
            ]"
          />
        </template>
        <template #status="{ row }">
          <el-switch
            size="small"
            :model-value="row.status == 0"
            @change="(checked) => handleEditStatus(checked, row)"
          />
        </template>
        <template #action="{ row }">
          <btn-items
            :divider="true"
            type="link"
            :items="[
              {
                preset: 'edit',
                permission: 'monitor:job:edit',
                onClick: () => handleEdit(row)
              },
              {
                preset: 'del',
                permission: 'monitor:job:remove',
                onClick: () => handleRemove(row)
              },
              {
                preset: 'more',
                dropdownItems: [
                  {
                    title: '执行一次',
                    icon: PlayFilled,
                    permission: 'monitor:job:changeStatus',
                    onClick: () => handleExecute(row)
                  },
                  {
                    preset: 'detail',
                    title: '查看详情',
                    permission: 'monitor:job:query',
                    divided: true,
                    onClick: () => handleDetail(row)
                  },
                  {
                    title: '调度日志',
                    icon: LogOutlined,
                    permission: 'monitor:job:query',
                    onClick: () => handleOpenLog(row)
                  }
                ]
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
  import { LogOutlined, PlayFilled } from '@/components/icons';
  import JobSearch from './components/job-search.vue';
  import {
    pageJobs,
    removeJobs,
    updateJobStatus,
    exportJobs,
    runJob
  } from '@/api/monitor/job';

  defineOptions({ name: 'MonitorJob' });

  const { openModal } = useModal();

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
      prop: 'jobName',
      label: '任务名称',
      align: 'center',
      minWidth: 110
    },
    {
      prop: 'jobGroup',
      label: '任务组名',
      align: 'center',
      minWidth: 110
    },
    {
      prop: 'invokeTarget',
      label: '调用目标字符串',
      align: 'center',
      minWidth: 140
    },
    {
      prop: 'cronExpression',
      label: 'cron执行表达式',
      align: 'center',
      minWidth: 140
    },
    {
      prop: 'status',
      label: '状态',
      width: 80,
      align: 'center',
      slot: 'status',
      formatter: (row) => (row.status == 0 ? '正常' : '暂停')
    },
    {
      columnKey: 'action',
      label: '操作',
      width: 220,
      align: 'center',
      slot: 'action',
      hideInPrint: true,
      hideInExport: true
    }
  ]);

  /** 表格选中数据 */
  const selections = ref([]);

  /** 表格数据源 */
  const datasource = ({ pages, where }) => {
    return pageJobs({ ...where, ...pages });
  };

  /** 搜索 */
  const reload = (where, page) => {
    tableRef.value?.reload?.({ where, page });
  };

  /** 打开编辑弹窗 */
  const handleEdit = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/job-edit.vue'),
      componentProps: { data: row, onDone: () => reload() }
    });
  };

  /** 详情 */
  const handleDetail = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/job-detail.vue'),
      componentProps: { data: row }
    });
  };

  /** 打开调度日志弹窗 */
  const handleOpenLog = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/job-log.vue'),
      componentProps: { data: row }
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
      `是否确认删除任务名称为"${rows.map((d) => d.jobName).join()}"的数据项?`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        removeJobs(rows.map((d) => d.jobId))
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

  /** 修改状态 */
  const handleEditStatus = (checked, row) => {
    const status = checked ? '0' : '1';
    updateJobStatus(row.jobId, status)
      .then((msg) => {
        row.status = status;
        EleMessage.success({ message: msg, plain: true });
      })
      .catch((e) => {
        EleMessage.error({ message: e.message, plain: true });
      });
  };

  /** 导出数据 */
  const handleExport = () => {
    const loading = EleMessage.loading({ message: '请求中..', plain: true });
    tableRef.value?.fetch?.(({ where, orders }) => {
      exportJobs({ ...where, ...orders })
        .then(() => {
          loading.close();
        })
        .catch((e) => {
          loading.close();
          EleMessage.error({ message: e.message, plain: true });
        });
    });
  };

  /** 执行一次 */
  const handleExecute = (row) => {
    ElMessageBox.confirm(
      `确认要立即执行一次"${row.jobName}"任务吗?`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        runJob(row.jobId, row.jobGroup)
          .then(() => {
            loading.close();
            EleMessage.success({ message: '执行成功', plain: true });
            reload();
          })
          .catch((e) => {
            loading.close();
            EleMessage.error({ message: e.message, plain: true });
          });
      })
      .catch(() => {});
  };
</script>
