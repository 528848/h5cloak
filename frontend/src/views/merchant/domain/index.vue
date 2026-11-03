<template>
  <ele-page flex-table="auto">
    <summary-cards
      :items="summaryItems"
      :cols="4"
      clickable
      :active-key="activeStatus"
      @select="handleSummarySelect"
    />
    <domain-search ref="searchRef" @search="(where) => reload(where, 1)" />
    <ele-card
      class="table-section"
      :body-style="{ paddingTop: '8px' }"
      flex-table="auto"
    >
      <ele-pro-table
        ref="tableRef"
        row-key="domainId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        cache-key="merchantDomainTableV2"
      >
        <template #toolbar>
          <btn-items
            :items="[
              {
                preset: 'add',
                title: '添加域名',
                permission: 'site:domain:add',
                onClick: () => handleAdd()
              }
            ]"
          />
        </template>
        <template #domain="{ row }">
          <el-link type="primary" underline="never" @click="openGuide(row)">
            <span class="mono">{{ row.domain }}</span>
          </el-link>
        </template>
        <template #domainType="{ row }">
          {{ getOptionLabel(domainTypeOptions, row.domainType) }}
        </template>
        <template #status="{ row }">
          <el-tag
            :type="getOptionType(domainStatusOptions, row.status)"
            effect="light"
            size="small"
          >
            {{ getOptionLabel(domainStatusOptions, row.status) }}
          </el-tag>
        </template>
        <template #health="{ row }">
          <el-tag
            :type="getOptionType(healthOptions, row.healthStatus)"
            effect="plain"
            size="small"
          >
            {{ getOptionLabel(healthOptions, row.healthStatus) }}
          </el-tag>
        </template>
        <template #action="{ row }">
          <el-space :size="6" spacer="|">
            <el-button link type="primary" @click="openGuide(row)"
              >指引</el-button
            >
            <el-button
              v-permission="'site:domain:verify'"
              link
              type="primary"
              @click="handleVerify(row)"
            >
              校验
            </el-button>
            <el-button
              v-permission="'site:domain:remove'"
              link
              type="danger"
              @click="handleRemove(row)"
            >
              删除
            </el-button>
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
  import { getSiteStats } from '@/api/merchant/site';
  import {
    pageMerchantDomains,
    verifyMerchantDomain,
    removeMerchantDomain
  } from '@/api/merchant/domain';
  import DomainSearch from './components/domain-search.vue';
  import {
    domainStatusOptions,
    domainTypeOptions,
    healthOptions,
    getOptionLabel,
    getOptionType
  } from '../site/constants';

  defineOptions({ name: 'MerchantDomain' });

  const { openModal } = useModal();
  const tableRef = ref(null);
  const searchRef = ref(null);
  const activeStatus = ref('');
  const stats = ref({
    domainTotal: 0,
    domainBound: 0,
    domainPending: 0,
    domainAbnormal: 0
  });

  const summaryItems = computed(() => [
    {
      key: '',
      label: '全部域名',
      value: stats.value.domainTotal,
      desc: '已添加到站点的域名',
      type: 'primary'
    },
    {
      key: 'bound',
      label: '已绑定',
      value: stats.value.domainBound,
      desc: '校验通过，可用于访问',
      type: 'success'
    },
    {
      key: 'pending',
      label: '待校验',
      value: stats.value.domainPending,
      desc: '尚未按指引完成解析',
      type: 'warning'
    },
    {
      key: 'abnormal',
      label: '解析异常',
      value: stats.value.domainAbnormal,
      desc: '当前解析与期望不一致',
      type: 'danger',
      alert: stats.value.domainAbnormal > 0
    }
  ]);

  const columns = [
    {
      type: 'index',
      columnKey: 'index',
      width: 50,
      align: 'center',
      fixed: 'left'
    },
    {
      prop: 'domain',
      label: '域名',
      minWidth: 180,
      slot: 'domain',
      formatter: (row) => row.domain
    },
    { prop: 'siteName', label: '所属站点', minWidth: 140 },
    {
      prop: 'domainType',
      label: '类型',
      width: 100,
      align: 'center',
      slot: 'domainType',
      formatter: (row) => getOptionLabel(domainTypeOptions, row.domainType)
    },
    { prop: 'verifyType', label: '校验方式', width: 100, align: 'center' },
    {
      prop: 'status',
      label: '状态',
      width: 100,
      align: 'center',
      slot: 'status',
      formatter: (row) => getOptionLabel(domainStatusOptions, row.status)
    },
    {
      prop: 'healthStatus',
      label: '解析',
      width: 90,
      align: 'center',
      slot: 'health',
      formatter: (row) => getOptionLabel(healthOptions, row.healthStatus)
    },
    {
      prop: 'verifyMessage',
      label: '最近结果',
      minWidth: 220,
      formatter: (row) => row.verifyMessage || '-'
    },
    {
      prop: 'lastCheckedAt',
      label: '最后校验',
      minWidth: 160,
      formatter: (row) => row.lastCheckedAt || '-'
    },
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
    pageMerchantDomains({ ...where, pageNum: page, pageSize: limit });

  const loadStats = () => {
    getSiteStats()
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

  const openGuide = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/domain-guide.vue'),
      componentProps: { data: row }
    });
  };

  const handleAdd = () => {
    openModal({
      custom: true,
      asyncComponent: () => import('./components/domain-edit.vue'),
      componentProps: {
        onDone: (row) => {
          reload();
          if (row) openGuide(row);
        }
      }
    });
  };

  const handleVerify = (row) => {
    const loading = EleMessage.loading({ message: '校验中..', plain: true });
    verifyMerchantDomain(row.domainId)
      .then((res) => {
        loading.close();
        EleMessage.success({ message: res.msg, plain: true });
        reload();
        openGuide(res.data);
      })
      .catch((e) => {
        loading.close();
        EleMessage.error({ message: e.message, plain: true });
      });
  };

  const handleRemove = (row) => {
    ElMessageBox.confirm(`是否确认删除域名「${row.domain}」？`, '系统提示', {
      type: 'warning',
      draggable: true
    })
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        removeMerchantDomain(row.domainId)
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
