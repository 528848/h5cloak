<template>
  <ele-page>
    <ele-card class="page-section" :body-style="{ padding: '12px 18px' }">
      <div class="filter-bar">
        <el-space wrap>
          <el-select
            v-model="siteId"
            placeholder="全部站点"
            clearable
            style="width: 200px"
          >
            <el-option
              v-for="s in sites"
              :key="s.siteId"
              :label="s.name"
              :value="s.siteId"
            />
          </el-select>
          <el-radio-group v-model="rangeKey" size="default">
            <el-radio-button
              v-for="r in ranges"
              :key="r.value"
              :value="r.value"
              :label="r.value"
              >{{ r.label }}</el-radio-button
            >
          </el-radio-group>
          <el-date-picker
            v-if="rangeKey === 'custom'"
            v-model="customRange"
            type="daterange"
            value-format="YYYY-MM-DD"
            start-placeholder="开始"
            end-placeholder="结束"
            style="width: 260px"
          />
        </el-space>
        <span class="muted-text"
          >最后更新 {{ data.updatedAt || '-' }}
          <el-button link type="primary" :icon="Refresh" @click="load"
            >刷新</el-button
          ></span
        >
      </div>
    </ele-card>

    <el-alert
      v-if="quotaPct >= 80"
      :type="quotaPct >= 100 ? 'error' : 'warning'"
      show-icon
      :closable="false"
      class="page-section"
      :title="`本月判定额度已使用 ${quotaPct}%（${fmtNum(data.quota?.used)} / ${fmtNum(data.quota?.decisions)}），超出后按「${policyLabel}」处理`"
    />

    <summary-cards :items="kpis" :cols="6" />

    <el-row :gutter="16" class="page-section">
      <el-col :lg="16" :xs="24">
        <section-card
          title="放行与审核趋势"
          :subtitle="data.trendDaily ? '按日' : '按小时'"
          class="chart-card"
        >
          <template #extra>
            <el-radio-group v-model="trendMode" size="small">
              <el-radio-button value="content" label="content"
                >内容</el-radio-button
              >
              <el-radio-button value="touch" label="touch"
                >触点</el-radio-button
              >
            </el-radio-group>
          </template>
          <v-chart
            v-if="trend.length"
            ref="trendRef"
            :option="trendOption"
            autoresize
            style="height: 300px"
          />
          <el-empty v-else :image-size="72" description="该时段暂无访问" />
        </section-card>
      </el-col>
      <el-col :lg="8" :xs="24">
        <section-card title="渠道 TOP" subtitle="按放行次数" class="chart-card">
          <v-chart
            v-if="data.channels?.length"
            ref="channelRef"
            :option="rankBarOption(data.channels, channelLabel)"
            autoresize
            style="height: 300px"
          />
          <el-empty v-else :image-size="72" description="暂无放行" />
        </section-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="page-section">
      <el-col :lg="8" :xs="24">
        <section-card title="分销码 TOP" subtitle="放行次数" class="chart-card">
          <v-chart
            v-if="data.codes?.length"
            ref="codeRef"
            :option="rankBarOption(data.codes)"
            autoresize
            style="height: 260px"
          />
          <el-empty v-else :image-size="64" />
        </section-card>
      </el-col>
      <el-col :lg="8" :xs="24">
        <section-card
          title="来源置信度"
          subtitle="放行流量的来源可信度"
          class="chart-card"
        >
          <v-chart
            v-if="data.confidence?.length"
            ref="confRef"
            :option="donutOption(data.confidence, confidenceLabel)"
            autoresize
            style="height: 260px"
          />
          <el-empty v-else :image-size="64" />
        </section-card>
      </el-col>
      <el-col :lg="8" :xs="24">
        <section-card
          title="内置浏览器"
          subtitle="来自社媒 App 的访问"
          class="chart-card"
        >
          <v-chart
            v-if="inAppItems.length"
            ref="inAppRef"
            :option="donutOption(inAppItems, inAppLabel)"
            autoresize
            style="height: 260px"
          />
          <el-empty v-else :image-size="64" />
        </section-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="page-section">
      <el-col :lg="8" :xs="24">
        <section-card title="国家 / 地区" subtitle="按访问" class="chart-card">
          <v-chart
            v-if="countryItems.length"
            ref="countryRef"
            :option="rankBarOption(countryItems)"
            autoresize
            style="height: 240px"
          />
          <el-empty v-else :image-size="64" description="需配置 GeoIP" />
        </section-card>
      </el-col>
      <el-col :lg="8" :xs="24">
        <section-card
          title="校验失败原因"
          subtitle="伪造或失效参数"
          class="chart-card"
        >
          <v-chart
            v-if="data.validationReasons?.length"
            ref="reasonRef"
            :option="rankBarOption(data.validationReasons, reasonLabel)"
            autoresize
            style="height: 240px"
          />
          <el-empty v-else :image-size="64" description="没有校验失败" />
        </section-card>
      </el-col>
      <el-col :lg="8" :xs="24">
        <section-card title="接入方式" subtitle="按入口" class="chart-card">
          <v-chart
            v-if="data.entries?.length"
            ref="entryRef"
            :option="donutOption(data.entries, entryLabel)"
            autoresize
            style="height: 240px"
          />
          <el-empty v-else :image-size="64" />
        </section-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="page-section">
      <el-col :lg="12" :xs="24">
        <section-card title="健康" subtitle="域名、证书、源站">
          <template #extra>
            <el-link
              type="primary"
              underline="never"
              @click="go('/merchant/alert')"
              >告警 {{ data.alerts?.open || 0 }}</el-link
            >
          </template>
          <div v-if="healthRows.length" class="health-list">
            <div v-for="h in healthRows" :key="h.key" class="health-item">
              <span
                class="health-dot"
                :class="h.ok ? 'is-ok' : 'is-bad'"
              ></span>
              <span class="health-name mono">{{ h.name }}</span>
              <span class="health-kind muted-text">{{ h.kind }}</span>
              <span class="health-status" :class="h.ok ? '' : 'error-text'">{{
                h.status
              }}</span>
            </div>
          </div>
          <el-empty v-else :image-size="64" description="尚未配置域名或源站" />
        </section-card>
      </el-col>
      <el-col :lg="12" :xs="24">
        <section-card title="额度与转化" subtitle="本月">
          <div class="side-kv">
            <span>本月判定次数</span>
            <em
              >{{ fmtNum(data.quota?.used) }} /
              {{ fmtNum(data.quota?.decisions) }}（{{ quotaPct }}%）</em
            >
          </div>
          <el-progress
            :percentage="Math.min(quotaPct, 100)"
            :status="
              quotaPct >= 100 ? 'exception' : quotaPct >= 80 ? 'warning' : ''
            "
            :stroke-width="8"
            style="margin: 6px 0 14px"
          />
          <div class="side-kv">
            <span>超额策略</span><em>{{ policyLabel }}</em>
          </div>
          <div class="side-kv">
            <span>所选时段转化</span
            ><em
              >{{ fmtNum(summary.conversions) }} 笔 ·
              {{ fmtAmount(data.conversionAmount) }}</em
            >
          </div>
          <div class="side-kv">
            <span>放行转化率</span
            ><em>{{ pct(summary.conversions, summary.realTotal) }}%</em>
          </div>
          <div class="side-kv">
            <span>平均判定耗时</span
            ><em
              >{{
                summary.visits
                  ? Math.round(summary.latencySum / summary.visits)
                  : 0
              }}
              ms</em
            >
          </div>
          <div class="side-kv">
            <span>降级 / 回源错误</span
            ><em
              >{{ fmtNum(summary.degraded) }} /
              {{ fmtNum(summary.originErrors) }}</em
            >
          </div>
        </section-card>
      </el-col>
    </el-row>
  </ele-page>
</template>

<script setup>
  import { ref, computed, watch, onMounted } from 'vue';
  import { useRouter } from 'vue-router';
  import { Refresh } from '@element-plus/icons-vue';
  import VChart from 'vue-echarts';
  import { EleMessage } from 'ele-admin-plus';
  import { useEcharts } from '@/utils/use-echarts';
  import { useCurrentSite } from '@/utils/use-site';
  import { getDashboard } from '@/api/merchant/ops';
  import {
    lineOption,
    rankBarOption,
    donutOption,
    pct,
    fmtNum,
    fmtAmount
  } from '@/utils/charts';
  import {
    confidenceOptions,
    errorReasons,
    getOptionLabel
  } from '@/views/merchant/decision-log/constants';
  import { inAppOptions } from '@/views/merchant/decision-log/constants';

  defineOptions({ name: 'MerchantDashboard' });

  const router = useRouter();
  const { sites, siteId: currentSite, load: loadSites } = useCurrentSite();
  const siteId = ref(null);
  const rangeKey = ref('today');
  const customRange = ref([]);
  const trendMode = ref('content');
  const data = ref({});
  const loading = ref(false);
  const ranges = [
    { value: 'today', label: '今日' },
    { value: 'yesterday', label: '昨日' },
    { value: '7d', label: '近 7 日' },
    { value: '30d', label: '近 30 日' },
    { value: 'custom', label: '自定义' }
  ];
  const trendRef = ref(null);
  const channelRef = ref(null);
  const codeRef = ref(null);
  const confRef = ref(null);
  const inAppRef = ref(null);
  const countryRef = ref(null);
  const reasonRef = ref(null);
  const entryRef = ref(null);
  useEcharts([
    trendRef,
    channelRef,
    codeRef,
    confRef,
    inAppRef,
    countryRef,
    reasonRef,
    entryRef
  ]);

  const summary = computed(() => data.value.summary || {});
  const trend = computed(() => data.value.trend || []);
  const quotaPct = computed(() =>
    pct(data.value.quota?.used || 0, data.value.quota?.decisions || 0)
  );
  const policyLabel = computed(
    () =>
      ({ alert: '只告警', review_all: '全部返回审核内容', deny: '拒绝服务' })[
        data.value.quota?.policy
      ] || '只告警'
  );

  const kpis = computed(() => {
    const s = summary.value;
    const base = (s.visits || 0) - (s.bypass || 0);
    return [
      {
        key: 'visits',
        label: '访问',
        value: s.visits,
        desc: `放行率 ${pct(s.realTotal, base)}%`
      },
      {
        key: 'real',
        label: '放行',
        value: s.realTotal,
        type: 'success',
        desc: `首访 ${fmtNum(s.firstReal)} · 回访 ${fmtNum(s.returnReal)} · 交接 ${fmtNum(s.handoffReal)}`
      },
      {
        key: 'review',
        label: '审核返回',
        value: s.review,
        type: 'warning',
        desc: `占比 ${pct(s.review, base)}%`
      },
      {
        key: 'block',
        label: '拦截 / 404',
        value: (s.block || 0) + (s.notFound || 0),
        type: 'danger',
        desc: `拦截 ${fmtNum(s.block)}`
      },
      {
        key: 'failed',
        label: '校验失败',
        value: s.validationFailed,
        type: 'info',
        desc: `内置浏览器 ${fmtNum(s.inApp)}`
      },
      {
        key: 'conv',
        label: '转化',
        value: s.conversions,
        type: 'success',
        desc: fmtAmount(data.value.conversionAmount)
      }
    ];
  });

  const trendOption = computed(() =>
    trendMode.value === 'content'
      ? lineOption(trend.value, [
          { name: '放行', key: 'real' },
          { name: '审核', key: 'review' },
          { name: '拦截', key: 'block' }
        ])
      : lineOption(trend.value, [
          { name: '首访', key: 'first' },
          { name: '回访', key: 'return' },
          { name: '交接', key: 'handoff' }
        ])
  );

  const channelLabel = (k) => (k === '' ? '（无归因）' : k);
  const confidenceLabel = (k) =>
    getOptionLabel(confidenceOptions, k) || k || '-';
  const inAppLabel = (k) =>
    k === ''
      ? '系统浏览器'
      : inAppOptions.find((o) => o.value === k)?.label || k;
  const reasonLabel = (k) => errorReasons[k] || k;
  const entryLabel = (k) =>
    ({
      hosted: '托管',
      api_v2: '服务端 v2',
      api_v1: '服务端 V1',
      js: 'JS SDK'
    })[k] || k;
  const inAppItems = computed(() => data.value.inApp || []);
  const countryItems = computed(() =>
    (data.value.countries || []).map((i) => ({ k: i.k || '未知', v: i.v }))
  );

  const healthRows = computed(() => {
    const h = data.value.health || {};
    const rows = [];
    (h.domains || []).forEach((d) =>
      rows.push({
        key: 'd' + d.domain,
        name: d.domain,
        kind: d.status === 'bound' ? '域名' : '域名 · 未绑定',
        ok: d.status !== 'bound' || d.healthStatus !== 'unhealthy',
        status:
          d.status !== 'bound'
            ? '待校验'
            : d.healthStatus === 'unhealthy'
              ? '解析异常'
              : d.certExpiresAt
                ? `证书 ${String(d.certExpiresAt).slice(0, 10)} 到期`
                : '正常'
      })
    );
    (h.origins || []).forEach((o) =>
      rows.push({
        key: 'o' + o.siteId + o.kind,
        name: o.siteName,
        kind: o.kind === 'review' ? '审核源站' : '真实源站',
        ok: o.healthStatus !== 'unhealthy',
        status:
          o.healthStatus === 'unhealthy'
            ? `不可达：${o.message || ''}`
            : o.healthStatus === 'healthy'
              ? '正常'
              : '未检查'
      })
    );
    return rows;
  });

  const params = computed(() => {
    const p = { range: rangeKey.value };
    if (siteId.value) p.siteId = siteId.value;
    if (rangeKey.value === 'custom' && customRange.value?.length === 2) {
      p.from = customRange.value[0];
      p.to = customRange.value[1];
    }
    return p;
  });

  const load = () => {
    if (rangeKey.value === 'custom' && customRange.value?.length !== 2) return;
    loading.value = true;
    getDashboard(params.value)
      .then((d) => (data.value = d || {}))
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => (loading.value = false));
  };
  const go = (path) => router.push(path);

  watch(params, load);
  onMounted(() => {
    loadSites().then(() => {
      siteId.value = currentSite.value || null;
      load();
    });
  });
</script>

<style scoped>
  .filter-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
  }

  .chart-card :deep(.el-card__body) {
    padding-top: 6px;
  }

  .health-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .health-item {
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 13px;
  }

  .health-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex: none;
  }

  .health-dot.is-ok {
    background: var(--el-color-success);
  }

  .health-dot.is-bad {
    background: var(--el-color-danger);
    box-shadow: 0 0 0 3px var(--el-color-danger-light-8);
  }

  .health-name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .health-kind {
    flex: none;
    font-size: 12px;
  }

  .health-status {
    flex: none;
    max-width: 45%;
    font-size: 12px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .muted-text {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .error-text {
    color: var(--el-color-danger);
  }
</style>
