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
            <el-form-item label="关键字">
              <el-input
                v-model.trim="form.keyword"
                clearable
                placeholder="站点名称"
              />
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
                  v-for="o in siteStatusOptions"
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
        row-key="siteId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        cache-key="adminSiteTable"
      >
        <template #siteType="{ row }">
          <el-tag
            :type="getOptionType(siteTypeOptions, row.siteType)"
            effect="plain"
            size="small"
          >
            {{ getOptionLabel(siteTypeOptions, row.siteType) }}
          </el-tag>
        </template>
        <template #status="{ row }">
          <el-tag
            :type="getOptionType(siteStatusOptions, row.status)"
            effect="light"
            size="small"
          >
            {{ getOptionLabel(siteStatusOptions, row.status) }}
          </el-tag>
          <el-tag
            v-if="row.platformHold"
            type="danger"
            effect="plain"
            size="small"
            style="margin-left: 6px"
          >
            平台停用
          </el-tag>
        </template>
        <template #domains="{ row }">
          {{ row.boundDomainCount || 0 }} / {{ row.domainCount || 0 }}
        </template>
        <template #action="{ row }">
          <el-button
            v-permission="'site:site:status'"
            link
            :type="row.status === 'enabled' ? 'danger' : 'success'"
            @click="handleStatus(row)"
          >
            {{ row.status === 'enabled' ? '强制停用' : '恢复启用' }}
          </el-button>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref, computed, onMounted } from 'vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import {
    pageAdminSites,
    getAdminSiteStats,
    updateAdminSiteStatus
  } from '@/api/admin/site';
  import {
    siteStatusOptions,
    siteTypeOptions,
    getOptionLabel,
    getOptionType
  } from '@/views/merchant/site/constants';

  defineOptions({ name: 'AdminSite' });

  const tableRef = ref(null);
  const stats = ref({ total: 0, enabled: 0, disabled: 0, domainAbnormal: 0 });
  const [form, resetFields] = useFormData({ keyword: '', status: '' });

  const summaryItems = computed(() => [
    {
      key: 'total',
      label: '全部站点',
      value: stats.value.total,
      desc: '合作商户创建的站点',
      type: 'primary'
    },
    {
      key: 'enabled',
      label: '启用中',
      value: stats.value.enabled,
      desc: '可进行访问归因',
      type: 'success'
    },
    {
      key: 'disabled',
      label: '已停用',
      value: stats.value.disabled,
      desc: '含平台强制停用',
      type: 'info'
    },
    {
      key: 'abnormal',
      label: '域名异常',
      value: stats.value.domainAbnormal,
      desc: '解析与期望不一致',
      type: 'danger',
      alert: stats.value.domainAbnormal > 0
    }
  ]);

  const columns = [
    { type: 'index', columnKey: 'index', width: 50, align: 'center' },
    { prop: 'name', label: '站点名称', minWidth: 160 },
    {
      prop: 'merchantName',
      label: '所属商户',
      minWidth: 140,
      formatter: (row) => row.merchantName || '-'
    },
    {
      prop: 'siteType',
      label: '接入方式',
      width: 110,
      align: 'center',
      slot: 'siteType',
      formatter: (row) => getOptionLabel(siteTypeOptions, row.siteType)
    },
    {
      prop: 'status',
      label: '状态',
      width: 150,
      align: 'center',
      slot: 'status',
      formatter: (row) => getOptionLabel(siteStatusOptions, row.status)
    },
    {
      columnKey: 'domains',
      label: '域名（已绑定/全部）',
      width: 160,
      align: 'center',
      slot: 'domains',
      formatter: (row) =>
        `${row.boundDomainCount || 0} / ${row.domainCount || 0}`
    },
    {
      prop: 'publishableKey',
      label: '公钥',
      minWidth: 180,
      formatter: (row) => row.publishableKey
    },
    { prop: 'createdAt', label: '创建时间', minWidth: 160 },
    {
      columnKey: 'action',
      label: '操作',
      width: 120,
      align: 'center',
      slot: 'action',
      hideInPrint: true,
      hideInExport: true
    }
  ];

  const datasource = ({ page, limit, where }) =>
    pageAdminSites({ ...where, pageNum: page, pageSize: limit });

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

  const handleStatus = (row) => {
    const next = row.status === 'enabled' ? 'disabled' : 'enabled';
    const text = next === 'enabled' ? '恢复启用' : '强制停用';
    ElMessageBox.confirm(
      `是否确认${text}站点「${row.name}」？停用后该站点立即停止判定。`,
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => {
        const loading = EleMessage.loading({
          message: '请求中..',
          plain: true
        });
        updateAdminSiteStatus(row.siteId, next)
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
