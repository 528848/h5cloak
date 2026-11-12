<template>
  <ele-page>
    <section-card
      class="page-section"
      title="全局开关"
      subtitle="保存后数秒内在网关生效"
    >
      <el-alert
        type="warning"
        :closable="false"
        show-icon
        title="紧急审核打开后，全平台原本放行的访问会改为审核页。静态资源和明确拒绝的访问保持原样。"
        style="margin-bottom: 16px"
      />
      <el-form label-width="140px" :disabled="!canEdit">
        <el-form-item label="紧急全平台审核">
          <div class="switch-line">
            <el-switch
              v-model="form.emergencyReview"
              :before-change="confirmEmergency"
            />
            <span class="hint">打开前会再次确认。关闭后按各站点自己的规则放行。</span>
          </div>
        </el-form-item>
        <el-form-item label="托管接入">
          <div class="switch-line">
            <el-switch v-model="form.entries.hosted" />
            <span class="hint">关闭后，绑定域名上的访问不再进入判定。</span>
          </div>
        </el-form-item>
        <el-form-item label="开放接口 v2">
          <div class="switch-line">
            <el-switch v-model="form.entries.api_v2" />
            <span class="hint">关闭后，v2 判定与上报接口停止服务。</span>
          </div>
        </el-form-item>
        <el-form-item label="开放接口 v1">
          <div class="switch-line">
            <el-switch v-model="form.entries.api_v1" />
            <span class="hint">关闭后，旧版判定接口停止服务。</span>
          </div>
        </el-form-item>
        <el-form-item label="JS SDK">
          <div class="switch-line">
            <el-switch v-model="form.entries.js" />
            <span class="hint">关闭后，网页脚本上报不再受理。</span>
          </div>
        </el-form-item>
        <el-form-item label="日志保留上限">
          <el-input-number
            v-model="form.logRetentionDays"
            :min="7"
            :max="3650"
          />
          <span class="hint">天。商户自身设置更短时仍按商户执行。</span>
        </el-form-item>
        <el-form-item label="每密钥每秒">
          <el-input-number v-model="form.ratePerKey" :min="1" :max="10000" />
          <span class="hint">次。超过后该密钥返回限流。</span>
        </el-form-item>
        <el-form-item label="每站点每分钟">
          <el-input-number v-model="form.ratePerSite" :min="1" :max="1000000" />
          <span class="hint">次。超过后该站点返回限流。</span>
        </el-form-item>
        <el-form-item label="Webhook 重试">
          <el-input-number
            v-model="form.webhookMaxAttempts"
            :min="1"
            :max="20"
          />
          <span class="hint">次。对新产生的投递生效。</span>
        </el-form-item>
      </el-form>
    </section-card>

    <section-card
      class="page-section"
      title="广告 Referer 与内置浏览器"
      subtitle="用来判断访问是否来自广告或社媒 App"
    >
      <el-form label-width="140px" :disabled="!canEdit">
        <el-form-item label="自定义 Referer">
          <div class="switch-line">
            <el-switch v-model="form.adReferersCustom" />
            <span class="hint">关闭后恢复内置域名表。</span>
          </div>
        </el-form-item>
        <el-form-item label="Referer 域名">
          <el-input
            v-model="refererText"
            type="textarea"
            :rows="6"
            placeholder="每行一个域名，例如 facebook.com"
            :disabled="!canEdit || !form.adReferersCustom"
          />
        </el-form-item>
        <el-form-item label="自定义浏览器特征">
          <div class="switch-line">
            <el-switch v-model="form.inAppMarkersCustom" />
            <span class="hint">关闭后恢复内置特征表。</span>
          </div>
        </el-form-item>
        <el-form-item label="特征列表">
          <el-input
            v-model="markerText"
            type="textarea"
            :rows="6"
            placeholder="每行：特征 应用名，例如 FBAN facebook"
            :disabled="!canEdit || !form.inAppMarkersCustom"
          />
        </el-form-item>
      </el-form>
    </section-card>

    <section-card
      class="page-section"
      title="静态资源规则"
      subtitle="全平台生效。样式、脚本、图片和字体直接回源，不进入归因。商户端不展示，也不能关闭。"
    >
      <el-table :data="staticPathRules" border>
        <el-table-column prop="pattern" label="路径" min-width="140" />
        <el-table-column label="匹配" width="120">
          <template #default="{ row }">{{
            matchLabel(row.matchType)
          }}</template>
        </el-table-column>
        <el-table-column label="动作" width="140">
          <template #default="{ row }">{{
            actionLabel(row.action)
          }}</template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="140" />
      </el-table>
    </section-card>

    <section-card
      class="page-section"
      title="新建站点的默认路径规则"
      subtitle="只影响之后新建的站点，已有站点保持自己的副本。静态资源规则不在这里。"
    >
      <el-form :disabled="!canEdit">
        <el-form-item label="自定义规则" label-width="140px">
          <div class="switch-line">
            <el-switch v-model="form.defaultPathRulesCustom" />
            <span class="hint">关闭后，新站点使用内置路径规则。参数字典仍在「参数字典」维护。</span>
          </div>
        </el-form-item>
      </el-form>
      <el-table :data="form.defaultPathRules" border>
        <el-table-column label="匹配" min-width="180">
          <template #default="{ row }">
            <el-input
              v-model="row.pattern"
              placeholder="如 /assets"
              :disabled="!canEdit || !form.defaultPathRulesCustom"
            />
          </template>
        </el-table-column>
        <el-table-column label="方式" width="140">
          <template #default="{ row }">
            <el-select
              v-model="row.matchType"
              class="ele-fluid"
              :disabled="!canEdit || !form.defaultPathRulesCustom"
            >
              <el-option label="前缀" value="prefix" />
              <el-option label="精确" value="exact" />
              <el-option label="后缀" value="suffix" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="动作" width="160">
          <template #default="{ row }">
            <el-select
              v-model="row.action"
              class="ele-fluid"
              :disabled="!canEdit || !form.defaultPathRulesCustom"
            >
              <el-option label="直接放行" value="bypass" />
              <el-option label="始终真实" value="always_real" />
              <el-option label="始终审核" value="always_review" />
              <el-option label="拒绝" value="block" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="排序" width="110">
          <template #default="{ row }">
            <el-input-number
              v-model="row.sort"
              :min="0"
              :controls="false"
              :disabled="!canEdit || !form.defaultPathRulesCustom"
            />
          </template>
        </el-table-column>
        <el-table-column label="备注" min-width="140">
          <template #default="{ row }">
            <el-input
              v-model="row.remark"
              placeholder="可选"
              :disabled="!canEdit || !form.defaultPathRulesCustom"
            />
          </template>
        </el-table-column>
      </el-table>
    </section-card>

    <section-card
      class="page-section"
      title="网关节点"
      subtitle="只展示最近有心跳的节点"
    >
      <el-empty v-if="!nodes.length" :image-size="70" description="暂无在线节点" />
      <el-row v-else :gutter="12">
        <el-col
          v-for="n in nodes"
          :key="n.node"
          :lg="8"
          :md="12"
          :sm="24"
          :xs="24"
        >
          <div class="node-card">
            <strong>{{ n.node }}</strong>
            <span class="hint">{{ n.version || 'dev' }}</span>
            <div class="node-meta">
              最近心跳 {{ n.seenAt || '-' }} · 站点 {{ n.sites }} · QPS
              {{ n.qps }}
            </div>
          </div>
        </el-col>
      </el-row>
    </section-card>

    <ele-card>
      <div class="form-footer">
        <span v-if="form.updatedAt" class="hint footer-time">
          上次保存 {{ form.updatedAt }}
        </span>
        <el-button v-if="canEdit" @click="discard">取消</el-button>
        <el-button
          v-permission="'system:config:edit'"
          type="primary"
          :loading="saving"
          @click="save"
        >
          保存
        </el-button>
      </div>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { computed, reactive, ref } from 'vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage } from 'ele-admin-plus';
  import { usePermission } from '@/utils/use-permission';
  import { getSettings, saveSettings, listNodes } from '@/api/admin/platform';

  defineOptions({ name: 'AdminSystem' });

  const { hasPermission } = usePermission();
  const canEdit = computed(() => hasPermission('system:config:edit'));
  const saving = ref(false);
  const staticPathRules = ref([]);
  const matchLabel = (v) =>
    ({ prefix: '前缀', exact: '精确', suffix: '后缀' })[v] || v;
  const actionLabel = (v) =>
    ({
      bypass: '不拦截',
      always_real: '始终真实',
      always_review: '始终审核',
      block: '拒绝'
    })[v] || v;
  const nodes = ref([]);
  const refererText = ref('');
  const markerText = ref('');
  const form = reactive({
    emergencyReview: false,
    entries: { hosted: true, api_v2: true, api_v1: true, js: true },
    logRetentionDays: 90,
    ratePerKey: 50,
    ratePerSite: 2000,
    webhookMaxAttempts: 5,
    adReferersCustom: false,
    inAppMarkersCustom: false,
    defaultPathRulesCustom: false,
    defaultPathRules: [],
    updatedAt: ''
  });

  const confirmEmergency = () => {
    const turningOn = !form.emergencyReview;
    const message = turningOn
      ? '打开后，全平台原本放行的访问会在数秒内改为审核页。静态资源和明确拒绝的访问保持原样。是否确认打开？'
      : '关闭后，全平台恢复按各站点自己的规则放行。是否确认关闭？';
    return ElMessageBox.confirm(message, '系统提示', {
      type: 'warning',
      draggable: true,
      confirmButtonText: turningOn ? '确认打开' : '确认关闭',
      cancelButtonText: '取消'
    })
      .then(() => true)
      .catch(() => false);
  };

  const load = async () => {
    const data = await getSettings();
    Object.assign(form, data);
    form.entries = {
      hosted: true,
      api_v2: true,
      api_v1: true,
      js: true,
      ...(data.entries || {})
    };
    form.defaultPathRules = data.defaultPathRules || [];
    staticPathRules.value = data.staticPathRules || [];
    refererText.value = (data.adReferers || []).join('\n');
    markerText.value = (data.inAppMarkers || [])
      .map((m) => `${m.marker} ${m.app}`)
      .join('\n');
    nodes.value = (await listNodes()) || [];
  };

  const discard = () => {
    load().catch((e) => EleMessage.error({ message: e.message, plain: true }));
  };

  const save = async () => {
    saving.value = true;
    try {
      const msg = await saveSettings({
        ...form,
        adReferers: refererText.value
          .split('\n')
          .map((s) => s.trim())
          .filter(Boolean),
        inAppMarkers: markerText.value
          .split('\n')
          .map((line) => line.trim())
          .filter(Boolean)
          .map((line) => {
            const [marker, ...rest] = line.split(/\s+/);
            return { marker, app: rest.join(' ') };
          })
      });
      EleMessage.success({ message: msg, plain: true });
      await load();
    } catch (e) {
      EleMessage.error({ message: e.message, plain: true });
    } finally {
      saving.value = false;
    }
  };

  load().catch((e) => EleMessage.error({ message: e.message, plain: true }));
</script>

<style scoped>
  .switch-line {
    display: flex;
    align-items: center;
    min-height: 32px;
  }

  .hint {
    margin-left: 8px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }

  .node-card {
    margin-bottom: 12px;
    padding: 14px 16px;
    border-radius: 10px;
    background: var(--el-fill-color-extra-light);
  }

  .node-card strong {
    margin-right: 8px;
  }

  .node-meta {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }

  .form-footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 12px;
  }

  .footer-time {
    margin-right: auto;
    margin-left: 0;
  }
</style>
