<template>
  <ele-page flex-table="auto">
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        ref="tableRef"
        row-key="domainId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        cache-key="merchantDomainTable"
      >
        <template #toolbar>
          <btn-items
            :items="[
              {
                preset: 'add',
                permission: 'cloak:domain:add',
                onClick: () => handleAdd()
              }
            ]"
          />
        </template>
        <template #status="{ row }">
          <el-tag :type="statusType[row.status] || 'info'">
            {{ statusText[row.status] || row.status }}
          </el-tag>
        </template>
        <template #action="{ row }">
          <btn-items
            divider
            type="link"
            :items="[
              {
                title: '校验',
                permission: 'cloak:domain:verify',
                onClick: () => handleVerify(row)
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
  import {
    pageMerchantDomains,
    verifyMerchantDomain
  } from '@/api/merchant/domain';

  defineOptions({ name: 'MerchantDomain' });

  const { openModal } = useModal();

  const tableRef = ref(null);

  const statusText = {
    pending: '待验证',
    bound: '已绑定',
    abnormal: '异常',
    disabled: '已停用'
  };

  const statusType = {
    pending: 'warning',
    bound: 'success',
    abnormal: 'danger',
    disabled: 'info'
  };

  const columns = [
    { type: 'index', columnKey: 'index', width: 55, align: 'center' },
    { prop: 'domain', label: '域名', minWidth: 180 },
    { prop: 'verifyType', label: '解析类型', width: 110, align: 'center' },
    { prop: 'cloakCount', label: '关联斗篷', width: 110, align: 'center' },
    { prop: 'status', label: '状态', width: 110, align: 'center', slot: 'status' },
    { prop: 'lastCheckedAt', label: '最后检测', minWidth: 160, align: 'center' },
    { columnKey: 'action', label: '操作', width: 100, align: 'center', slot: 'action' }
  ];

  const datasource = ({ page, limit, where }) => {
    return pageMerchantDomains({ ...where, pageNum: page, pageSize: limit });
  };

  /** 刷新表格 */
  const reload = () => {
    tableRef.value?.reload?.();
  };

  /** 打开添加弹窗 */
  const handleAdd = () => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/domain-edit.vue'),
      componentProps: { onDone: () => reload() }
    });
  };

  /** 校验域名 */
  const handleVerify = (row) => {
    ElMessageBox.confirm(`是否确认校验域名"${row.domain}"?`, '系统提示', {
      type: 'warning',
      draggable: true
    })
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        verifyMerchantDomain(row.domainId)
          .then((data) => {
            loading.close();
            EleMessage.success({ message: data.msg, plain: true });
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
