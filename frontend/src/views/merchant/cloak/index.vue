<template>
  <ele-page flex-table="auto">
    <cloak-search @search="(where) => reload(where, 1)" />
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        ref="tableRef"
        row-key="cloakId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="false"
        :highlight-current-row="true"
        cache-key="merchantCloakTableV4"
      >
        <template #toolbar>
          <btn-items
            :items="[
              {
                preset: 'add',
                permission: 'cloak:strategy:add',
                onClick: () => handleAdd()
              }
            ]"
          />
        </template>
        <template #linkInfo="{ row }">
          <div class="cloak-link-info">
            <div class="cloak-link-row">
              <span class="cloak-link-label">真实链接:</span>
              <span class="cloak-link-url" :title="row.blackUrl">{{
                row.blackUrl
              }}</span>
              <ele-copyable
                v-if="row.blackUrl"
                :text="row.blackUrl"
                class="cloak-link-copy"
              />
            </div>
            <div class="cloak-link-row">
              <span class="cloak-link-label">审核链接:</span>
              <span class="cloak-link-url" :title="row.whiteUrl">{{
                row.whiteUrl
              }}</span>
              <ele-copyable
                v-if="row.whiteUrl"
                :text="row.whiteUrl"
                class="cloak-link-copy"
              />
            </div>
            <div class="cloak-link-row">
              <span class="cloak-link-label">跳转短链:</span>
              <span class="cloak-link-url" :title="row.shortLinkUrl || '-'">
                {{ row.shortLinkUrl || '-' }}
              </span>
              <ele-copyable
                v-if="row.shortLinkUrl"
                :text="row.shortLinkUrl"
                class="cloak-link-copy"
              />
            </div>
          </div>
        </template>
        <template #targetRegions="{ row }">
          {{ formatCountryCodes(row.targetRegions) }}
        </template>
        <template #mode="{ row }">
          <el-tag size="small" type="primary" effect="light">
            {{ modeText[row.mode] || row.mode || '斗篷' }}
          </el-tag>
        </template>
        <template #linkStats="{ row }">
          <div class="cloak-link-stats">
            <div class="cloak-stat-row">
              <span class="cloak-stat-label">审核链接次数</span>
              <span class="cloak-stat-value">{{ row.whiteHits ?? 0 }}</span>
            </div>
            <div class="cloak-stat-row">
              <span class="cloak-stat-label">真实链接次数</span>
              <span class="cloak-stat-value">{{ row.blackHits ?? 0 }}</span>
            </div>
            <div class="cloak-stat-row">
              <span class="cloak-stat-label">今日审核链接次数</span>
              <span class="cloak-stat-value">{{
                row.todayWhiteHits ?? 0
              }}</span>
            </div>
            <div class="cloak-stat-row">
              <span class="cloak-stat-label">今日真实链接次数</span>
              <span class="cloak-stat-value">{{
                row.todayBlackHits ?? 0
              }}</span>
            </div>
          </div>
        </template>
        <template #status="{ row }">
          <el-tag
            :type="row.status === 'enabled' ? 'success' : 'info'"
            size="small"
          >
            {{ row.status === 'enabled' ? '启用' : '停用' }}
          </el-tag>
        </template>
        <template #remark="{ row }">
          {{ row.remark || '-' }}
        </template>
        <template #action="{ row }">
          <div class="cloak-action-cell">
            <btn-items
              divider
              type="link"
              :items="[
                {
                  preset: 'edit',
                  permission: 'cloak:strategy:edit',
                  onClick: () => handleEdit(row)
                },
                {
                  title: '复制',
                  permission: 'cloak:strategy:copy',
                  onClick: () => handleCopy(row)
                },
                {
                  title: row.status === 'enabled' ? '停用' : '启用',
                  permission: 'cloak:strategy:status',
                  onClick: () => handleStatus(row)
                },
                {
                  preset: 'del',
                  permission: 'cloak:strategy:remove',
                  onClick: () => handleRemove(row)
                }
              ]"
            />
          </div>
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
    pageMerchantCloaks,
    updateMerchantCloakStatus,
    removeMerchantCloak
  } from '@/api/merchant/cloak';
  import CloakSearch from './components/cloak-search.vue';
  import { formatCountryCodes } from '@/utils/country-regions';

  defineOptions({ name: 'MerchantCloak' });

  const { openModal } = useModal();

  const tableRef = ref(null);

  const modeText = {
    cloak: '斗篷',
    green: '绿色',
    full_open: '全开',
    review: '审核'
  };

  const columns = [
    { prop: 'cloakName', label: '名称', minWidth: 90, fixed: 'left' },
    {
      columnKey: 'linkInfo',
      label: '链接信息',
      minWidth: 260,
      slot: 'linkInfo'
    },
    {
      prop: 'status',
      label: '状态',
      width: 80,
      align: 'center',
      slot: 'status'
    },
    {
      columnKey: 'targetRegions',
      label: '投放地区',
      minWidth: 120,
      align: 'center',
      slot: 'targetRegions',
      showOverflowTooltip: true
    },
    {
      prop: 'mode',
      label: '模式',
      width: 88,
      align: 'center',
      slot: 'mode'
    },
    {
      prop: 'remark',
      label: '备注',
      minWidth: 100,
      slot: 'remark',
      showOverflowTooltip: true
    },
    {
      columnKey: 'linkStats',
      label: '链接数据',
      minWidth: 210,
      slot: 'linkStats'
    },
    {
      prop: 'createdAt',
      label: '创建时间',
      width: 168,
      align: 'center'
    },
    {
      prop: 'deviceWhitelist',
      label: '设备白名单',
      width: 118,
      align: 'center',
      showOverflowTooltip: true
    },
    {
      prop: 'blockPc',
      label: '屏蔽PC',
      width: 88,
      align: 'center'
    },
    {
      prop: 'blockProxy',
      label: '屏蔽代理',
      width: 92,
      align: 'center'
    },
    {
      prop: 'allowedSourcesText',
      label: '允许访问来源',
      minWidth: 130,
      align: 'center',
      showOverflowTooltip: true
    },
    {
      prop: 'intermediatePage',
      label: '中间页',
      width: 100,
      align: 'center'
    },
    {
      prop: 'intermediatePageContent',
      label: '中间页内容',
      width: 108,
      align: 'center'
    },
    {
      columnKey: 'action',
      label: '操作',
      width: 248,
      align: 'center',
      slot: 'action',
      fixed: 'right'
    }
  ];

  const datasource = ({ page, limit, where }) => {
    return pageMerchantCloaks({ ...where, pageNum: page, pageSize: limit });
  };

  const reload = (where, page) => {
    tableRef.value?.reload?.({ where, page });
  };

  const handleAdd = () => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/cloak-edit.vue'),
      componentProps: { onDone: () => reload() }
    });
  };

  const handleEdit = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/cloak-edit.vue'),
      componentProps: { data: row, onDone: () => reload() }
    });
  };

  const handleCopy = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/cloak-edit.vue'),
      componentProps: { data: row, copy: true, onDone: () => reload() }
    });
  };

  const handleRemove = (row) => {
    ElMessageBox.confirm(
      `是否确认删除斗篷"${row.cloakName}"? 删除后不可恢复。`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        removeMerchantCloak(row.cloakId)
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

  const handleStatus = (row) => {
    const nextStatus = row.status === 'enabled' ? 'disabled' : 'enabled';
    const actionText = nextStatus === 'enabled' ? '启用' : '停用';
    ElMessageBox.confirm(
      `是否确认${actionText}斗篷"${row.cloakName}"?`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        updateMerchantCloakStatus(row.cloakId, nextStatus)
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
  .cloak-action-cell {
    white-space: nowrap;
  }

  .cloak-action-cell :deep(.ele-btn-items) {
    flex-wrap: nowrap;
  }

  .cloak-link-info {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 4px 0;
  }

  .cloak-link-row {
    display: flex;
    align-items: center;
    gap: 4px;
    line-height: 20px;
    font-size: 13px;
  }

  .cloak-link-label {
    flex-shrink: 0;
    color: var(--el-text-color-secondary);
  }

  .cloak-link-url {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--el-text-color-regular);
  }

  .cloak-link-copy {
    flex-shrink: 0;
  }

  .cloak-link-stats {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 4px 0;
  }

  .cloak-stat-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    line-height: 20px;
    font-size: 13px;
  }

  .cloak-stat-label {
    flex-shrink: 0;
    padding: 0 8px;
    border-radius: 4px;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
  }

  .cloak-stat-value {
    min-width: 36px;
    text-align: right;
    color: var(--el-text-color-primary);
    font-variant-numeric: tabular-nums;
  }
</style>
