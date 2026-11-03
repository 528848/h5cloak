<template>
  <ele-page flex-table="auto">
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        ref="tableRef"
        row-key="merchantId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        cache-key="adminMerchantTable"
      >
        <template #toolbar>
          <btn-items
            :items="[
              {
                preset: 'add',
                permission: 'merchant:account:add',
                onClick: () => handleAdd()
              }
            ]"
          />
          <ele-text
            v-if="portalLoginUrl"
            type="secondary"
            style="margin-left: 12px"
          >
            商户端统一登录地址：
            <el-link
              :href="portalLoginUrl"
              target="_blank"
              type="primary"
              :underline="false"
            >
              {{ portalLoginUrl }}
            </el-link>
            <ele-copyable :text="portalLoginUrl" style="margin-left: 4px" />
          </ele-text>
        </template>
        <template #loginInfo="{ row }">
          <div class="merchant-login-info">
            <div class="merchant-login-url">
              <el-link
                :href="getLoginUrl(row)"
                target="_blank"
                type="primary"
                :underline="false"
              >
                {{ getLoginUrl(row) }}
              </el-link>
              <ele-copyable :text="getLoginUrl(row)" style="margin-left: 4px" />
            </div>
            <ele-text v-if="row.adminUsername" type="placeholder" size="sm">
              登录账号：{{ row.adminUsername }}
              <ele-copyable
                :text="row.adminUsername"
                style="margin-left: 4px; vertical-align: -2px"
              />
            </ele-text>
            <ele-text v-else type="placeholder" size="sm">未配置管理员账号</ele-text>
          </div>
        </template>
        <template #status="{ row }">
          <el-tag :type="row.status === 'enabled' ? 'success' : 'info'">
            {{ row.status === 'enabled' ? '启用' : '停用' }}
          </el-tag>
        </template>
        <template #action="{ row }">
          <btn-items
            divider
            type="link"
            :items="[
              {
                title: row.status === 'enabled' ? '停用' : '启用',
                permission: 'merchant:account:status',
                onClick: () => handleStatus(row)
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
  import { pageMerchants, updateMerchantStatus } from '@/api/admin/merchant';

  defineOptions({ name: 'AdminMerchant' });

  const { openModal } = useModal();

  const tableRef = ref(null);

  /** 商户端登录地址 */
  const portalLoginUrl = ref('');

  const fallbackLoginUrl = `${window.location.origin}${
    import.meta.env.BASE_URL || '/'
  }login`.replace(/([^:]\/)\/+/g, '$1');

  const columns = [
    { type: 'index', columnKey: 'index', width: 55, align: 'center' },
    { prop: 'merchantName', label: '商户名称', minWidth: 140 },
    {
      columnKey: 'loginInfo',
      label: '登录信息',
      minWidth: 260,
      slot: 'loginInfo'
    },
    { prop: 'contactName', label: '联系人', minWidth: 100 },
    { prop: 'domainCount', label: '域名数', width: 90, align: 'center' },
    { prop: 'cloakCount', label: '斗篷数', width: 90, align: 'center' },
    { prop: 'status', label: '状态', width: 90, align: 'center', slot: 'status' },
    { prop: 'createdAt', label: '创建时间', minWidth: 160, align: 'center' },
    { columnKey: 'action', label: '操作', width: 100, align: 'center', slot: 'action' }
  ];

  const getLoginUrl = (row) => {
    return row?.loginUrl || portalLoginUrl.value || fallbackLoginUrl;
  };

  const datasource = ({ page, limit, where }) => {
    return pageMerchants({ ...where, pageNum: page, pageSize: limit }).then(
      (data) => {
        if (data.loginUrl) {
          portalLoginUrl.value = data.loginUrl;
        } else if (!portalLoginUrl.value && data.rows?.[0]?.loginUrl) {
          portalLoginUrl.value = data.rows[0].loginUrl;
        }
        return data;
      }
    );
  };

  /** 刷新表格 */
  const reload = () => {
    tableRef.value?.reload?.();
  };

  /** 打开添加弹窗 */
  const handleAdd = () => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/merchant-edit.vue'),
      componentProps: { onDone: () => reload() }
    });
  };

  /** 修改商户状态 */
  const handleStatus = (row) => {
    const nextStatus = row.status === 'enabled' ? 'disabled' : 'enabled';
    const actionText = nextStatus === 'enabled' ? '启用' : '停用';
    ElMessageBox.confirm(
      `是否确认${actionText}商户"${row.merchantName}"?`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        updateMerchantStatus(row.merchantId, nextStatus)
          .then((msg) => {
            loading.close();
            EleMessage.success({ message: msg, plain: true });
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

<style lang="scss" scoped>
  .merchant-login-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
    line-height: 1.5;
  }

  .merchant-login-url {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 4px;
  }
</style>
