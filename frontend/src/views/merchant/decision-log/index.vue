<template>
  <ele-page flex-table="auto">
    <summary-cards
      :items="summaryItems"
      :cols="5"
      clickable
      :active-key="activeContent"
      @select="handleSummarySelect"
    />
    <log-search
      ref="searchRef"
      v-model:site-id="siteId"
      :sites="sites"
      :show-site="isMerchant"
      @search="(where) => reload(where, 1)"
    />
    <ele-card
      class="table-section"
      :body-style="{ paddingTop: '8px' }"
      flex-table="auto"
    >
      <ele-pro-table
        ref="tableRef"
        row-key="requestId"
        :columns="columns"
        :datasource="datasource"
        :show-overflow-tooltip="true"
        :highlight-current-row="true"
        :cache-key="`${portal}DecisionLogTable`"
        :page-size="20"
      >
        <template #createdAt="{ row }">
          <el-link
            type="primary"
            underline="never"
            @click="handleDetail(row)"
            >{{ row.createdAt }}</el-link
          >
        </template>
        <template #path="{ row }">
          <span class="mono">{{ row.path }}</span>
        </template>
        <template #content="{ row }">
          <el-tag
            :type="getOptionType(contentOptions, row.content)"
            effect="light"
            size="small"
          >
            {{ getOptionLabel(contentOptions, row.content) }}
          </el-tag>
        </template>
        <template #touchType="{ row }">
          <el-tag
            :type="getOptionType(touchOptions, row.touchType)"
            effect="plain"
            size="small"
          >
            {{ getOptionLabel(touchOptions, row.touchType)
            }}{{
              row.handoffKind
                ? ` · ${row.handoffKind === 'token' ? '令牌' : '匹配'}`
                : ''
            }}
          </el-tag>
        </template>
        <template #channel="{ row }">
          <span v-if="row.channel" class="mono">{{ row.channel }}</span>
          <span v-else-if="row.errorReasons?.length" class="error-text">{{
            row.errorReasons.map((r) => errorReasons[r] || r).join('、')
          }}</span>
          <span v-else class="muted-text">-</span>
        </template>
        <template #client="{ row }">
          <el-tag
            v-if="row.inAppBrowser"
            type="warning"
            effect="plain"
            size="small"
            >{{ row.inAppBrowser }}</el-tag
          >
          <span
            class="muted-text"
            :style="row.inAppBrowser ? 'margin-left:6px' : ''"
            >{{ row.deviceFamily || '-'
            }}{{ row.country ? ` · ${row.country}` : '' }}</span
          >
        </template>
        <template #origin="{ row }">
          <span :class="{ 'error-text': row.fallback }">
            {{ originKindLabels[row.originKind] || row.originKind || '-'
            }}<template v-if="row.originStatus">
              {{ row.originStatus }}</template
            >
          </span>
        </template>
        <template #action="{ row }">
          <el-button link type="primary" @click="handleDetail(row)"
            >详情</el-button
          >
        </template>
      </ele-pro-table>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { ref, computed, watch, onMounted } from 'vue';
  import { useModal } from 'ele-admin-plus';
  import { APP_PORTAL } from '@portal';
  import {
    pageDecisionLogs,
    getDecisionLogStats
  } from '@/api/merchant/decision-log';
  import LogSearch from './components/log-search.vue';
  import {
    contentOptions,
    touchOptions,
    errorReasons,
    originKindLabels,
    getOptionLabel,
    getOptionType
  } from './constants';

  defineOptions({ name: 'DecisionLogPage' });

  const portal = APP_PORTAL;
  const isMerchant = portal === 'merchant';
  const { openModal } = useModal();
  const tableRef = ref(null);
  const searchRef = ref(null);
  const sites = ref([]);
  const siteId = ref(null);
  const activeContent = ref('');
  const stats = ref({
    total: 0,
    real: 0,
    review: 0,
    block: 0,
    first: 0,
    return: 0,
    handoff: 0,
    none: 0,
    monthUsage: null
  });

  if (isMerchant) {
    import('@/utils/use-site').then(({ useCurrentSite }) => {
      const cs = useCurrentSite();
      cs.load().then(() => {
        sites.value = cs.sites.value;
      });
    });
  }

  const summaryItems = computed(() => [
    {
      key: '',
      label: '24 小时判定',
      value: stats.value.total,
      desc:
        stats.value.monthUsage != null
          ? `按页面和 IP · 本月 ${stats.value.monthUsage}`
          : '按页面和 IP',
      type: 'primary'
    },
    {
      key: 'real',
      label: '真实内容',
      value: stats.value.real,
      desc: `同一页面、同一 IP 计 1 次 · 首访 ${stats.value.first} · 回访 ${stats.value.return} · 交接 ${stats.value.handoff}`,
      type: 'success'
    },
    {
      key: 'review',
      label: '审核内容',
      value: stats.value.review,
      desc: '按页面和 IP · 无归因或校验失败',
      type: 'warning'
    },
    {
      key: 'block',
      label: '拒绝 / 404',
      value: stats.value.block,
      desc: '路径规则或策略拦截',
      type: 'danger',
      alert: stats.value.block > 0
    },
    {
      key: 'bypass',
      label: '直通',
      value: stats.value.bypass || 0,
      desc: '静态资源不计入上面的页面统计',
      type: 'info'
    }
  ]);

  const columns = computed(() => {
    const cols = [
      {
        prop: 'createdAt',
        label: '时间',
        width: 190,
        slot: 'createdAt',
        formatter: (r) => r.createdAt
      }
    ];
    if (!isMerchant) {
      cols.push({ prop: 'merchantName', label: '商户', minWidth: 110 });
    }
    cols.push(
      { prop: 'siteName', label: '站点', minWidth: 110 },
      {
        prop: 'path',
        label: '路径',
        minWidth: 180,
        slot: 'path',
        formatter: (r) => r.path
      },
      {
        prop: 'content',
        label: '返回',
        width: 100,
        align: 'center',
        slot: 'content',
        formatter: (r) => getOptionLabel(contentOptions, r.content)
      },
      {
        prop: 'touchType',
        label: '触点',
        width: 110,
        align: 'center',
        slot: 'touchType',
        formatter: (r) => getOptionLabel(touchOptions, r.touchType)
      },
      {
        columnKey: 'channel',
        label: '渠道 / 失败原因',
        minWidth: 180,
        slot: 'channel'
      },
      { columnKey: 'client', label: '客户端', minWidth: 170, slot: 'client' },
      { columnKey: 'origin', label: '响应来源', width: 120, slot: 'origin' },
      {
        prop: 'latencyMs',
        label: '耗时',
        width: 80,
        align: 'right',
        formatter: (r) => `${r.latencyMs} ms`
      },
      {
        columnKey: 'action',
        label: '操作',
        width: 70,
        align: 'center',
        fixed: 'right',
        slot: 'action',
        hideInPrint: true,
        hideInExport: true
      }
    );
    return cols;
  });

  const datasource = ({ page, limit, where }) =>
    pageDecisionLogs({
      siteId: siteId.value || undefined,
      ...where,
      pageNum: page,
      pageSize: limit
    });

  const loadStats = () => {
    getDecisionLogStats(siteId.value || undefined)
      .then((d) => (stats.value = d))
      .catch(() => {});
  };

  const reload = (where, page) => {
    tableRef.value?.reload?.({
      where: where ?? searchRef.value?.where?.(),
      page
    });
    loadStats();
  };

  const handleSummarySelect = (item) => {
    activeContent.value = item.key;
    searchRef.value?.setContent(item.key);
  };

  const handleDetail = (row) =>
    openModal({
      custom: true,
      asyncComponent: () => import('./components/log-detail.vue'),
      componentProps: { requestId: row.requestId }
    });

  watch(siteId, () => reload(undefined, 1));
  onMounted(loadStats);
</script>
