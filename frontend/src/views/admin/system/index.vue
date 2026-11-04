<template>
  <ele-page>
    <ele-card header="全局开关">
      <el-alert
        type="warning"
        :closable="false"
        show-icon
        title="紧急审核打开后，全平台原本放行的访问会改为审核页。网关在数秒内生效，静态资源和明确拒绝的访问保持原样。"
        style="margin-bottom: 16px"
      />
      <el-form label-width="140px" :disabled="!canEdit">
        <el-form-item label="紧急全平台审核">
          <el-switch v-model="form.emergencyReview" />
        </el-form-item>
        <el-form-item label="托管接入">
          <el-switch v-model="form.entries.hosted" />
        </el-form-item>
        <el-form-item label="开放接口 v2">
          <el-switch v-model="form.entries.api_v2" />
        </el-form-item>
        <el-form-item label="开放接口 v1">
          <el-switch v-model="form.entries.api_v1" />
        </el-form-item>
        <el-form-item label="JS SDK">
          <el-switch v-model="form.entries.js" />
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
        </el-form-item>
        <el-form-item label="每站点每分钟">
          <el-input-number v-model="form.ratePerSite" :min="1" :max="1000000" />
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
    </ele-card>

    <ele-card header="广告 Referer 与内置浏览器" style="margin-top: 12px">
      <el-form label-width="140px" :disabled="!canEdit">
        <el-form-item label="自定义 Referer">
          <el-switch v-model="form.adReferersCustom" />
          <span class="hint">关闭后恢复内置域名表。</span>
        </el-form-item>
        <el-form-item label="Referer 域名">
          <el-input
            v-model="refererText"
            type="textarea"
            :rows="6"
            placeholder="每行一个域名"
            :disabled="!canEdit || !form.adReferersCustom"
          />
        </el-form-item>
        <el-form-item label="自定义浏览器特征">
          <el-switch v-model="form.inAppMarkersCustom" />
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
    </ele-card>

    <ele-card header="新建站点的默认路径规则" style="margin-top: 12px">
      <el-form :disabled="!canEdit">
        <el-form-item label="自定义规则" label-width="140px">
          <el-switch v-model="form.defaultPathRulesCustom" />
          <span class="hint"
            >只影响之后新建的站点，已有站点保持自己的副本。内置参数字典仍在「参数字典」维护。</span
          >
        </el-form-item>
      </el-form>
      <el-table :data="form.defaultPathRules" border size="small">
        <el-table-column label="匹配" min-width="180">
          <template #default="{ row }">
            <el-input
              v-model="row.pattern"
              :disabled="!canEdit || !form.defaultPathRulesCustom"
            />
          </template>
        </el-table-column>
        <el-table-column label="方式" width="130">
          <template #default="{ row }">
            <el-select
              v-model="row.matchType"
              :disabled="!canEdit || !form.defaultPathRulesCustom"
            >
              <el-option label="前缀" value="prefix" />
              <el-option label="精确" value="exact" />
              <el-option label="后缀" value="suffix" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="动作" width="150">
          <template #default="{ row }">
            <el-select
              v-model="row.action"
              :disabled="!canEdit || !form.defaultPathRulesCustom"
            >
              <el-option label="直接放行" value="bypass" />
              <el-option label="始终真实" value="always_real" />
              <el-option label="始终审核" value="always_review" />
              <el-option label="拒绝" value="block" />
            </el-select>
          </template>
        </el-table-column>
        <el-table-column label="排序" width="100">
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
              :disabled="!canEdit || !form.defaultPathRulesCustom"
            />
          </template>
        </el-table-column>
      </el-table>
    </ele-card>

    <ele-card header="网关节点" style="margin-top: 12px">
      <el-empty v-if="!nodes.length" description="暂无在线节点" />
      <el-row :gutter="12">
        <el-col v-for="n in nodes" :key="n.node" :span="8">
          <el-card shadow="never">
            <div>{{ n.node }} · {{ n.version || 'dev' }}</div>
            <div class="hint"
              >最近心跳 {{ n.seenAt || '-' }} · 站点 {{ n.sites }} · QPS
              {{ n.qps }}</div
            >
          </el-card>
        </el-col>
      </el-row>
    </ele-card>

    <div style="margin-top: 16px">
      <el-button
        v-permission="'system:config:edit'"
        type="primary"
        :loading="saving"
        @click="save"
      >
        保存
      </el-button>
      <span v-if="form.updatedAt" class="hint"
        >上次保存 {{ form.updatedAt }}</span
      >
    </div>
  </ele-page>
</template>

<script setup>
  import { computed, reactive, ref } from 'vue';
  import { EleMessage } from 'ele-admin-plus';
  import { usePermission } from '@/utils/use-permission';
  import { getSettings, saveSettings, listNodes } from '@/api/admin/platform';

  defineOptions({ name: 'AdminSystem' });

  const { hasPermission } = usePermission();
  const canEdit = computed(() => hasPermission('system:config:edit'));
  const saving = ref(false);
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
    refererText.value = (data.adReferers || []).join('\n');
    markerText.value = (data.inAppMarkers || [])
      .map((m) => `${m.marker} ${m.app}`)
      .join('\n');
    nodes.value = (await listNodes()) || [];
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
  .hint {
    margin-left: 8px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
</style>
