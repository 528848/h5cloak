<template>
  <ele-page flex-table="auto">
    <api-key-search @search="(where) => reload(where, 1)" />
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        ref="tableRef"
        row-key="apiKeyId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        cache-key="merchantApiKeyTableV3"
      >
        <template #toolbar>
          <div class="api-key-toolbar">
            <btn-items
              :items="[
                {
                  preset: 'add',
                  permission: 'cloak:apiKey:add',
                  onClick: () => handleAdd()
                }
              ]"
            />
            <el-tooltip content="使用教程" placement="top">
              <el-button circle class="api-key-guide-btn" @click="goGuide">
                <el-icon><QuestionFilled /></el-icon>
              </el-button>
            </el-tooltip>
          </div>
        </template>
        <template #mode="{ row }">
          <el-tag size="small" type="primary" effect="light">
            {{ row.modeText || '斗篷' }}
          </el-tag>
        </template>
        <template #status="{ row }">
          <el-tag :type="row.status === 'enabled' ? 'success' : 'info'">
            {{ row.status === 'enabled' ? '启用' : '停用' }}
          </el-tag>
        </template>
        <template #lastCalledAt="{ row }">
          {{ row.lastCalledAt || '-' }}
        </template>
        <template #action="{ row }">
          <btn-items
            divider
            type="link"
            :items="[
              {
                title: '重置',
                permission: 'cloak:apiKey:reset',
                onClick: () => handleReset(row)
              },
              {
                preset: 'del',
                permission: 'cloak:apiKey:remove',
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
  import { ref, nextTick } from 'vue';
  import { useRouter } from 'vue-router';
  import { ElMessageBox } from 'element-plus';
  import { QuestionFilled } from '@element-plus/icons-vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import {
    pageApiKeys,
    resetApiKey,
    removeApiKey
  } from '@/api/merchant/api-key';
  import ApiKeySearch from './components/api-key-search.vue';

  defineOptions({ name: 'MerchantApiKey' });

  const router = useRouter();
  const { openModal } = useModal();
  const tableRef = ref(null);

  const goGuide = () => {
    router.push('/merchant/api-key/guide');
  };

  const openSecretModal = (secret) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/api-key-secret.vue'),
      componentProps: { secret }
    });
  };

  const columns = [
    { type: 'index', columnKey: 'index', width: 55, align: 'center' },
    { prop: 'name', label: 'Key 名称', minWidth: 140 },
    { prop: 'mode', label: '模式', width: 90, align: 'center', slot: 'mode' },
    { prop: 'keyPrefix', label: 'Token 前缀', minWidth: 120 },
    { prop: 'remark', label: '备注', minWidth: 120, showOverflowTooltip: true },
    { prop: 'status', label: '状态', width: 90, align: 'center', slot: 'status' },
    {
      prop: 'createdAt',
      label: '创建时间',
      width: 168,
      align: 'center'
    },
    {
      prop: 'lastCalledAt',
      label: '最后调用',
      minWidth: 160,
      align: 'center',
      slot: 'lastCalledAt'
    },
    { columnKey: 'action', label: '操作', width: 150, align: 'center', slot: 'action' }
  ];

  const datasource = ({ page, limit, where }) => {
    return pageApiKeys({ ...where, pageNum: page, pageSize: limit });
  };

  const reload = (where, page) => {
    tableRef.value?.reload?.({ where, page });
  };

  const handleAdd = () => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/api-key-edit.vue'),
      componentProps: {
        onDone: (secret) => {
          reload(undefined, 1);
          if (secret) {
            nextTick(() => {
              openSecretModal(secret);
            });
          }
        }
      }
    });
  };

  const handleReset = (row) => {
    ElMessageBox.confirm(
      `是否确认重置 API Key"${row.name}"? 重置后旧 Token 将立即失效。`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        resetApiKey(row.apiKeyId)
          .then((res) => {
            loading.close();
            EleMessage.success({ message: res.msg, plain: true });
            reload();
            const secret = res.data?.secret;
            if (secret) {
              nextTick(() => {
                openSecretModal(secret);
              });
            }
          })
          .catch((e) => {
            loading.close();
            EleMessage.error({ message: e.message, plain: true });
          });
      })
      .catch(() => {});
  };

  const handleRemove = (row) => {
    ElMessageBox.confirm(
      `是否确认删除 API Key"${row.name}"? 删除后不可恢复。`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        removeApiKey(row.apiKeyId)
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

<style scoped>
  .api-key-toolbar {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }

  .api-key-guide-btn {
    --el-button-size: 32px;
  }
</style>
