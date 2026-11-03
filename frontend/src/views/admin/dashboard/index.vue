<template>
  <ele-page>
    <el-row :gutter="16">
      <el-col v-for="item in metrics" :key="item.label" :lg="6" :md="12" :xs="24">
        <ele-card :body-style="{ padding: '18px 20px' }">
          <div class="metric-label">{{ item.label }}</div>
          <div class="metric-value">{{ item.value }}</div>
        </ele-card>
      </el-col>
    </el-row>
    <ele-card header="平台运营概览" style="margin-top: 16px">
      <el-empty description="后续接入访问趋势、命中趋势和 API 调用趋势" />
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { onMounted, ref } from 'vue';
  import { getAdminDashboard } from '@/api/admin/dashboard';

  defineOptions({ name: 'AdminDashboard' });

  const metrics = ref([
    { label: '今日访问量', value: 0 },
    { label: '今日黑页命中', value: 0 },
    { label: '启用商户', value: 0 },
    { label: '异常域名', value: 0 }
  ]);

  const loadData = async () => {
    const data = await getAdminDashboard();
    metrics.value = [
      { label: '今日访问量', value: data.todayVisits ?? 0 },
      { label: '今日黑页命中', value: data.todayBlackHits ?? 0 },
      { label: '启用商户', value: data.enabledMerchantCount ?? 0 },
      { label: '异常域名', value: data.abnormalDomainCount ?? 0 }
    ];
  };

  onMounted(() => {
    loadData();
  });
</script>

<style scoped>
  .metric-label {
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .metric-value {
    margin-top: 8px;
    color: var(--el-text-color-primary);
    font-size: 28px;
    font-weight: 600;
  }
</style>
