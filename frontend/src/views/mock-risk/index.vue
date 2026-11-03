<template>
  <div class="mock-risk-page">
    <el-alert
      title="跨境支付风控系统前端 Mock 联调页面（基于 PRD）"
      type="success"
      :closable="false"
      show-icon
      class="mb16"
    />

    <el-row :gutter="12" class="mb16">
      <el-col
        v-for="item in overviewMetrics"
        :key="item.label"
        :xs="24"
        :sm="12"
        :md="6"
      >
        <el-card shadow="hover">
          <div class="metric-label">{{ item.label }}</div>
          <div class="metric-value">{{ item.value }}</div>
          <div class="metric-trend">{{ item.trend }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never">
      <el-tabs v-model="activeTab" type="border-card">
        <el-tab-pane label="渠道分佣" name="channel">
          <module-table
            title="渠道分佣策略列表"
            :data="channelCommissionRows"
            :columns="channelColumns"
          />
        </el-tab-pane>

        <el-tab-pane label="PayPal账号池治理" name="account">
          <module-table
            title="账号池治理列表"
            :data="accountPoolRows"
            :columns="accountColumns"
          />
        </el-tab-pane>

        <el-tab-pane label="商品池管理" name="product">
          <module-table
            title="商品池清单"
            :data="productPoolRows"
            :columns="productColumns"
          />
        </el-tab-pane>

        <el-tab-pane label="AB账单管理" name="abBill">
          <module-table
            title="AB账单任务列表"
            :data="abBillRows"
            :columns="abBillColumns"
          />
        </el-tab-pane>

        <el-tab-pane label="客诉中台" name="complaint">
          <module-table
            title="客诉工单列表"
            :data="complaintRows"
            :columns="complaintColumns"
          />
        </el-tab-pane>

        <el-tab-pane label="履约风控" name="fulfillment">
          <module-table
            title="履约异常工单"
            :data="fulfillmentRows"
            :columns="fulfillmentColumns"
          />
        </el-tab-pane>

        <el-tab-pane label="路由策略中心" name="routing">
          <module-table
            title="路由策略列表"
            :data="routeStrategyRows"
            :columns="routeStrategyColumns"
          />
        </el-tab-pane>

        <el-tab-pane label="商户管理/客户后台" name="merchant">
          <module-table
            title="商户风险治理列表"
            :data="merchantRows"
            :columns="merchantColumns"
          />
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script setup>
  import { computed, defineComponent, h, ref } from 'vue';
  import {
    overviewMetrics,
    channelCommissionRows,
    accountPoolRows,
    productPoolRows,
    abBillRows,
    complaintRows,
    fulfillmentRows,
    routeStrategyRows,
    merchantRows
  } from './mock-data';

  const activeTab = ref('channel');

  const statusTypeMap = {
    enabled: 'success',
    disabled: 'info',
    active: 'success',
    limited: 'warning',
    blocked: 'danger',
    healthy: 'success',
    warning: 'warning',
    fused: 'danger',
    on_sale: 'success',
    frozen: 'danger',
    consistent: 'success',
    inconsistent: 'danger',
    submitted: 'success',
    pending_material: 'warning',
    draft: 'info',
    processing: 'primary',
    pending_result: 'warning',
    closed: 'success',
    escalated: 'danger',
    new: 'info',
    approved: 'success',
    reviewing: 'warning',
    normal: 'success',
    observing: 'warning',
    locked: 'danger',
    rotating: 'warning'
  };

  const labelMap = {
    enabled: '启用',
    disabled: '停用',
    active: '正常',
    limited: '受限',
    blocked: '熔断',
    healthy: '健康',
    warning: '预警',
    fused: '已熔断',
    on_sale: '上架中',
    frozen: '冻结',
    consistent: '一致',
    inconsistent: '不一致',
    submitted: '已提交',
    pending_material: '待补充资料',
    draft: '草稿',
    processing: '处理中',
    pending_result: '待结果',
    closed: '已关闭',
    escalated: '已升级',
    new: '新建',
    approved: '已通过',
    reviewing: '审核中',
    normal: '正常',
    observing: '观察中',
    locked: '已锁定',
    rotating: '轮换中'
  };

  const asPercent = (value, digits = 2) => `${(value * 100).toFixed(digits)}%`;

  const channelColumns = computed(() => [
    { prop: 'channelCode', label: '渠道编码' },
    { prop: 'channelName', label: '渠道名称' },
    {
      prop: 'commissionRate',
      label: '分佣比例',
      formatter: ({ row }) => asPercent(row.commissionRate)
    },
    { prop: 'settleRule', label: '结算规则' },
    { prop: 'payoutDate', label: '发放日历' },
    {
      prop: 'lastSettleAmount',
      label: '最近结算金额',
      formatter: ({ row }) => `$${row.lastSettleAmount.toFixed(2)}`
    },
    { prop: 'status', label: '状态', tag: true }
  ]);

  const accountColumns = computed(() => [
    { prop: 'accountId', label: '账号ID' },
    { prop: 'scene', label: '治理场景' },
    { prop: 'boundMerchantCount', label: '绑定商户数' },
    {
      prop: 'successRate',
      label: '成功率',
      formatter: ({ row }) => asPercent(row.successRate)
    },
    {
      prop: 'complaintRate',
      label: '客诉率',
      formatter: ({ row }) => asPercent(row.complaintRate, 3)
    },
    { prop: 'riskStatus', label: '风险状态', tag: true },
    { prop: 'accountStatus', label: '账号状态', tag: true }
  ]);

  const productColumns = computed(() => [
    { prop: 'sku', label: 'SKU' },
    { prop: 'category', label: '分类' },
    { prop: 'displayTitle', label: '展示标题' },
    {
      prop: 'unitPrice',
      label: '单价',
      formatter: ({ row }) => `$${row.unitPrice.toFixed(2)}`
    },
    { prop: 'stock', label: '库存' },
    { prop: 'siteScope', label: '站点范围' },
    { prop: 'status', label: '状态', tag: true }
  ]);

  const abBillColumns = computed(() => [
    { prop: 'orderNo', label: '订单号' },
    { prop: 'merchantName', label: '商户' },
    {
      prop: 'aBillAmount',
      label: 'A账单金额',
      formatter: ({ row }) => `$${row.aBillAmount}`
    },
    {
      prop: 'bBillAmount',
      label: 'B账单金额',
      formatter: ({ row }) => `$${row.bBillAmount}`
    },
    { prop: 'itemDiff', label: '商品数量差异' },
    { prop: 'consistencyStatus', label: '金额一致性', tag: true },
    { prop: 'submitStatus', label: '提交状态', tag: true }
  ]);

  const complaintColumns = computed(() => [
    { prop: 'caseNo', label: '客诉单号' },
    { prop: 'merchantName', label: '商户' },
    { prop: 'disputeType', label: '争议类型' },
    { prop: 'amount', label: '金额', formatter: ({ row }) => `$${row.amount}` },
    { prop: 'riskLevel', label: '风险等级' },
    { prop: 'slaLeft', label: 'SLA剩余' },
    { prop: 'status', label: '状态', tag: true }
  ]);

  const fulfillmentColumns = computed(() => [
    { prop: 'caseNo', label: '风控单号' },
    { prop: 'orderNo', label: '订单号' },
    { prop: 'anomalyType', label: '异常类型' },
    { prop: 'anomalyLevel', label: '异常等级' },
    { prop: 'logisticsProvider', label: '物流商' },
    { prop: 'latestTrack', label: '最新轨迹' },
    { prop: 'status', label: '状态', tag: true }
  ]);

  const routeStrategyColumns = computed(() => [
    { prop: 'strategyCode', label: '策略编码' },
    { prop: 'strategyName', label: '策略名称' },
    { prop: 'strategyType', label: '策略类型' },
    { prop: 'priority', label: '优先级' },
    {
      prop: 'hitRate',
      label: '命中率',
      formatter: ({ row }) => asPercent(row.hitRate)
    },
    {
      prop: 'successRate',
      label: '策略成功率',
      formatter: ({ row }) => asPercent(row.successRate)
    },
    { prop: 'approvalStatus', label: '审批状态', tag: true }
  ]);

  const merchantColumns = computed(() => [
    { prop: 'merchantId', label: '商户ID' },
    { prop: 'merchantName', label: '商户名称' },
    { prop: 'siteCount', label: '站点数' },
    {
      prop: 'complaintRate',
      label: '客诉率',
      formatter: ({ row }) => asPercent(row.complaintRate, 3)
    },
    { prop: 'lockStatus', label: '风控状态', tag: true },
    { prop: 'keyStatus', label: '密钥状态', tag: true },
    { prop: 'latestRiskEvent', label: '最新风险事件' }
  ]);

  const ModuleTable = defineComponent({
    name: 'ModuleTable',
    props: {
      title: { type: String, required: true },
      data: { type: Array, required: true },
      columns: { type: Array, required: true }
    },
    setup(props) {
      return () =>
        h('div', { class: 'module-table' }, [
          h('div', { class: 'module-title' }, props.title),
          h(
            'div',
            { class: 'module-tools' },
            '已内置 mock 的多状态数据（正常/预警/熔断/待补充等），可直接联调前端交互'
          ),
          h(
            'el-table',
            {
              data: props.data,
              border: true,
              stripe: true,
              style: { width: '100%' }
            },
            () =>
              props.columns.map((col) =>
                h(
                  'el-table-column',
                  {
                    prop: col.prop,
                    label: col.label,
                    minWidth: col.minWidth || 120
                  },
                  col.tag
                    ? {
                        default: (scope) =>
                          h(
                            'el-tag',
                            {
                              type:
                                statusTypeMap[scope.row[col.prop]] || 'info',
                              effect: 'light'
                            },
                            () =>
                              labelMap[scope.row[col.prop]] ||
                              scope.row[col.prop]
                          )
                      }
                    : col.formatter
                      ? {
                          default: (scope) => col.formatter(scope)
                        }
                      : void 0
                )
              )
          )
        ]);
    }
  });
</script>

<style scoped>
  .mock-risk-page {
    padding: 16px;
  }

  .mb16 {
    margin-bottom: 16px;
  }

  .metric-label {
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .metric-value {
    margin-top: 8px;
    font-size: 24px;
    line-height: 1.1;
    font-weight: 600;
  }

  .metric-trend {
    margin-top: 8px;
    color: var(--el-color-primary);
    font-size: 12px;
  }

  .module-title {
    font-size: 15px;
    font-weight: 600;
    margin-bottom: 8px;
  }

  .module-tools {
    margin-bottom: 12px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }
</style>
