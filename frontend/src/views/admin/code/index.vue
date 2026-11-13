<template>
  <ele-page flex-table="auto">
    <ele-card search-form class="page-section">
      <el-form label-width="72px" @keyup.enter="reload" @submit.prevent="">
        <el-row :gutter="12">
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="关键词">
              <el-input
                v-model.trim="form.keyword"
                clearable
                placeholder="归因码 / 名称"
              />
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="商户">
              <el-select
                v-model="form.merchantId"
                clearable
                filterable
                placeholder="全部"
                class="ele-fluid"
                @change="onMerchantChange"
              >
                <el-option
                  v-for="m in merchants"
                  :key="m.merchantId"
                  :label="m.merchantName"
                  :value="m.merchantId"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="站点">
              <el-select
                v-model="form.siteId"
                clearable
                filterable
                placeholder="全部"
                class="ele-fluid"
              >
                <el-option
                  v-for="s in sites"
                  :key="s.siteId"
                  :label="s.name"
                  :value="s.siteId"
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
                <el-option label="启用" value="enabled" />
                <el-option label="停用" value="disabled" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :lg="8" :md="12" :sm="24" :xs="24">
            <el-form-item label="创建时间">
              <el-date-picker
                v-model="form.createdRange"
                type="datetimerange"
                value-format="YYYY-MM-DD HH:mm:ss"
                range-separator="至"
                start-placeholder="开始"
                end-placeholder="结束"
                class="ele-fluid"
              />
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label-width="0">
              <btn-items
                :wrap="false"
                :items="[
                  { preset: 'search', onClick: () => reload() },
                  { preset: 'reset', onClick: () => reset() }
                ]"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </ele-card>
    <ele-card :body-style="{ paddingTop: '8px' }" flex-table="auto">
      <ele-pro-table
        ref="tableRef"
        row-key="codeId"
        :columns="columns"
        :datasource="datasource"
        cache-key="adminPlatformCodeTable"
      >
        <template #code="{ row }">
          <code-cell :text="row.code" />
        </template>
        <template #action="{ row }">
          <el-button
            v-permission="'platform:code:status'"
            link
            :type="row.status === 'enabled' ? 'danger' : 'success'"
            @click="toggle(row)"
          >
            {{ row.status === 'enabled' ? '强制停用' : '恢复启用' }}
          </el-button>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref, onMounted } from 'vue';
  import CodeCell from '@/components/CodeCell.vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { pageMerchants } from '@/api/admin/merchant';
  import { pageAdminSites } from '@/api/admin/site';
  import {
    pagePlatformCodes,
    updatePlatformCodeStatus
  } from '@/api/admin/platform';

  defineOptions({ name: 'AdminPlatformCode' });

  const tableRef = ref(null);
  const merchants = ref([]);
  const sites = ref([]);
  const [form, resetFields] = useFormData({
    keyword: '',
    merchantId: undefined,
    siteId: undefined,
    status: '',
    createdRange: []
  });

  const where = () => ({
    keyword: form.keyword,
    merchantId: form.merchantId || undefined,
    siteId: form.siteId || undefined,
    status: form.status || undefined,
    beginTime: form.createdRange?.[0],
    endTime: form.createdRange?.[1]
  });

  const loadSites = () => {
    pageAdminSites({
      merchantId: form.merchantId || undefined,
      pageNum: 1,
      pageSize: 500
    })
      .then((res) => (sites.value = res.rows || []))
      .catch(() => {});
  };

  const onMerchantChange = () => {
    form.siteId = undefined;
    loadSites();
  };
  const columns = [
    { prop: 'code', label: '归因码', minWidth: 160, slot: 'code' },
    { prop: 'name', label: '名称', minWidth: 100, formatter: (row) => row.name || '-' },
    { prop: 'merchantName', label: '商户', minWidth: 100, formatter: (row) => row.merchantName || '-' },
    { prop: 'siteName', label: '站点', minWidth: 120, formatter: (row) => row.siteName || '-' },
    {
      prop: 'status',
      label: '状态',
      width: 90,
      formatter: (row) => (row.status === 'enabled' ? '启用' : '停用')
    },
    { prop: 'channel', label: '渠道', minWidth: 90, formatter: (row) => row.channel || '-' },
    { prop: 'createdAt', label: '创建时间', width: 160 },
    {
      columnKey: 'action',
      label: '操作',
      width: 100,
      slot: 'action',
      fixed: 'right'
    }
  ];

  const datasource = ({ page, limit, where: filters }) => {
    return pagePlatformCodes({
      ...(filters || {}),
      pageNum: page,
      pageSize: limit
    });
  };

  const reload = () => tableRef.value?.reload?.({ where: where(), page: 1 });

  const reset = () => {
    resetFields();
    loadSites();
    reload();
  };

  onMounted(() => {
    pageMerchants({ pageNum: 1, pageSize: 500 })
      .then((res) => (merchants.value = res.rows || []))
      .catch(() => {});
    loadSites();
  });

  const toggle = (row) => {
    const status = row.status === 'enabled' ? 'disabled' : 'enabled';
    const text = status === 'disabled' ? '强制停用' : '恢复启用';
    ElMessageBox.confirm(`${text}归因码「${row.code}」？`, '系统提示', {
      type: 'warning'
    })
      .then(async () => {
        const msg = await updatePlatformCodeStatus([row.codeId], status);
        EleMessage.success({ message: msg, plain: true });
        tableRef.value?.reload?.();
      })
      .catch(() => {});
  };
</script>
