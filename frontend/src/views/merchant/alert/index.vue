<template>
  <ele-page>
    <summary-cards
      :items="kpis"
      :cols="4"
      clickable
      :active-key="activeKey"
      @select="selectKpi"
    />

    <ele-card class="table-section" :body-style="{ paddingTop: '4px' }">
      <ele-tabs v-model="tab" :items="tabs" />

      <!-- 告警列表 -->
      <template v-if="tab === 'alerts'">
        <div class="toolbar">
          <el-space wrap>
            <el-select
              v-model="filter.status"
              placeholder="状态"
              clearable
              style="width: 120px"
              @change="reload"
            >
              <el-option value="open" label="未恢复" />
              <el-option value="resolved" label="已恢复" />
            </el-select>
            <el-select
              v-model="filter.type"
              placeholder="类型"
              clearable
              style="width: 180px"
              @change="reload"
            >
              <el-option
                v-for="d in defaults"
                :key="d.type"
                :value="d.type"
                :label="d.name"
              />
            </el-select>
            <el-select
              v-model="filter.severity"
              placeholder="级别"
              clearable
              style="width: 110px"
              @change="reload"
            >
              <el-option
                v-for="s in severities"
                :key="s.value"
                :value="s.value"
                :label="s.label"
              />
            </el-select>
          </el-space>
          <el-button link type="primary" :icon="Refresh" @click="reload"
            >刷新</el-button
          >
        </div>
        <ele-pro-table
          ref="tableRef"
          row-key="id"
          :columns="columns"
          :datasource="datasource"
          :toolbar="false"
          :cache-key="'merchantAlertTable'"
          :page-size="20"
          :highlight-current-row="true"
        >
          <template #severity="{ row }">
            <el-tag :type="sevType(row.severity)" size="small" effect="dark">{{
              sevLabel(row.severity)
            }}</el-tag>
          </template>
          <template #title="{ row }">
            <div class="alert-title">{{ row.title }}</div>
            <div class="muted-text alert-msg">{{ row.message }}</div>
          </template>
          <template #status="{ row }">
            <el-tag
              :type="row.status === 'open' ? 'danger' : 'success'"
              size="small"
              effect="light"
              >{{ row.status === 'open' ? '未恢复' : '已恢复' }}</el-tag
            >
          </template>
          <template #notified="{ row }">
            <el-space :size="4" wrap>
              <el-tag
                v-for="(v, k) in row.notified || {}"
                :key="k"
                size="small"
                :type="
                  v === 'ok'
                    ? 'success'
                    : v.startsWith('skipped')
                      ? 'info'
                      : 'danger'
                "
                effect="plain"
              >
                {{ channelLabel(k) }}
              </el-tag>
            </el-space>
          </template>
          <template #action="{ row }">
            <el-button link type="primary" @click="showDetail(row)"
              >详情</el-button
            >
            <el-button
              v-if="row.status === 'open'"
              v-permission="'alert:edit'"
              link
              type="warning"
              @click="resolve(row)"
              >关闭</el-button
            >
          </template>
        </ele-pro-table>
      </template>

      <!-- 规则 -->
      <template v-else-if="tab === 'rules'">
        <p class="muted-text tip"
          >每类告警一条默认规则；修改后只对本商户生效，可随时恢复默认。流量类告警按滚动窗口评估，恢复类（源站
          / 域名 / 证书）在问题消失后自动关闭。</p
        >
        <div class="rule-list">
          <div v-for="r in rules" :key="r.type + r.siteId" class="rule-item">
            <div class="rule-head">
              <el-switch
                v-model="r.enabled"
                :disabled="!canEdit"
                @change="save(r)"
              />
              <div class="rule-name">
                <strong>{{ r.name }}</strong>
                <el-tag
                  v-if="r.custom"
                  size="small"
                  type="primary"
                  effect="plain"
                  >已自定义</el-tag
                >
                <el-tag
                  size="small"
                  :type="sevType(r.severity)"
                  effect="light"
                  >{{ sevLabel(r.severity) }}</el-tag
                >
              </div>
              <span class="muted-text rule-desc">{{
                defaultOf(r.type)?.desc
              }}</span>
              <el-space>
                <el-button
                  v-permission="'alert:edit'"
                  link
                  type="primary"
                  @click="edit(r)"
                  >编辑</el-button
                >
                <el-button
                  v-if="r.custom"
                  v-permission="'alert:edit'"
                  link
                  @click="reset(r)"
                  >恢复默认</el-button
                >
              </el-space>
            </div>
            <div class="rule-meta muted-text">
              <span v-if="r.threshold.pct">阈值 {{ r.threshold.pct }}%</span>
              <span v-if="r.threshold.count"
                >阈值 {{ r.threshold.count }} 次</span
              >
              <span v-if="r.threshold.days"
                >提前 {{ r.threshold.days }} 天</span
              >
              <span v-if="r.threshold.minVisits"
                >最少 {{ r.threshold.minVisits }} 次样本</span
              >
              <span>窗口 {{ r.windowMinutes }} 分钟</span>
              <span>冷却 {{ r.cooldownMinutes }} 分钟</span>
              <span
                >通知：{{
                  (r.channels || []).map(channelLabel).join('、') || '无'
                }}</span
              >
            </div>
          </div>
        </div>
      </template>

      <!-- 通知设置 -->
      <template v-else>
        <el-row :gutter="24">
          <el-col :lg="12" :xs="24">
            <el-form label-position="top" style="max-width: 520px">
              <el-form-item label="接收邮箱（每行一个，最多 10 个）">
                <el-input
                  v-model="settings.emailsText"
                  type="textarea"
                  :rows="4"
                  placeholder="ops@example.com"
                />
                <div class="muted-text form-tip">
                  平台 SMTP：<el-tag
                    size="small"
                    :type="settings.emailConfigured ? 'success' : 'info'"
                    effect="plain"
                    >{{
                      settings.emailConfigured
                        ? '已配置'
                        : '未配置，邮件渠道会跳过'
                    }}</el-tag
                  >
                  <el-button
                    link
                    type="primary"
                    size="small"
                    :disabled="!settings.emailConfigured"
                    @click="test('email')"
                    >发送测试</el-button
                  >
                </div>
              </el-form-item>
              <el-form-item label="Telegram Chat ID">
                <el-input
                  v-model="settings.telegramChatId"
                  placeholder="与平台 Bot 对话后获取的 chat_id"
                />
                <div class="muted-text form-tip">
                  平台 Bot：<el-tag
                    size="small"
                    :type="settings.telegramConfigured ? 'success' : 'info'"
                    effect="plain"
                    >{{
                      settings.telegramConfigured
                        ? '已配置'
                        : '未配置，Telegram 渠道会跳过'
                    }}</el-tag
                  >
                  <el-button
                    link
                    type="primary"
                    size="small"
                    :disabled="!settings.telegramConfigured"
                    @click="test('telegram')"
                    >发送测试</el-button
                  >
                </div>
              </el-form-item>
              <el-form-item label="启用的通知渠道">
                <el-checkbox-group v-model="settings.channels">
                  <el-checkbox value="inapp" label="inapp">站内</el-checkbox>
                  <el-checkbox value="email" label="email">邮件</el-checkbox>
                  <el-checkbox value="telegram" label="telegram"
                    >Telegram</el-checkbox
                  >
                  <el-checkbox value="webhook" label="webhook"
                    >Webhook</el-checkbox
                  >
                </el-checkbox-group>
                <div class="muted-text form-tip"
                  >这里关闭的渠道对所有规则生效；Webhook
                  目标在「Webhook」页配置。</div
                >
              </el-form-item>
              <el-form-item>
                <el-button
                  v-permission="'alert:edit'"
                  type="primary"
                  :loading="saving"
                  @click="saveSettings"
                  >保存</el-button
                >
                <el-button @click="test('inapp')">发一条站内测试通知</el-button>
              </el-form-item>
            </el-form>
          </el-col>
          <el-col :lg="12" :xs="24">
            <section-card title="最近站内通知" subtitle="右上角铃铛同步显示">
              <template #extra>
                <el-button link type="primary" @click="markAllRead"
                  >全部已读</el-button
                >
              </template>
              <div v-if="notifications.length" class="notif-list">
                <div
                  v-for="n in notifications"
                  :key="n.id"
                  class="notif-item"
                  :class="{ 'is-unread': !n.readAt }"
                >
                  <el-tag
                    size="small"
                    :type="sevType(n.severity)"
                    effect="light"
                    >{{ kindLabel(n.kind) }}</el-tag
                  >
                  <div class="notif-body">
                    <div>{{ n.title }}</div>
                    <div class="muted-text">{{ n.content }}</div>
                  </div>
                  <span class="muted-text notif-time">{{
                    fmtTime(n.createdAt)
                  }}</span>
                </div>
              </div>
              <el-empty v-else :image-size="64" description="暂无通知" />
            </section-card>
          </el-col>
        </el-row>
      </template>
    </ele-card>

    <!-- 规则编辑 -->
    <ele-modal
      v-model="editVisible"
      :title="`编辑规则：${editing?.name || ''}`"
      width="520px"
      @closed="editing = null"
    >
      <el-form v-if="editing" label-width="110px">
        <el-form-item label="级别">
          <el-radio-group v-model="editing.severity">
            <el-radio-button
              v-for="s in severities"
              :key="s.value"
              :value="s.value"
              :label="s.value"
              >{{ s.label }}</el-radio-button
            >
          </el-radio-group>
        </el-form-item>
        <el-form-item
          v-if="fieldsOf(editing.type).includes('pct')"
          label="百分比阈值"
        >
          <el-input-number
            v-model="editing.threshold.pct"
            :min="1"
            :max="100"
          />
          <span class="muted-text" style="margin-left: 8px">%</span>
        </el-form-item>
        <el-form-item
          v-if="fieldsOf(editing.type).includes('count')"
          label="次数阈值"
        >
          <el-input-number
            v-model="editing.threshold.count"
            :min="1"
            :max="1000000"
          />
        </el-form-item>
        <el-form-item
          v-if="fieldsOf(editing.type).includes('days')"
          label="提前天数"
        >
          <el-input-number
            v-model="editing.threshold.days"
            :min="1"
            :max="90"
          />
        </el-form-item>
        <el-form-item
          v-if="fieldsOf(editing.type).includes('minVisits')"
          label="最少样本"
        >
          <el-input-number
            v-model="editing.threshold.minVisits"
            :min="1"
            :max="100000"
          />
          <div class="muted-text form-tip"
            >窗口内访问少于该值不评估，避免小流量误报</div
          >
        </el-form-item>
        <el-form-item label="评估窗口">
          <el-input-number
            v-model="editing.windowMinutes"
            :min="5"
            :max="1440"
          />
          <span class="muted-text" style="margin-left: 8px">分钟</span>
        </el-form-item>
        <el-form-item label="冷却时间">
          <el-input-number
            v-model="editing.cooldownMinutes"
            :min="0"
            :max="10080"
          />
          <span class="muted-text" style="margin-left: 8px"
            >分钟内同一目标不重复告警</span
          >
        </el-form-item>
        <el-form-item label="通知渠道">
          <el-checkbox-group v-model="editing.channels">
            <el-checkbox value="inapp" label="inapp">站内</el-checkbox>
            <el-checkbox value="email" label="email">邮件</el-checkbox>
            <el-checkbox value="telegram" label="telegram"
              >Telegram</el-checkbox
            >
            <el-checkbox value="webhook" label="webhook">Webhook</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save(editing, true)"
          >保存</el-button
        >
      </template>
    </ele-modal>

    <!-- 告警详情 -->
    <ele-modal v-model="detailVisible" title="告警详情" width="600px">
      <template v-if="detail">
        <div class="side-kv"
          ><span>类型</span><em>{{ detail.typeName }}</em></div
        >
        <div class="side-kv"
          ><span>站点</span><em>{{ detail.siteName || '全部' }}</em></div
        >
        <div class="side-kv"
          ><span>目标</span><em class="mono">{{ detail.targetKey }}</em></div
        >
        <div class="side-kv"
          ><span>首次触发</span><em>{{ detail.firedAt }}</em></div
        >
        <div class="side-kv"
          ><span>最近出现</span><em>{{ detail.lastSeenAt }}</em></div
        >
        <div v-if="detail.resolvedAt" class="side-kv"
          ><span>恢复时间</span><em>{{ detail.resolvedAt }}</em></div
        >
        <div class="side-kv"
          ><span>说明</span><em>{{ detail.message }}</em></div
        >
        <pre class="code-block mono">{{
          JSON.stringify(detail.detail || {}, null, 2)
        }}</pre>
        <div class="side-kv"
          ><span>通知结果</span
          ><em>{{
            Object.entries(detail.notified || {})
              .map(([k, v]) => `${channelLabel(k)}：${v}`)
              .join('；') || '-'
          }}</em></div
        >
      </template>
    </ele-modal>
  </ele-page>
</template>

<script setup>
  import { ref, computed, onMounted, watch } from 'vue';
  import { useRoute } from 'vue-router';
  import { Refresh } from '@element-plus/icons-vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage } from 'ele-admin-plus';
  import { usePermission } from '@/utils/use-permission';
  import {
    getAlertRules,
    saveAlertRule,
    deleteAlertRule,
    pageAlerts,
    getAlertStats,
    resolveAlert,
    getNotifySettings,
    saveNotifySettings,
    testNotifyChannel,
    pageNotifications,
    markNotificationsRead
  } from '@/api/merchant/ops';

  defineOptions({ name: 'MerchantAlert' });

  const route = useRoute();
  const { hasPermission } = usePermission();
  const canEdit = hasPermission('alert:edit');
  const tab = ref('alerts');
  const tabs = [
    { name: 'alerts', label: '告警' },
    { name: 'rules', label: '规则' },
    { name: 'settings', label: '通知设置' }
  ];
  const severities = [
    { value: 'info', label: '提示' },
    { value: 'warning', label: '警告' },
    { value: 'critical', label: '严重' }
  ];
  const tableRef = ref(null);
  const stats = ref({});
  const filter = ref({ status: 'open', type: '', severity: '' });
  const activeKey = ref('open');
  const rules = ref([]);
  const defaults = ref([]);
  const editing = ref(null);
  const editVisible = ref(false);
  const detail = ref(null);
  const detailVisible = ref(false);
  const saving = ref(false);
  const settings = ref({
    emailsText: '',
    telegramChatId: '',
    channels: [],
    emailConfigured: false,
    telegramConfigured: false
  });
  const notifications = ref([]);

  const kpis = computed(() => [
    {
      key: 'open',
      label: '未恢复',
      value: stats.value.open || 0,
      type: 'danger',
      alert: (stats.value.open || 0) > 0
    },
    {
      key: 'critical',
      label: '严重',
      value: stats.value['open:critical'] || 0,
      type: 'danger'
    },
    {
      key: 'warning',
      label: '警告',
      value: stats.value['open:warning'] || 0,
      type: 'warning'
    },
    {
      key: 'resolved',
      label: '已恢复',
      value: stats.value.resolved || 0,
      type: 'success'
    }
  ]);
  const selectKpi = (item) => {
    activeKey.value = item.key;
    tab.value = 'alerts';
    if (item.key === 'open')
      filter.value = { status: 'open', type: '', severity: '' };
    else if (item.key === 'resolved')
      filter.value = { status: 'resolved', type: '', severity: '' };
    else filter.value = { status: 'open', type: '', severity: item.key };
    reload();
  };

  const columns = [
    { prop: 'severity', label: '级别', slot: 'severity', width: 80 },
    { prop: 'title', label: '告警', slot: 'title', minWidth: 320 },
    { prop: 'siteName', label: '站点', width: 140 },
    { prop: 'status', label: '状态', slot: 'status', width: 90 },
    { prop: 'firedAt', label: '触发', width: 160 },
    { prop: 'lastSeenAt', label: '最近', width: 160 },
    { prop: 'notified', label: '通知', slot: 'notified', minWidth: 200 },
    {
      prop: 'action',
      label: '操作',
      slot: 'action',
      width: 120,
      align: 'center'
    }
  ];
  const datasource = ({ page, limit }) =>
    pageAlerts({ ...filter.value, pageNum: page, pageSize: limit });
  const reload = () => {
    tableRef.value?.reload?.({ page: 1 });
    getAlertStats()
      .then((s) => (stats.value = s || {}))
      .catch(() => {});
  };

  const sevLabel = (s) => severities.find((x) => x.value === s)?.label || s;
  const sevType = (s) =>
    ({ info: 'info', warning: 'warning', critical: 'danger' })[s] || 'info';
  const channelLabel = (k) =>
    ({
      inapp: '站内',
      email: '邮件',
      telegram: 'Telegram',
      webhook: 'Webhook'
    })[String(k).replace(':resolved', '')] +
    (String(k).endsWith(':resolved') ? '（恢复）' : '');
  const kindLabel = (k) =>
    ({ alert: '告警', alert_resolved: '恢复', export: '导出', system: '系统' })[
      k
    ] || k;
  const fmtTime = (v) => (v ? String(v).replace('T', ' ').slice(0, 19) : '');
  const defaultOf = (t) => defaults.value.find((d) => d.type === t);
  const fieldsOf = (t) => defaultOf(t)?.fields || [];

  const loadRules = () =>
    getAlertRules()
      .then((d) => {
        rules.value = (d.rules || []).map((r) => ({
          ...r,
          threshold: { ...(r.threshold || {}) },
          channels: [...(r.channels || [])]
        }));
        defaults.value = d.defaults || [];
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));
  const edit = (r) => {
    editing.value = JSON.parse(JSON.stringify(r));
    editVisible.value = true;
  };
  const save = (r, closeModal = false) => {
    saving.value = true;
    saveAlertRule({
      siteId: r.siteId || 0,
      type: r.type,
      enabled: r.enabled,
      severity: r.severity,
      threshold: r.threshold,
      windowMinutes: r.windowMinutes,
      cooldownMinutes: r.cooldownMinutes,
      channels: r.channels
    })
      .then(() => {
        EleMessage.success({ message: '已保存', plain: true });
        if (closeModal) editVisible.value = false;
        loadRules();
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => (saving.value = false));
  };
  const reset = (r) =>
    deleteAlertRule(r.id)
      .then(() => {
        EleMessage.success({ message: '已恢复默认', plain: true });
        loadRules();
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));

  const resolve = (row) =>
    ElMessageBox.confirm(
      '确定手动关闭该告警？若问题仍存在且过了冷却时间会再次触发。',
      '关闭告警',
      { type: 'warning', draggable: true }
    )
      .then(() => resolveAlert(row.id))
      .then(() => {
        EleMessage.success({ message: '已关闭', plain: true });
        reload();
      })
      .catch(() => {});
  const showDetail = (row) => {
    detail.value = row;
    detailVisible.value = true;
  };

  const loadSettings = () =>
    getNotifySettings()
      .then((d) => {
        settings.value = {
          ...d,
          emailsText: (d.emails || []).join('\n'),
          channels: d.channels || []
        };
      })
      .catch(() => {});
  const saveSettings = () => {
    saving.value = true;
    saveNotifySettings({
      emails: settings.value.emailsText
        .split(/\r?\n|,/)
        .map((s) => s.trim())
        .filter(Boolean),
      telegramChatId: settings.value.telegramChatId,
      channels: settings.value.channels
    })
      .then(() => EleMessage.success({ message: '已保存', plain: true }))
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => (saving.value = false));
  };
  const test = (ch) =>
    testNotifyChannel(ch)
      .then(() => {
        EleMessage.success({ message: '已发送', plain: true });
        loadNotifications();
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));
  const loadNotifications = () =>
    pageNotifications({ pageNum: 1, pageSize: 10 })
      .then((r) => (notifications.value = r.rows || []))
      .catch(() => {});
  const markAllRead = () =>
    markNotificationsRead([]).then(() => {
      loadNotifications();
      window.dispatchEvent(new CustomEvent('cloak:notifications-changed'));
    });

  watch(tab, (t) => {
    if (t === 'rules') loadRules();
    if (t === 'settings') {
      loadSettings();
      loadNotifications();
    }
  });
  onMounted(() => {
    getAlertStats()
      .then((s) => (stats.value = s || {}))
      .catch(() => {});
    loadRules();
    if (route.query.id) {
      filter.value = { status: '', type: '', severity: '' };
      activeKey.value = '';
    }
  });
</script>

<style scoped>
  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
    margin: 4px 0 10px;
  }

  .tip {
    margin: 4px 0 14px;
    font-size: 12px;
  }

  .alert-title {
    font-weight: 500;
  }

  .alert-msg {
    font-size: 12px;
    white-space: normal;
    line-height: 1.5;
  }

  .rule-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .rule-item {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 10px;
    padding: 12px 14px;
  }

  .rule-head {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }

  .rule-name {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 200px;
  }

  .rule-desc {
    flex: 1;
    min-width: 200px;
    font-size: 12px;
  }

  .rule-meta {
    display: flex;
    gap: 16px;
    flex-wrap: wrap;
    margin-top: 8px;
    font-size: 12px;
  }

  .form-tip {
    margin-top: 6px;
    font-size: 12px;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .notif-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .notif-item {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    font-size: 13px;
    padding: 8px 10px;
    border-radius: 8px;
  }

  .notif-item.is-unread {
    background: var(--el-fill-color-light);
  }

  .notif-body {
    flex: 1;
    min-width: 0;
  }

  .notif-time {
    flex: none;
    font-size: 12px;
  }

  .code-block {
    margin: 8px 0;
    padding: 10px 12px;
    border-radius: 8px;
    background: var(--el-fill-color-light);
    font-size: 12px;
    max-height: 220px;
    overflow: auto;
  }

  .muted-text {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
</style>
