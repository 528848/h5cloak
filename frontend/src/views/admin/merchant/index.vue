<template>
  <ele-page flex-table="auto">
    <summary-cards
      :items="summaryItems"
      :cols="4"
      clickable
      :active-key="activeStatus"
      @select="handleSummarySelect"
    />
    <merchant-search ref="searchRef" @search="(where) => reload(where, 1)" />
    <ele-card
      class="table-section"
      :body-style="{ paddingTop: '8px' }"
      flex-table="auto"
    >
      <ele-pro-table
        ref="tableRef"
        row-key="merchantId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        cache-key="adminMerchantTableV2"
      >
        <template #toolbar>
          <btn-items
            :items="[
              {
                preset: 'add',
                title: '开通商户',
                permission: 'merchant:account:add',
                onClick: () => handleAdd()
              }
            ]"
          />
        </template>
        <template #merchantName="{ row }">
          <el-link type="primary" underline="never" @click="handleDetail(row)">
            {{ row.merchantName }}
          </el-link>
        </template>
        <template #adminUsername="{ row }">
          <span v-if="row.adminUsername" class="mono">{{
            row.adminUsername
          }}</span>
          <span v-else class="muted-text">未配置</span>
        </template>
        <template #status="{ row }">
          <el-tag
            :type="getOptionType(merchantStatusOptions, row.effectiveStatus)"
            effect="light"
          >
            {{ getOptionLabel(merchantStatusOptions, row.effectiveStatus) }}
          </el-tag>
        </template>
        <template #expireAt="{ row }">
          <span v-if="row.expireAt">{{ row.expireAt }}</span>
          <span v-else class="muted-text">长期</span>
        </template>
        <template #quota="{ row }">
          <span class="muted-text">
            站点 {{ row.quota.siteQuota }} / 域名 {{ row.quota.domainQuota }} /
            判定 {{ formatNumber(row.quota.decisionQuota) }}/月
          </span>
        </template>
        <template #action="{ row }">
          <el-space :size="6" spacer="|">
            <el-button link type="primary" @click="handleDetail(row)"
              >详情</el-button
            >
            <el-button
              v-permission="'merchant:account:edit'"
              link
              type="primary"
              @click="handleEdit(row)"
            >
              编辑
            </el-button>
            <el-dropdown
              trigger="click"
              @command="(cmd) => handleCommand(cmd, row)"
            >
              <el-button link type="primary">更多</el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item
                    v-permission="'merchant:account:quota'"
                    command="quota"
                  >
                    额度与功能
                  </el-dropdown-item>
                  <el-dropdown-item
                    v-permission="'merchant:account:resetPwd'"
                    command="resetPwd"
                  >
                    重置管理员密码
                  </el-dropdown-item>
                  <el-dropdown-item
                    v-permission="'merchant:account:impersonate'"
                    command="impersonate"
                  >
                    模拟登录
                  </el-dropdown-item>
                  <el-dropdown-item
                    v-if="row.status !== 'enabled'"
                    v-permission="'merchant:account:status'"
                    command="enabled"
                    divided
                  >
                    启用
                  </el-dropdown-item>
                  <el-dropdown-item
                    v-if="row.status === 'enabled'"
                    v-permission="'merchant:account:status'"
                    command="disabled"
                    divided
                  >
                    停用
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </el-space>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref, computed, onMounted } from 'vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import {
    pageMerchants,
    getMerchantStats,
    updateMerchantStatus
  } from '@/api/admin/merchant';
  import { impersonateMerchant } from '@/api/admin/platform';
  import MerchantSearch from './components/merchant-search.vue';
  import {
    merchantStatusOptions,
    getOptionLabel,
    getOptionType
  } from './constants';

  defineOptions({ name: 'AdminMerchant' });

  const { openModal } = useModal();
  const tableRef = ref(null);
  const searchRef = ref(null);
  const stats = ref({
    total: 0,
    enabled: 0,
    disabled: 0,
    frozen: 0,
    expired: 0
  });
  const activeStatus = ref('');

  const summaryItems = computed(() => [
    {
      key: '',
      label: '全部商户',
      value: stats.value.total,
      desc: '平台开通的合作商户',
      type: 'primary'
    },
    {
      key: 'enabled',
      label: '启用中',
      value: stats.value.enabled,
      desc: '可登录并正常判定',
      type: 'success'
    },
    {
      key: 'disabled',
      label: '已停用',
      value: stats.value.disabled,
      desc: '登录与判定均停止',
      type: 'info'
    },
    {
      key: 'frozen',
      label: '已冻结',
      value: stats.value.frozen,
      desc: '平台强制冻结',
      type: 'danger',
      alert: stats.value.frozen > 0
    }
  ]);

  const formatNumber = (n) => (n == null ? '-' : Number(n).toLocaleString());

  const columns = [
    {
      type: 'index',
      columnKey: 'index',
      width: 50,
      align: 'center',
      fixed: 'left'
    },
    {
      prop: 'merchantName',
      label: '商户名称',
      minWidth: 160,
      slot: 'merchantName',
      formatter: (row) => row.merchantName
    },
    {
      prop: 'adminUsername',
      label: '管理员账号',
      minWidth: 130,
      slot: 'adminUsername',
      formatter: (row) => row.adminUsername || '-'
    },
    {
      prop: 'contactName',
      label: '联系人',
      minWidth: 100,
      formatter: (row) => row.contactName || '-'
    },
    {
      prop: 'contactEmail',
      label: '邮箱',
      minWidth: 160,
      formatter: (row) => row.contactEmail || '-'
    },
    {
      prop: 'status',
      label: '状态',
      width: 90,
      align: 'center',
      slot: 'status',
      formatter: (row) =>
        getOptionLabel(merchantStatusOptions, row.effectiveStatus)
    },
    {
      prop: 'expireAt',
      label: '到期时间',
      minWidth: 150,
      slot: 'expireAt',
      formatter: (row) => row.expireAt || '长期'
    },
    {
      columnKey: 'quota',
      label: '额度',
      minWidth: 220,
      slot: 'quota',
      formatter: (row) =>
        `站点 ${row.quota.siteQuota} / 域名 ${row.quota.domainQuota} / 判定 ${row.quota.decisionQuota}/月`
    },
    { prop: 'createdAt', label: '开通时间', minWidth: 160 },
    {
      columnKey: 'action',
      label: '操作',
      width: 180,
      align: 'center',
      fixed: 'right',
      slot: 'action',
      hideInPrint: true,
      hideInExport: true
    }
  ];

  const datasource = ({ page, limit, where }) =>
    pageMerchants({ ...where, pageNum: page, pageSize: limit });

  const loadStats = () => {
    getMerchantStats()
      .then((data) => (stats.value = data))
      .catch(() => {});
  };

  const reload = (where, page) => {
    tableRef.value?.reload?.({ where, page });
    loadStats();
  };

  const handleSummarySelect = (item) => {
    activeStatus.value = item.key;
    searchRef.value?.setStatus(item.key);
  };

  const handleAdd = () => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/merchant-edit.vue'),
      componentProps: { onDone: () => reload() }
    });
  };

  const handleEdit = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/merchant-edit.vue'),
      componentProps: { data: row, onDone: () => reload() }
    });
  };

  const handleDetail = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/merchant-detail.vue'),
      componentProps: { merchantId: row.merchantId, onDone: () => reload() }
    });
  };

  const handleCommand = (cmd, row) => {
    if (cmd === 'impersonate') {
      impersonateMerchant(row.merchantId)
        .then((data) => {
          window.open(data.url, '_blank');
          EleMessage.success({
            message: '已打开商户后台，链接 60 秒内有效',
            plain: true
          });
        })
        .catch((e) => EleMessage.error({ message: e.message, plain: true }));
      return;
    }
    if (cmd === 'quota') {
      openModal({
        custom: true,
        asyncComponent: () => import('./components/merchant-quota.vue'),
        componentProps: { data: row, onDone: () => reload() }
      });
      return;
    }
    if (cmd === 'resetPwd') {
      openModal({
        custom: true,
        asyncComponent: () =>
          import('./components/merchant-reset-password.vue'),
        componentProps: { data: row }
      });
      return;
    }
    handleStatus(row, cmd);
  };

  const handleStatus = (row, status) => {
    const text = getOptionLabel(merchantStatusOptions, status);
    const extra =
      status === 'enabled'
        ? ''
        : '，该商户下所有账号将被强制下线，判定接口停止服务';
    ElMessageBox.confirm(
      `是否确认${text}商户「${row.merchantName}」${extra}？`,
      '系统提示',
      {
        type: 'warning',
        draggable: true
      }
    )
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        updateMerchantStatus(row.merchantId, status)
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

  onMounted(loadStats);
</script>
