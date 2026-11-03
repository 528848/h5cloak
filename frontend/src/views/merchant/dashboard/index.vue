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
    <ele-card header="商户访问趋势" style="margin-top: 16px">
      <el-empty :image-size="80" />
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { onMounted, ref } from 'vue';
  import { getMerchantDashboard } from '@/api/merchant/dashboard';

  defineOptions({ name: 'MerchantDashboard' });

  const metrics = ref([
    { label: '今日访问量', value: 0 },
    { label: '今日白页命中', value: 0 },
    { label: '今日黑页命中', value: 0 },
    { label: 'API 调用量', value: 0 }
  ]);

  const loadData = async () => {
    const data = await getMerchantDashboard();
    metrics.value = [
      { label: '今日访问量', value: data.todayVisits ?? 0 },
      { label: '今日白页命中', value: data.todayWhiteHits ?? 0 },
      { label: '今日黑页命中', value: data.todayBlackHits ?? 0 },
      { label: 'API 调用量', value: data.todayApiCalls ?? 0 }
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
