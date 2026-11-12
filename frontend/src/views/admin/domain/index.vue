<template>
  <ele-page flex-table="auto">
    <summary-cards :items="summaryItems" :cols="4" />
    <ele-card search-form class="page-section">
      <el-form
        label-width="72px"
        @keyup.enter="handleSearch"
        @submit.prevent=""
      >
        <el-row :gutter="12">
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="域名">
              <el-input
                v-model.trim="form.keyword"
                clearable
                placeholder="域名关键字"
              />
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="所属商户">
              <el-select
                v-model="form.merchantId"
                filterable
                remote
                clearable
                reserve-keyword
                placeholder="输入商户名称搜索"
                :remote-method="searchMerchants"
                :loading="merchantLoading"
                class="ele-fluid"
              >
                <el-option
                  v-for="item in merchantOptions"
                  :key="item.merchantId"
                  :label="item.merchantName"
                  :value="item.merchantId"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="状态">
              <el-select
                v-model="form.status"
                clearable
                placeholder="全部"
                class="ele-fluid"
              >
                <el-option
                  v-for="o in domainStatusOptions"
                  :key="o.value"
                  :label="o.label"
                  :value="o.value"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label-width="0">
              <btn-items
                :wrap="false"
                :items="[
                  { preset: 'search', onClick: () => handleSearch() },
                  { preset: 'reset', onClick: () => handleReset() }
                ]"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </ele-card>
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
        cache-key="adminDomainTableV2"
      >
        <template #domain="{ row }">
          <span class="mono">{{ row.domain }}</span>
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
        <template #action="{ row }">
          <el-button
            v-permission="'site:domain:verify'"
            link
            type="primary"
            @click="handleVerify(row)"
          >
            校验
          </el-button>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref, computed, onMounted } from 'vue';
  import { EleMessage } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { pageAdminDomains, verifyAdminDomain } from '@/api/admin/domain';
  import { pageMerchants } from '@/api/admin/merchant';
  import { getAdminSiteStats } from '@/api/admin/site';
  import {
    domainStatusOptions,
    domainTypeOptions,
    getOptionLabel,
    getOptionType
  } from '@/views/merchant/site/constants';

  defineOptions({ name: 'AdminDomain' });

  const tableRef = ref(null);
  const stats = ref({
    domainTotal: 0,
    domainBound: 0,
    domainPending: 0,
    domainAbnormal: 0
  });
  const merchantOptions = ref([]);
  const merchantLoading = ref(false);
  let merchantTimer;
  const [form, resetFields] = useFormData({
    keyword: '',
    status: '',
    merchantId: undefined
  });

  const searchMerchants = (keyword) => {
    clearTimeout(merchantTimer);
    merchantTimer = setTimeout(async () => {
      merchantLoading.value = true;
      try {
        const res = await pageMerchants({
          keyword: (keyword || '').trim(),
          pageNum: 1,
          pageSize: 20
        });
        merchantOptions.value = res.rows || [];
      } catch (e) {
        merchantOptions.value = [];
      } finally {
        merchantLoading.value = false;
      }
    }, 300);
  };

  const summaryItems = computed(() => [
    {
      key: 'total',
      label: '全部域名',
      value: stats.value.domainTotal,
      desc: '全平台已添加的域名',
      type: 'primary'
    },
    {
      key: 'bound',
      label: '已绑定',
      value: stats.value.domainBound,
      desc: '校验通过',
      type: 'success'
    },
    {
      key: 'pending',
      label: '待校验',
      value: stats.value.domainPending,
      desc: '尚未完成解析',
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
    { type: 'index', columnKey: 'index', width: 50, align: 'center' },
    {
      prop: 'domain',
      label: '域名',
      minWidth: 180,
      slot: 'domain',
      formatter: (row) => row.domain
    },
    {
      prop: 'merchantName',
      label: '所属商户',
      minWidth: 140,
      formatter: (row) => row.merchantName || '-'
    },
    { prop: 'siteName', label: '所属站点', minWidth: 140 },
    {
      prop: 'domainType',
      label: '类型',
      width: 100,
      align: 'center',
      formatter: (row) => getOptionLabel(domainTypeOptions, row.domainType)
    },
    { prop: 'verifyType', label: '校验方式', width: 100, align: 'center' },
    {
      prop: 'status',
      label: '状态',
      width: 110,
      align: 'center',
      slot: 'status',
      formatter: (row) => getOptionLabel(domainStatusOptions, row.status)
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
      width: 90,
      align: 'center',
      slot: 'action',
      hideInPrint: true,
      hideInExport: true
    }
  ];

  const datasource = ({ page, limit, where }) =>
    pageAdminDomains({ ...where, pageNum: page, pageSize: limit });

  const loadStats = () => {
    getAdminSiteStats()
      .then((data) => (stats.value = data))
      .catch(() => {});
  };

  const reload = (where, page) => {
    tableRef.value?.reload?.({ where, page });
    loadStats();
  };

  const handleSearch = () => reload({ ...form }, 1);
  const handleReset = () => {
    resetFields();
    handleSearch();
  };

  const handleVerify = (row) => {
    const loading = EleMessage.loading({ message: '校验中..', plain: true });
    verifyAdminDomain(row.domainId)
      .then((res) => {
        loading.close();
        EleMessage.success({
          message: res.data?.verifyMessage || res.msg,
          plain: true
        });
        reload();
      })
      .catch((e) => {
        loading.close();
        EleMessage.error({ message: e.message, plain: true });
      });
  };

  onMounted(loadStats);
</script>
