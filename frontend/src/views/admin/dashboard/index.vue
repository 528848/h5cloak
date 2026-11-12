<template>
  <ele-page>
    <ele-card class="page-section" :body-style="{ padding: '12px 18px' }">
      <div class="filter-bar">
        <el-radio-group v-model="rangeKey">
          <el-radio-button
            v-for="r in ranges"
            :key="r.value"
            :value="r.value"
            :label="r.value"
            >{{ r.label }}</el-radio-button
          >
        </el-radio-group>
        <span class="muted-text">
          最后更新 {{ data.updatedAt || '-' }}
          <el-button
            link
            type="primary"
            :icon="Refresh"
            :loading="loading"
            @click="refresh"
            >刷新</el-button
          >
        </span>
      </div>
    </ele-card>

    <summary-cards :items="kpis" :cols="6" />

    <el-row :gutter="16" class="page-section">
      <el-col :lg="16" :xs="24">
        <section-card
          title="全平台流量"
          :subtitle="(data.trend?.length || 0) > 48 ? '按日' : '按小时'"
        >
          <v-chart
            v-if="data.trend?.length"
            :key="data.updatedAt"
            ref="trendRef"
            :option="
              lineOption(data.trend, [
                { name: '放行', key: 'real' },
                { name: '审核', key: 'review' },
                { name: '拦截', key: 'block' }
              ])
            "
            autoresize
            style="height: 300px"
          />
          <el-empty v-else :image-size="72" description="该时段暂无访问" />
        </section-card>
      </el-col>
      <el-col :lg="8" :xs="24">
        <section-card
          title="网关节点"
          subtitle="心跳 15 秒，60 秒无心跳视为离线"
        >
          <div v-if="data.nodes?.length" class="node-list">
            <div v-for="n in data.nodes" :key="n.node" class="node-item">
              <div class="node-head">
                <span class="node-dot"></span>
                <strong class="mono">{{ n.node }}</strong>
                <el-tag size="small" effect="plain">{{ n.version }}</el-tag>
              </div>
              <div class="node-grid">
                <div
                  ><span>QPS</span
                  ><em>{{ Number(n.qps || 0).toFixed(1) }}</em></div
                >
                <div
                  ><span>P99</span
                  ><em>{{ Number(n.p99Ms || 0).toFixed(1) }} ms</em></div
                >
                <div>
                  <span>回源错误率</span
                  ><em :class="{ 'error-text': n.originErrorRate > 0.05 }"
                    >{{
                      (Number(n.originErrorRate || 0) * 100).toFixed(1)
                    }}%</em
                  >
                </div>
                <div
                  ><span>站点快照</span><em>{{ n.sites }}</em></div
                >
                <div
                  ><span>累计请求</span><em>{{ fmtNum(n.requests) }}</em></div
                >
                <div
                  ><span>启动</span><em>{{ n.startedAt }}</em></div
                >
              </div>
            </div>
          </div>
          <el-empty v-else :image-size="64" description="没有在线网关节点" />
        </section-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="page-section">
      <el-col :lg="8" :xs="24">
        <section-card title="平台规模" subtitle="实体数量与状态">
          <div class="side-kv"
            ><span>商户</span
            ><em>{{ c.merchants_enabled }} 启用 / {{ c.merchants }}</em></div
          >
          <div class="side-kv"
            ><span>站点</span
            ><em>{{ c.sites_enabled }} 启用 / {{ c.sites }}</em></div
          >
          <div class="side-kv"
            ><span>域名</span
            ><em>{{ c.domains_bound }} 已绑定 / {{ c.domains }}</em></div
          >
          <div class="side-kv"
            ><span>归因码</span><em>{{ fmtNum(c.codes) }}</em></div
          >
          <div class="side-kv"
            ><span>域名异常</span
            ><em :class="{ 'error-text': c.domains_unhealthy }">{{
              c.domains_unhealthy || 0
            }}</em></div
          >
          <div class="side-kv"
            ><span>源站不可达</span
            ><em :class="{ 'error-text': c.origins_unhealthy }">{{
              c.origins_unhealthy || 0
            }}</em></div
          >
          <div class="side-kv"
            ><span>14 天内到期证书</span
            ><em :class="{ 'error-text': c.certs_expiring }">{{
              c.certs_expiring || 0
            }}</em></div
          >
        </section-card>
      </el-col>
      <el-col :lg="8" :xs="24">
        <section-card title="接入方式" subtitle="按入口">
          <v-chart
            v-if="data.entries?.length"
            :key="data.updatedAt"
            ref="entryRef"
            :option="donutOption(data.entries, entryLabel)"
            autoresize
            style="height: 240px"
          />
          <el-empty v-else :image-size="64" />
        </section-card>
      </el-col>
      <el-col :lg="8" :xs="24">
        <section-card title="不可达源站" subtitle="全平台">
          <template #extra>
            <el-link type="primary" underline="never" @click="openAlerts"
              >未恢复告警 {{ data.alerts?.open || 0 }}</el-link
            >
          </template>
          <div v-if="data.unhealthyOrigins?.length" class="origin-list">
            <div
              v-for="o in data.unhealthyOrigins"
              :key="o.id"
              class="origin-item"
            >
              <div
                ><strong>{{ o.siteName }}</strong>
                <span class="muted-text"
                  >· 商户 {{ o.merchantId }} ·
                  {{ o.kind === 'review' ? '审核' : '真实' }}</span
                ></div
              >
              <div class="mono muted-text">{{ o.url }}</div>
              <div class="error-text">{{ o.message }}</div>
            </div>
          </div>
          <el-empty v-else :image-size="64" description="全部源站可达" />
        </section-card>
      </el-col>
    </el-row>

    <ele-card class="table-section" :body-style="{ paddingTop: '8px' }">
      <ele-pro-table
        ref="alertTable"
        row-key="id"
        :columns="alertColumns"
        :datasource="alertDatasource"
        :cache-key="'adminDashboardAlerts'"
        :page-size="10"
      >
        <template #toolbar>
          <span class="muted-text">全平台未恢复告警</span>
        </template>
        <template #severity="{ row }">
          <el-tag
            :type="
              { info: 'info', warning: 'warning', critical: 'danger' }[
                row.severity
              ] || 'info'
            "
            size="small"
            effect="dark"
            >{{ row.severity }}</el-tag
          >
        </template>
        <template #title="{ row }">
          <div>{{ row.title }}</div>
          <div class="muted-text">{{ row.message }}</div>
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref, computed, watch, onMounted } from 'vue';
  import { Refresh } from '@element-plus/icons-vue';
  import VChart from 'vue-echarts';
  import { EleMessage } from 'ele-admin-plus';
  import { useEcharts } from '@/utils/use-echarts';
  import { getDashboard, pageAlerts } from '@/api/merchant/ops';
  import {
    lineOption,
    donutOption,
    pct,
    fmtNum,
    fmtAmount
  } from '@/utils/charts';

  defineOptions({ name: 'AdminDashboard' });

  const rangeKey = ref('today');
  const data = ref({});
  const loading = ref(false);
  const trendRef = ref(null);
  const entryRef = ref(null);
  const alertTable = ref(null);
  useEcharts([trendRef, entryRef]);
  const ranges = [
    { value: 'today', label: '今日' },
    { value: 'yesterday', label: '昨日' },
    { value: '7d', label: '近 7 日' },
    { value: '30d', label: '近 30 日' }
  ];
  const c = computed(() => data.value.counts || {});
  const kpis = computed(() => {
    const s = data.value.summary || {};
    const base = (s.visits || 0) - (s.bypass || 0);
    return [
      {
        key: 'visits',
        label: '访问',
        value: s.visits,
        desc: `按页面和 IP · 放行率 ${pct(s.realTotal, base)}%`
      },
      {
        key: 'real',
        label: '放行',
        value: s.realTotal,
        type: 'success',
        desc: `同一页面、同一 IP 计 1 次 · 首访 ${fmtNum(s.firstReal)} · 回访 ${fmtNum(s.returnReal)}`
      },
      {
        key: 'review',
        label: '审核返回',
        value: s.review,
        type: 'warning',
        desc: `同一页面、同一 IP 计 1 次 · 占比 ${pct(s.review, base)}%`
      },
      { key: 'block', label: '拦截', value: s.block, type: 'danger' },
      {
        key: 'degraded',
        label: '降级判定',
        value: s.degraded,
        type: 'info',
        alert: (s.degraded || 0) > 0,
        desc: `回源错误 ${fmtNum(s.originErrors)}`
      },
      {
        key: 'alerts',
        label: '未恢复告警',
        value: data.value.alerts?.open || 0,
        type: 'danger',
        alert: (data.value.alerts?.open || 0) > 0,
        desc: `严重 ${data.value.alerts?.['open:critical'] || 0}`
      }
    ];
  });
  const entryLabel = (k) =>
    ({
      hosted: '托管',
      api_v2: '服务端 v2',
      api_v1: '服务端 V1',
      js: 'JS SDK'
    })[k] || k;
  const alertColumns = [
    { prop: 'severity', label: '级别', slot: 'severity', width: 90 },
    { prop: 'merchantId', label: '商户', width: 80 },
    { prop: 'title', label: '告警', slot: 'title', minWidth: 320 },
    { prop: 'siteName', label: '站点', width: 140 },
    { prop: 'firedAt', label: '触发', width: 160 },
    { prop: 'lastSeenAt', label: '最近', width: 160 }
  ];
  const alertDatasource = ({ page, limit }) =>
    pageAlerts({ status: 'open', pageNum: page, pageSize: limit });

  const refresh = () => {
    if (loading.value) {
      return;
    }
    loading.value = true;
    getDashboard({ range: rangeKey.value, _t: Date.now() })
      .then((d) => {
        data.value = d || {};
        alertTable.value?.reload?.({ page: 1 });
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => {
        loading.value = false;
      });
  };
  const openAlerts = () =>
    alertTable.value?.$el?.scrollIntoView?.({ behavior: 'smooth' });
  void fmtAmount;

  watch(rangeKey, refresh);
  onMounted(refresh);
</script>

<style scoped>
  .filter-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
  }

  .node-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .node-item {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 10px;
    padding: 10px 12px;
  }

  .node-head {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
  }

  .node-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--el-color-success);
    box-shadow: 0 0 0 3px var(--el-color-success-light-8);
  }

  .node-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 6px 10px;
    font-size: 12px;
  }

  .node-grid div {
    display: flex;
    flex-direction: column;
  }

  .node-grid span {
    color: var(--el-text-color-secondary);
  }

  .node-grid em {
    font-style: normal;
    font-weight: 600;
    font-size: 13px;
  }

  .origin-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
    font-size: 13px;
  }

  .origin-item {
    padding: 8px 10px;
    border-radius: 8px;
    background: var(--el-color-danger-light-9);
  }

  .muted-text {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .error-text {
    color: var(--el-color-danger);
  }
</style>
