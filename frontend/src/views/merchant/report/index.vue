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
          <el-radio-group v-model="rangeKey">
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
        <el-space>
          <el-tooltip content="聚合表与原始日志对账" placement="top">
            <el-button :icon="Checked" @click="doReconcile">对账</el-button>
          </el-tooltip>
          <el-button
            v-permission="'report:export'"
            type="primary"
            plain
            :icon="Download"
            @click="doExport"
            >导出当前报表</el-button
          >
        </el-space>
      </div>
    </ele-card>

    <el-alert
      v-if="reconcile"
      :type="reconcile.match ? 'success' : 'warning'"
      show-icon
      closable
      class="page-section"
      :title="
        reconcile.match
          ? `对账一致：聚合 ${reconcile.aggregated.visits} 次访问 / ${reconcile.aggregated.real} 次放行，与原始日志相同`
          : `对账差异：聚合 ${reconcile.aggregated.visits}/${reconcile.aggregated.real}，日志 ${reconcile.raw.visits}/${reconcile.raw.real}，聚合每分钟更新，差异应在一分钟内消失`
      "
      @close="reconcile = null"
    />

    <ele-card class="table-section" :body-style="{ paddingTop: '4px' }">
      <ele-tabs v-model="tab" :items="tabs" class="report-tabs" />
      <p class="muted-text tip">{{ tipOf(tab) }}</p>

      <!-- 渠道 -->
      <ele-pro-table
        v-if="tab === 'channel'"
        ref="channelTable"
        row-key="channel"
        :columns="channelColumns"
        :datasource="channelRows"
        :pagination="false"
        :loading="loading"
        :cache-key="'merchantReportChannel'"
      >
        <template #channel="{ row }">
          <span class="mono">{{ row.channel || '（无归因）' }}</span>
        </template>
        <template #rate="{ row }">
          {{ pct(row.conversions, row.visits) }}%
        </template>
        <template #conversionAmount="{ row }">{{
          fmtAmount(row.conversionAmount)
        }}</template>
      </ele-pro-table>

      <!-- 归因码（分销码 / 活动码） -->
      <ele-pro-table
        v-else-if="tab === 'code' || tab === 'campaign'"
        :key="tab"
        ref="codeTable"
        row-key="codeId"
        :columns="codeColumns"
        :datasource="codeDatasource"
        :cache-key="`merchantReport_${tab}`"
        :page-size="20"
      >
        <template #code="{ row }">
          <span class="mono">{{ row.code }}</span>
          <span v-if="row.name" class="muted-text" style="margin-left: 6px">{{
            row.name
          }}</span>
        </template>
        <template #status="{ row }">
          <el-tag
            :type="row.status === 'enabled' ? 'success' : 'info'"
            size="small"
            effect="plain"
            >{{ row.status === 'enabled' ? '启用' : '停用' }}</el-tag
          >
        </template>
        <template #rate="{ row }"
          >{{ pct(row.conversions, row.visits) }}%</template
        >
        <template #conversionAmount="{ row }">{{
          fmtAmount(row.conversionAmount)
        }}</template>
      </ele-pro-table>

      <!-- 路径 -->
      <ele-pro-table
        v-else-if="tab === 'path'"
        ref="pathTable"
        row-key="path"
        :columns="pathColumns"
        :datasource="pathDatasource"
        :cache-key="'merchantReportPath'"
        :page-size="20"
      >
        <template #path="{ row }">
          <span class="mono">{{ row.path }}</span>
        </template>
        <template #noAttrRate="{ row }"
          >{{ pct(row.noAttribution, row.visits) }}%</template
        >
      </ele-pro-table>

      <!-- 转化 -->
      <ele-pro-table
        v-else-if="tab === 'conversion'"
        ref="convTable"
        row-key="id"
        :columns="convColumns"
        :datasource="convDatasource"
        :cache-key="'merchantReportConversion'"
        :page-size="20"
      >
        <template #orderNo="{ row }">
          <span class="mono">{{ row.orderNo }}</span>
        </template>
        <template #amount="{ row }"
          >{{ fmtAmount(row.amount) }} {{ row.currency }}</template
        >
        <template #channel="{ row }">
          <span class="mono">{{ row.channel || '-' }}</span>
          <span v-if="row.code" class="muted-text"> · {{ row.code }}</span>
        </template>
        <template #occurredAt="{ row }">{{ fmtTime(row.occurredAt) }}</template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref, computed, watch, onMounted, nextTick } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { Download, Checked } from '@element-plus/icons-vue';
  import { ElNotification } from 'element-plus';
  import { EleMessage } from 'ele-admin-plus';
  import { useCurrentSite } from '@/utils/use-site';
  import {
    getChannelReport,
    pageCodeReport,
    pagePathReport,
    pageConversionReport,
    getReconcile,
    createExport
  } from '@/api/merchant/ops';
  import { pct, fmtAmount } from '@/utils/charts';

  defineOptions({ name: 'MerchantReport' });

  const route = useRoute();
  const router = useRouter();
  const { sites, siteId: currentSite, load: loadSites } = useCurrentSite();
  const siteId = ref(null);
  const rangeKey = ref('7d');
  const customRange = ref([]);
  const tab = ref('channel');
  const loading = ref(false);
  const reconcile = ref(null);
  const channelRows = ref([]);
  const codeTable = ref(null);
  const pathTable = ref(null);
  const convTable = ref(null);

  const ranges = [
    { value: 'today', label: '今日' },
    { value: 'yesterday', label: '昨日' },
    { value: '7d', label: '近 7 日' },
    { value: '30d', label: '近 30 日' },
    { value: 'custom', label: '自定义' }
  ];
  const tabs = [
    { name: 'channel', label: '渠道' },
    { name: 'code', label: '归因码' },
    { name: 'campaign', label: '活动码' },
    { name: 'path', label: '路径' },
    { name: 'conversion', label: '转化' }
  ];
  const tipOf = (t) =>
    ({
      channel: '按归因渠道汇总放行与转化，实时统计。',
      code: '按归因码汇总放行、回访、交接与转化，每分钟更新。',
      campaign: '按活动码汇总，每分钟更新。',
      path: '按网站页面路径汇总。页面里的图片、脚本、样式和其他文件请求不计入。',
      conversion: '服务端回传的转化明细及其归因。'
    })[t];

  const rangeParams = computed(() => {
    const p = { range: rangeKey.value };
    if (siteId.value) p.siteId = siteId.value;
    if (rangeKey.value === 'custom' && customRange.value?.length === 2) {
      p.from = customRange.value[0];
      p.to = customRange.value[1];
    }
    return p;
  });
  const ready = computed(
    () => rangeKey.value !== 'custom' || customRange.value?.length === 2
  );

  const channelColumns = [
    { prop: 'channel', label: '渠道', slot: 'channel', minWidth: 180 },
    {
      prop: 'visits',
      label: '放行',
      width: 100,
      align: 'right',
      sortable: true
    },
    { prop: 'firstVisits', label: '首访', width: 90, align: 'right' },
    { prop: 'returnVisits', label: '回访', width: 90, align: 'right' },
    { prop: 'handoffVisits', label: '交接', width: 90, align: 'right' },
    { prop: 'inApp', label: '内置浏览器', width: 110, align: 'right' },
    { prop: 'visitors', label: '访客', width: 90, align: 'right' },
    { prop: 'conversions', label: '转化', width: 90, align: 'right' },
    { prop: 'rate', label: '转化率', slot: 'rate', width: 90, align: 'right' },
    {
      prop: 'conversionAmount',
      label: '金额',
      slot: 'conversionAmount',
      width: 120,
      align: 'right'
    }
  ];
  const codeColumns = [
    { prop: 'code', label: '码', slot: 'code', minWidth: 200 },
    { prop: 'channel', label: '渠道标签', width: 120 },
    { prop: 'status', label: '状态', slot: 'status', width: 80 },
    { prop: 'visits', label: '放行', width: 100, align: 'right' },
    { prop: 'firstVisits', label: '首访', width: 90, align: 'right' },
    { prop: 'returnVisits', label: '回访', width: 90, align: 'right' },
    { prop: 'handoffVisits', label: '交接', width: 90, align: 'right' },
    { prop: 'inApp', label: '内置浏览器', width: 110, align: 'right' },
    { prop: 'conversions', label: '转化', width: 90, align: 'right' },
    { prop: 'rate', label: '转化率', slot: 'rate', width: 90, align: 'right' },
    {
      prop: 'conversionAmount',
      label: '金额',
      slot: 'conversionAmount',
      width: 120,
      align: 'right'
    }
  ];
  const pathColumns = [
    { prop: 'path', label: '路径', slot: 'path', minWidth: 240 },
    { prop: 'siteName', label: '站点', width: 140 },
    { prop: 'visits', label: '访问', width: 100, align: 'right' },
    { prop: 'realTotal', label: '放行', width: 100, align: 'right' },
    { prop: 'review', label: '审核', width: 100, align: 'right' },
    { prop: 'block', label: '拦截', width: 90, align: 'right' },
    { prop: 'noAttribution', label: '无归因访问', width: 110, align: 'right' },
    {
      prop: 'noAttrRate',
      label: '无归因占比',
      slot: 'noAttrRate',
      width: 110,
      align: 'right'
    }
  ];
  const convColumns = [
    { prop: 'occurredAt', label: '时间', slot: 'occurredAt', width: 170 },
    { prop: 'orderNo', label: '订单号', slot: 'orderNo', minWidth: 160 },
    { prop: 'siteName', label: '站点', width: 140 },
    { prop: 'status', label: '状态', width: 90 },
    {
      prop: 'amount',
      label: '金额',
      slot: 'amount',
      width: 130,
      align: 'right'
    },
    { prop: 'channel', label: '归因渠道 · 码', slot: 'channel', minWidth: 200 },
    {
      prop: 'visitorId',
      label: '访客',
      minWidth: 200,
      showOverflowTooltip: true
    }
  ];

  const loadChannel = () => {
    if (!ready.value) return;
    loading.value = true;
    getChannelReport(rangeParams.value)
      .then((rows) => (channelRows.value = rows || []))
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => (loading.value = false));
  };
  const ds =
    (fn, extra = {}) =>
    ({ page, limit }) =>
      fn({ ...rangeParams.value, ...extra, pageNum: page, pageSize: limit });
  const codeDatasource = computed(() =>
    ds(pageCodeReport, {
      codeType: tab.value === 'campaign' ? 'campaign' : 'distributor'
    })
  );
  const pathDatasource = ds(pagePathReport);
  const convDatasource = ds(pageConversionReport);

  const reloadAll = () => {
    if (!ready.value) return;
    if (tab.value === 'channel') loadChannel();
    nextTick(() => {
      codeTable.value?.reload?.({ page: 1 });
      pathTable.value?.reload?.({ page: 1 });
      convTable.value?.reload?.({ page: 1 });
    });
  };

  const doReconcile = () =>
    getReconcile(rangeParams.value)
      .then((d) => (reconcile.value = d))
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));

  const exportKind = () =>
    ({
      channel: 'report_channel',
      code: 'report_code',
      campaign: 'report_campaign',
      path: 'report_path',
      conversion: 'report_conversion'
    })[tab.value];
  const doExport = () => {
    const p = rangeParams.value;
    createExport({
      kind: exportKind(),
      siteId: p.siteId || 0,
      rangeKey: p.range,
      from: p.from,
      to: p.to
    })
      .then(() => {
        ElNotification({
          title: '已加入下载中心',
          message: '文件生成后可在下载中心获取',
          type: 'success',
          duration: 4000,
          onClick: () => router.push('/merchant/download')
        });
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));
  };
  const fmtTime = (v) => (v ? String(v).replace('T', ' ').slice(0, 19) : '-');

  watch(rangeParams, reloadAll);
  watch(tab, (t) => {
    if (t === 'channel') loadChannel();
  });
  onMounted(() => {
    if (route.query.tab === 'export') {
      router.replace('/merchant/download');
      return;
    }
    loadSites().then(() => {
      siteId.value = currentSite.value || null;
      reloadAll();
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

  .report-tabs {
    margin-bottom: 4px;
  }

  .tip {
    margin: 0 0 10px;
    font-size: 12px;
  }

  .muted-text {
    color: var(--el-text-color-secondary);
  }

  .error-text {
    color: var(--el-color-danger);
    font-size: 12px;
  }
</style>
