<template>
  <ele-page>
    <el-row :gutter="16">
      <el-col :lg="9" :xs="24">
        <section-card
          title="Webhook 目标"
          subtitle="告警与导出事件推送到你的服务"
          class="page-section"
        >
          <template #extra>
            <el-button
              v-permission="'webhook:edit'"
              type="primary"
              size="small"
              :icon="Plus"
              @click="openEdit()"
              >新增</el-button
            >
          </template>
          <div v-if="endpoints.length" class="ep-list">
            <div
              v-for="e in endpoints"
              :key="e.id"
              class="ep-item"
              :class="{ 'is-active': e.id === currentId }"
              @click="currentId = e.id"
            >
              <div class="ep-head">
                <strong>{{ e.name }}</strong>
                <el-tag
                  v-if="!e.enabled"
                  size="small"
                  type="info"
                  effect="plain"
                  >停用</el-tag
                >
                <el-tag
                  v-else-if="e.failureCount >= 3"
                  size="small"
                  type="danger"
                  effect="plain"
                  >连续失败 {{ e.failureCount }}</el-tag
                >
                <el-tag
                  v-else-if="e.lastStatus"
                  size="small"
                  :type="e.lastStatus < 300 ? 'success' : 'danger'"
                  effect="plain"
                  >{{ e.lastStatus }}</el-tag
                >
              </div>
              <div class="mono ep-url">{{ e.url }}</div>
              <div class="muted-text">
                {{
                  e.events?.length ? `${e.events.length} 个事件` : '全部事件'
                }}
                · 密钥 {{ e.secretHint || '-' }}
                <template v-if="e.lastDeliveredAt">
                  · 最近 {{ e.lastDeliveredAt }}</template
                >
              </div>
              <el-space class="ep-actions" :size="2">
                <el-button
                  v-permission="'webhook:edit'"
                  link
                  type="primary"
                  size="small"
                  @click.stop="sendTest(e)"
                  >测试</el-button
                >
                <el-button
                  v-permission="'webhook:edit'"
                  link
                  type="primary"
                  size="small"
                  @click.stop="openEdit(e)"
                  >编辑</el-button
                >
                <el-button
                  v-permission="'webhook:edit'"
                  link
                  size="small"
                  @click.stop="rotate(e)"
                  >重置密钥</el-button
                >
                <el-button
                  v-permission="'webhook:edit'"
                  link
                  type="danger"
                  size="small"
                  @click.stop="remove(e)"
                  >删除</el-button
                >
              </el-space>
            </div>
          </div>
          <el-empty v-else :image-size="72" description="还没有 Webhook" />
        </section-card>

        <section-card
          title="签名校验"
          subtitle="每次请求都带签名头"
          class="page-section"
        >
          <div class="side-kv"><span>X-Cloak-Event</span><em>事件名</em></div>
          <div class="side-kv"
            ><span>X-Cloak-Event-Id</span
            ><em>事件 ID（重试时不变，用于幂等）</em></div
          >
          <div class="side-kv"
            ><span>X-Cloak-Delivery</span><em>投递 ID</em></div
          >
          <div class="side-kv"
            ><span>X-Cloak-Timestamp</span><em>Unix 秒</em></div
          >
          <div class="side-kv"
            ><span>X-Cloak-Signature</span
            ><em class="mono"
              >v1=hex(HMAC-SHA256(secret, ts + "." + body))</em
            ></div
          >
          <div class="side-kv"
            ><span>重试</span
            ><em
              >非 2xx 或超时后 1 分、5 分、30 分、2 小时、6 小时重试，共 5
              次</em
            ></div
          >
          <el-tabs v-model="lang" class="sig-tabs">
            <el-tab-pane label="Node.js" name="node" />
            <el-tab-pane label="PHP" name="php" />
            <el-tab-pane label="Python" name="python" />
          </el-tabs>
          <pre class="code-block mono">{{ samples[lang] }}</pre>
        </section-card>
      </el-col>

      <el-col :lg="15" :xs="24">
        <ele-card class="table-section" :body-style="{ paddingTop: '8px' }">
          <ele-pro-table
            ref="tableRef"
            row-key="id"
            :columns="columns"
            :datasource="datasource"
            :cache-key="'merchantWebhookDeliveries'"
            :page-size="20"
          >
            <template #toolbar>
              <el-space>
                <span class="muted-text"
                  >投递记录{{
                    current ? `：${current.name}` : '（全部）'
                  }}</span
                >
                <el-select
                  v-model="statusFilter"
                  placeholder="状态"
                  clearable
                  style="width: 110px"
                  size="small"
                  @change="reload"
                >
                  <el-option value="pending" label="待投递" />
                  <el-option value="success" label="成功" />
                  <el-option value="failed" label="失败" />
                </el-select>
                <el-button link type="primary" :icon="Refresh" @click="reload"
                  >刷新</el-button
                >
              </el-space>
            </template>
            <template #event="{ row }">
              <span class="mono">{{ row.event }}</span>
            </template>
            <template #status="{ row }">
              <el-tag
                :type="statusType(row.status)"
                size="small"
                effect="light"
                >{{ statusLabel(row.status) }}</el-tag
              >
              <span class="muted-text" style="margin-left: 6px">{{
                row.responseStatus || row.lastError || ''
              }}</span>
            </template>
            <template #attempt="{ row }"
              >{{ row.attempt }} / {{ row.maxAttempts }}</template
            >
            <template #action="{ row }">
              <el-button link type="primary" @click="showDelivery(row)"
                >详情</el-button
              >
              <el-button
                v-permission="'webhook:edit'"
                link
                type="primary"
                @click="redo(row)"
                >重发</el-button
              >
            </template>
          </ele-pro-table>
        </ele-card>
      </el-col>
    </el-row>

    <ele-modal
      v-model="editVisible"
      :title="form.id ? '编辑 Webhook' : '新增 Webhook'"
      width="560px"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
        <el-form-item label="名称" prop="name"
          ><el-input v-model="form.name" maxlength="128"
        /></el-form-item>
        <el-form-item label="地址" prop="url">
          <el-input
            v-model="form.url"
            placeholder="https://your-server.com/cloak/webhook"
          />
          <div class="muted-text form-tip"
            >生产只允许 https 且不能是内网地址</div
          >
        </el-form-item>
        <el-form-item label="事件">
          <el-checkbox-group v-model="form.events">
            <el-checkbox
              v-for="ev in events"
              :key="ev"
              :value="ev"
              :label="ev"
              class="ev-check"
              ><span class="mono">{{ ev }}</span></el-checkbox
            >
          </el-checkbox-group>
          <div class="muted-text form-tip">不勾选表示订阅全部事件</div>
        </el-form-item>
        <el-form-item v-if="form.id" label="启用"
          ><el-switch v-model="form.enabled"
        /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save"
          >保存</el-button
        >
      </template>
    </ele-modal>

    <ele-modal v-model="secretVisible" title="Webhook 密钥" width="520px">
      <el-alert
        type="warning"
        show-icon
        :closable="false"
        title="密钥只显示这一次，请立即保存到你的服务端配置。"
        style="margin-bottom: 12px"
      />
      <el-input :model-value="secret" readonly>
        <template #append
          ><el-button @click="copy(secret)">复制</el-button></template
        >
      </el-input>
    </ele-modal>

    <ele-modal v-model="deliveryVisible" title="投递详情" width="640px">
      <template v-if="delivery">
        <div class="side-kv"
          ><span>事件</span
          ><em class="mono"
            >{{ delivery.event }} · {{ delivery.eventId }}</em
          ></div
        >
        <div class="side-kv"
          ><span>状态</span
          ><em
            >{{ statusLabel(delivery.status) }} · HTTP
            {{ delivery.responseStatus || '-' }} · 第
            {{ delivery.attempt }} 次</em
          ></div
        >
        <div v-if="delivery.lastError" class="side-kv"
          ><span>错误</span
          ><em class="error-text">{{ delivery.lastError }}</em></div
        >
        <div class="side-kv"
          ><span>下次重试</span
          ><em>{{
            delivery.status === 'pending' ? delivery.nextAttemptAt : '-'
          }}</em></div
        >
        <p class="muted-text" style="margin: 10px 0 4px">请求体</p>
        <pre class="code-block mono">{{
          JSON.stringify(delivery.payload, null, 2)
        }}</pre>
        <p class="muted-text" style="margin: 10px 0 4px">响应</p>
        <pre class="code-block mono">{{
          delivery.responseBody || '（空）'
        }}</pre>
      </template>
    </ele-modal>
  </ele-page>
</template>

<script setup>
  import { ref, computed, onMounted } from 'vue';
  import { Plus, Refresh } from '@element-plus/icons-vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage } from 'ele-admin-plus';
  import {
    getWebhooks,
    createWebhook,
    updateWebhook,
    rotateWebhookSecret,
    testWebhook,
    deleteWebhook,
    pageDeliveries,
    getDelivery,
    redeliver
  } from '@/api/merchant/ops';

  defineOptions({ name: 'MerchantWebhook' });

  const endpoints = ref([]);
  const events = ref([]);
  const currentId = ref(null);
  const current = computed(() =>
    endpoints.value.find((e) => e.id === currentId.value)
  );
  const statusFilter = ref('');
  const tableRef = ref(null);
  const editVisible = ref(false);
  const secretVisible = ref(false);
  const deliveryVisible = ref(false);
  const secret = ref('');
  const delivery = ref(null);
  const saving = ref(false);
  const formRef = ref(null);
  const form = ref({ id: 0, name: '', url: '', events: [], enabled: true });
  const lang = ref('node');
  const rules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    url: [{ required: true, message: '请输入地址', trigger: 'blur' }]
  };

  const columns = [
    { prop: 'createdAt', label: '时间', width: 160 },
    { prop: 'event', label: '事件', slot: 'event', minWidth: 180 },
    { prop: 'status', label: '状态', slot: 'status', minWidth: 160 },
    {
      prop: 'attempt',
      label: '尝试',
      slot: 'attempt',
      width: 80,
      align: 'center'
    },
    { prop: 'deliveredAt', label: '送达', width: 160 },
    {
      prop: 'action',
      label: '操作',
      slot: 'action',
      width: 110,
      align: 'center'
    }
  ];
  const datasource = ({ page, limit }) =>
    pageDeliveries({
      endpointId: currentId.value || undefined,
      status: statusFilter.value || undefined,
      pageNum: page,
      pageSize: limit
    });
  const reload = () => tableRef.value?.reload?.({ page: 1 });

  const load = () =>
    getWebhooks()
      .then((d) => {
        endpoints.value = d.list || [];
        events.value = d.events || [];
        if (
          currentId.value &&
          !endpoints.value.some((e) => e.id === currentId.value)
        )
          currentId.value = null;
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));

  const openEdit = (e) => {
    form.value = e
      ? {
          id: e.id,
          name: e.name,
          url: e.url,
          events: [...(e.events || [])],
          enabled: e.enabled
        }
      : { id: 0, name: '', url: '', events: [], enabled: true };
    editVisible.value = true;
  };
  const save = () => {
    formRef.value?.validate?.((ok) => {
      if (!ok) return;
      saving.value = true;
      const body = {
        name: form.value.name,
        url: form.value.url,
        events: form.value.events,
        enabled: form.value.enabled
      };
      const p = form.value.id
        ? updateWebhook(form.value.id, body)
        : createWebhook(body);
      p.then((d) => {
        editVisible.value = false;
        if (d?.secret) {
          secret.value = d.secret;
          secretVisible.value = true;
        } else {
          EleMessage.success({ message: '已保存', plain: true });
        }
        load();
      })
        .catch((e) => EleMessage.error({ message: e.message, plain: true }))
        .finally(() => (saving.value = false));
    });
  };
  const rotate = (e) =>
    ElMessageBox.confirm(
      '重置后旧密钥立即失效，需要同步更新你的服务端。继续？',
      '重置密钥',
      { type: 'warning', draggable: true }
    )
      .then(() => rotateWebhookSecret(e.id))
      .then((d) => {
        secret.value = d.secret;
        secretVisible.value = true;
        load();
      })
      .catch(() => {});
  const remove = (e) =>
    ElMessageBox.confirm(
      `删除「${e.name}」后不再推送，历史投递记录保留。`,
      '删除 Webhook',
      { type: 'warning', draggable: true }
    )
      .then(() => deleteWebhook(e.id))
      .then(() => {
        EleMessage.success({ message: '已删除', plain: true });
        load();
        reload();
      })
      .catch(() => {});
  const sendTest = (e) =>
    testWebhook(e.id)
      .then((d) => {
        EleMessage[d.status === 'success' ? 'success' : 'warning']({
          message:
            d.status === 'success'
              ? `测试成功，HTTP ${d.responseStatus}`
              : `测试失败：${d.lastError || d.responseStatus}`,
          plain: true
        });
        currentId.value = e.id;
        load();
        reload();
      })
      .catch((err) => EleMessage.error({ message: err.message, plain: true }));
  const showDelivery = (row) =>
    getDelivery(row.id)
      .then((d) => {
        delivery.value = d;
        deliveryVisible.value = true;
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));
  const redo = (row) =>
    redeliver(row.id)
      .then((d) => {
        EleMessage[d.status === 'success' ? 'success' : 'warning']({
          message:
            d.status === 'success' ? '重发成功' : `重发失败：${d.lastError}`,
          plain: true
        });
        reload();
        load();
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));
  const copy = async (text) => {
    try {
      await navigator.clipboard.writeText(text);
      EleMessage.success({ message: '已复制', plain: true });
    } catch {
      EleMessage.error({ message: '复制失败，请手动选择', plain: true });
    }
  };
  const statusLabel = (s) =>
    ({ pending: '待投递', success: '成功', failed: '失败' })[s] || s;
  const statusType = (s) =>
    ({ pending: 'info', success: 'success', failed: 'danger' })[s] || 'info';

  const samples = {
    node: `const crypto = require('crypto');
app.post('/cloak/webhook', express.raw({ type: '*/*' }), (req, res) => {
  const ts = req.get('X-Cloak-Timestamp');
  const sig = (req.get('X-Cloak-Signature') || '').replace(/^v1=/, '');
  const expect = crypto.createHmac('sha256', process.env.CLOAK_WEBHOOK_SECRET)
    .update(ts + '.' + req.body).digest('hex');
  if (Math.abs(Date.now() / 1000 - ts) > 300 ||
      !crypto.timingSafeEqual(Buffer.from(expect), Buffer.from(sig))) return res.sendStatus(401);
  const event = JSON.parse(req.body); // { id, event, merchantId, createdAt, data }
  // 用 event.id 做幂等
  res.sendStatus(200);
});`,
    php: `$raw = file_get_contents('php://input');
$ts  = $_SERVER['HTTP_X_CLOAK_TIMESTAMP'] ?? '';
$sig = preg_replace('/^v1=/', '', $_SERVER['HTTP_X_CLOAK_SIGNATURE'] ?? '');
$expect = hash_hmac('sha256', $ts . '.' . $raw, getenv('CLOAK_WEBHOOK_SECRET'));
if (abs(time() - (int) $ts) > 300 || !hash_equals($expect, $sig)) { http_response_code(401); exit; }
$event = json_decode($raw, true); // ['id','event','merchantId','createdAt','data']
http_response_code(200);`,
    python: `import hmac, hashlib, time
ts = request.headers.get("X-Cloak-Timestamp", "")
sig = request.headers.get("X-Cloak-Signature", "").removeprefix("v1=")
expect = hmac.new(SECRET.encode(), f"{ts}.".encode() + request.get_data(), hashlib.sha256).hexdigest()
if abs(time.time() - int(ts or 0)) > 300 or not hmac.compare_digest(expect, sig):
    abort(401)
event = request.get_json()  # id / event / merchantId / createdAt / data`
  };

  onMounted(load);
</script>

<style scoped>
  .ep-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .ep-item {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 10px;
    padding: 12px 14px;
    cursor: pointer;
    transition:
      border-color 0.15s,
      box-shadow 0.15s;
  }

  .ep-item:hover {
    border-color: var(--el-color-primary-light-5);
  }

  .ep-item.is-active {
    border-color: var(--el-color-primary);
    box-shadow: 0 0 0 2px var(--el-color-primary-light-8);
  }

  .ep-head {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 4px;
  }

  .ep-url {
    font-size: 12px;
    word-break: break-all;
    margin-bottom: 4px;
  }

  .ep-actions {
    margin-top: 6px;
  }

  .sig-tabs {
    margin-top: 10px;
  }

  .sig-tabs :deep(.el-tabs__header) {
    margin-bottom: 8px;
  }

  .code-block {
    margin: 0;
    padding: 12px 14px;
    border-radius: 8px;
    background: var(--el-fill-color-light);
    font-size: 12px;
    line-height: 1.6;
    overflow: auto;
    max-height: 260px;
    white-space: pre;
  }

  .ev-check {
    width: 100%;
    margin-right: 0;
    height: 26px;
  }

  .form-tip {
    margin-top: 4px;
  }

  .muted-text {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .error-text {
    color: var(--el-color-danger);
  }
</style>
