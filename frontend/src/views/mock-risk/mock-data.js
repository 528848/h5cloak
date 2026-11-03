export const overviewMetrics = [
  { label: '今日交易额(USD)', value: '1,286,430', trend: '+8.7%' },
  { label: '支付成功率', value: '96.82%', trend: '+0.5%' },
  { label: '在途客诉工单', value: '128', trend: '-12' },
  { label: '账号熔断数', value: '7', trend: '+2' }
];

export const channelCommissionRows = [
  {
    channelCode: 'CH-SEA-001',
    channelName: '东南亚代理联盟',
    commissionRate: 0.12,
    settleMode: 't_plus_n',
    settleRule: 'T+7',
    payoutDate: '每周二 10:00',
    status: 'enabled',
    lastSettleAmount: 12430.66
  },
  {
    channelCode: 'CH-US-009',
    channelName: 'US KOL Network',
    commissionRate: 0.08,
    settleMode: 'weekly',
    settleRule: '每周结算',
    payoutDate: '每周五 16:00',
    status: 'enabled',
    lastSettleAmount: 38499.12
  },
  {
    channelCode: 'CH-EU-015',
    channelName: '欧洲站长联盟',
    commissionRate: 0.1,
    settleMode: 'monthly',
    settleRule: '月结',
    payoutDate: '每月5号 12:00',
    status: 'disabled',
    lastSettleAmount: 8950.24
  }
];

export const accountPoolRows = [
  {
    accountId: 'PP-US-10032',
    scene: '100USD以内 + 3C',
    boundMerchantCount: 21,
    successRate: 0.978,
    complaintRate: 0.007,
    riskStatus: 'healthy',
    accountStatus: 'active'
  },
  {
    accountId: 'PP-US-10177',
    scene: '服饰品类 + 品牌A站点',
    boundMerchantCount: 16,
    successRate: 0.942,
    complaintRate: 0.021,
    riskStatus: 'warning',
    accountStatus: 'limited'
  },
  {
    accountId: 'PP-EU-20443',
    scene: '100-500USD + 混合品类',
    boundMerchantCount: 37,
    successRate: 0.901,
    complaintRate: 0.034,
    riskStatus: 'fused',
    accountStatus: 'blocked'
  }
];

export const productPoolRows = [
  {
    sku: 'SKU-3C-001',
    category: '3C',
    displayTitle: 'Wireless Earbuds Pro',
    unitPrice: 39.99,
    stock: 1230,
    siteScope: 'brand-a.com',
    status: 'on_sale'
  },
  {
    sku: 'SKU-FA-043',
    category: '服装',
    displayTitle: 'Summer Casual Shirt',
    unitPrice: 26.5,
    stock: 780,
    siteScope: 'fashion-b.com',
    status: 'on_sale'
  },
  {
    sku: 'SKU-HM-116',
    category: '家居',
    displayTitle: 'Smart Bedside Lamp',
    unitPrice: 58.0,
    stock: 0,
    siteScope: 'home-c.com',
    status: 'frozen'
  }
];

export const abBillRows = [
  {
    orderNo: 'A202604250001',
    merchantName: '深圳某跨境商户',
    aBillAmount: 129.5,
    bBillAmount: 129.5,
    itemDiff: 'A:2 / B:4',
    consistencyStatus: 'consistent',
    submitStatus: 'submitted'
  },
  {
    orderNo: 'A202604250029',
    merchantName: '杭州服饰商户',
    aBillAmount: 67.0,
    bBillAmount: 67.0,
    itemDiff: 'A:1 / B:3',
    consistencyStatus: 'consistent',
    submitStatus: 'pending_material'
  },
  {
    orderNo: 'A202604250103',
    merchantName: '广州家居商户',
    aBillAmount: 318.3,
    bBillAmount: 315.3,
    itemDiff: 'A:5 / B:6',
    consistencyStatus: 'inconsistent',
    submitStatus: 'draft'
  }
];

export const complaintRows = [
  {
    caseNo: 'C-20260425-88',
    merchantName: '深圳某跨境商户',
    disputeType: '货不对板',
    amount: 129.5,
    riskLevel: 'high',
    status: 'processing',
    slaLeft: '01:52:11'
  },
  {
    caseNo: 'C-20260425-91',
    merchantName: '杭州服饰商户',
    disputeType: '未收到货',
    amount: 67.0,
    riskLevel: 'middle',
    status: 'pending_result',
    slaLeft: '06:13:43'
  },
  {
    caseNo: 'C-20260424-15',
    merchantName: '宁波3C商户',
    disputeType: '重复扣款',
    amount: 21.99,
    riskLevel: 'low',
    status: 'closed',
    slaLeft: '--'
  }
];

export const fulfillmentRows = [
  {
    caseNo: 'F-20260425-002',
    orderNo: 'ORD-10032901',
    anomalyType: '轨迹停滞',
    anomalyLevel: 'middle',
    logisticsProvider: 'YunExpress',
    latestTrack: '2026-04-22 08:23 纽约站点处理中',
    status: 'processing'
  },
  {
    caseNo: 'F-20260425-007',
    orderNo: 'ORD-10044120',
    anomalyType: '地理瞬移',
    anomalyLevel: 'high',
    logisticsProvider: '4PX',
    latestTrack: '5分钟内 CN->US 跳转',
    status: 'escalated'
  },
  {
    caseNo: 'F-20260425-009',
    orderNo: 'ORD-10055179',
    anomalyType: '超时未发货',
    anomalyLevel: 'high',
    logisticsProvider: '待分配',
    latestTrack: '下单后 53 小时仍未揽件',
    status: 'new'
  }
];

export const routeStrategyRows = [
  {
    strategyCode: 'RSK-001',
    strategyName: '高风险商户优先稳定通道',
    strategyType: 'risk_first',
    priority: 100,
    hitRate: 0.34,
    successRate: 0.972,
    approvalStatus: 'approved'
  },
  {
    strategyCode: 'CST-006',
    strategyName: '低客诉账号优先路由',
    strategyType: 'success_first',
    priority: 90,
    hitRate: 0.29,
    successRate: 0.965,
    approvalStatus: 'approved'
  },
  {
    strategyCode: 'FEE-011',
    strategyName: '欧元订单成本最优',
    strategyType: 'cost_first',
    priority: 80,
    hitRate: 0.18,
    successRate: 0.931,
    approvalStatus: 'reviewing'
  }
];

export const merchantRows = [
  {
    merchantId: 'M-10021',
    merchantName: '深圳某跨境商户',
    siteCount: 4,
    complaintRate: 0.018,
    lockStatus: 'normal',
    keyStatus: 'healthy',
    latestRiskEvent: '无'
  },
  {
    merchantId: 'M-10098',
    merchantName: '杭州服饰商户',
    siteCount: 7,
    complaintRate: 0.043,
    lockStatus: 'observing',
    keyStatus: 'rotating',
    latestRiskEvent: '客诉率超过阈值，进入观察'
  },
  {
    merchantId: 'M-10231',
    merchantName: '广州家居商户',
    siteCount: 2,
    complaintRate: 0.079,
    lockStatus: 'locked',
    keyStatus: 'healthy',
    latestRiskEvent: '自动熔断，等待人工复核'
  }
];
