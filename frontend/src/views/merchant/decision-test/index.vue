<template>
  <ele-page hide-footer>
    <el-row :gutter="16">
      <el-col :lg="11" :xs="24">
        <section-card
          title="模拟一次访问"
          subtitle="用当前最新配置判定，不写日志、不影响真实访客"
          class="page-section"
        >
          <el-form
            ref="formRef"
            :model="form"
            :rules="rules"
            label-width="96px"
            @submit.prevent=""
          >
            <el-form-item label="站点" prop="siteId">
              <el-select
                v-model="form.siteId"
                placeholder="请选择站点"
                class="ele-fluid"
              >
                <el-option
                  v-for="s in sites"
                  :key="s.siteId"
                  :label="s.name"
                  :value="s.siteId"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="访问地址" prop="url">
              <el-input
                v-model.trim="form.url"
                type="textarea"
                :rows="2"
                class="mono"
                placeholder="https://shop.example.com/lp?ref=ALICE&utm_source=tiktok"
              />
            </el-form-item>
            <el-form-item label="浏览器">
              <el-select
                v-model="form.userAgent"
                filterable
                allow-create
                default-first-option
                placeholder="选择预设或粘贴 User-Agent"
                class="ele-fluid"
              >
                <el-option
                  v-for="p in uaPresets"
                  :key="p.label"
                  :label="p.label"
                  :value="p.value"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="访客 Cookie">
              <el-input
                v-model.trim="form.cookieToken"
                type="textarea"
                :rows="2"
                class="mono"
                placeholder="粘贴上一次结果中的令牌，模拟回访"
              />
            </el-form-item>
            <el-row :gutter="12">
              <el-col :span="12">
                <el-form-item label="访客 IP">
                  <el-input
                    v-model.trim="form.ip"
                    placeholder="203.0.113.10"
                    class="mono"
                  />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="Referer">
                  <el-input
                    v-model.trim="form.referer"
                    placeholder="https://l.facebook.com/"
                  />
                </el-form-item>
              </el-col>
            </el-row>
          </el-form>
          <template #footer>
            <div class="form-bar">
              <el-button plain @click="handleReset">清空</el-button>
              <el-button v-if="result?.cookie" plain @click="useCookie">
                用结果令牌模拟回访
              </el-button>
              <el-button type="primary" :loading="running" @click="handleRun">
                执行判定
              </el-button>
            </div>
          </template>
        </section-card>

        <section-card
          title="配置快照"
          subtitle="网关实际使用的配置版本"
          class="page-section"
        >
          <template #extra>
            <el-space>
              <el-button link type="primary" :icon="Refresh" @click="loadStatus"
                >刷新</el-button
              >
              <el-button
                v-permission="'site:site:edit'"
                link
                type="primary"
                :loading="publishing"
                @click="handlePublish"
              >
                立即发布
              </el-button>
            </el-space>
          </template>
          <div v-if="status" class="side-list">
            <div class="side-kv">
              <span>Redis</span>
              <el-tag
                :type="status.redis ? 'success' : 'info'"
                size="small"
                effect="plain"
              >
                {{ status.redis ? '已连接' : '未配置' }}
              </el-tag>
            </div>
            <div class="side-kv">
              <span>已发布</span>
              <el-tag
                :type="status.published ? 'success' : 'warning'"
                size="small"
                effect="plain"
              >
                {{ status.published ? status.publishedAt : '未发布' }}
              </el-tag>
            </div>
            <div class="side-kv">
              <span>与当前配置一致</span>
              <el-tag
                v-if="status.published"
                :type="status.inSync ? 'success' : 'danger'"
                size="small"
                effect="light"
              >
                {{ status.inSync ? '一致' : '不一致' }}
              </el-tag>
              <span v-else class="muted-text">-</span>
            </div>
            <div class="side-kv">
              <span>参数 / 归因码 / 路径规则</span>
              <strong>
                {{ status.local.params }} / {{ status.local.codes }} /
                {{ status.local.pathRules }}
              </strong>
            </div>
            <div class="side-kv">
              <span>令牌密钥版本</span>
              <strong>v{{ status.local.keyVersion }}</strong>
            </div>
            <div class="side-kv">
              <span>服务状态</span>
              <el-tag
                :type="status.local.serving ? 'success' : 'info'"
                size="small"
                effect="plain"
              >
                {{ status.local.serving ? '服务中' : '已停止' }}
              </el-tag>
            </div>
          </div>
          <el-empty v-else description="请选择站点" :image-size="60" />
        </section-card>
      </el-col>

      <el-col :lg="13" :xs="24">
        <section-card
          title="判定结果"
          subtitle="同一地址，访客看到哪种内容"
          class="page-section"
        >
          <template v-if="result">
            <div class="result-hero" :class="`is-${result.decision.content}`">
              <div class="result-main">
                <el-tag
                  :type="getOptionType(contentOptions, result.decision.content)"
                  effect="dark"
                  size="large"
                >
                  {{ getOptionLabel(contentOptions, result.decision.content) }}
                </el-tag>
                <el-tag
                  :type="getOptionType(touchOptions, result.decision.touchType)"
                  effect="light"
                >
                  {{ getOptionLabel(touchOptions, result.decision.touchType) }}
                  <template v-if="result.decision.handoffKind">
                    ·
                    {{
                      result.decision.handoffKind === 'token'
                        ? '交接令牌'
                        : '设备匹配'
                    }}
                  </template>
                </el-tag>
                <el-tag
                  v-if="result.decision.attributed"
                  :type="
                    getOptionType(confidenceOptions, result.decision.confidence)
                  "
                  effect="plain"
                >
                  置信度
                  {{
                    getOptionLabel(
                      confidenceOptions,
                      result.decision.confidence
                    )
                  }}
                </el-tag>
              </div>
              <div class="result-desc muted-text">
                耗时 {{ result.latencyMs }} ms · 快照
                {{ result.snapshot.builtAt }}
                <template v-if="!result.hostKnown">
                  ·
                  <span class="error-text"
                    >该域名未绑定到此站点，真实访问会返回 404</span
                  >
                </template>
              </div>
            </div>

            <el-descriptions
              :column="2"
              border
              class="detail-table"
              style="margin-top: 16px"
            >
              <el-descriptions-item label="渠道">
                <span v-if="result.decision.channel" class="mono">{{
                  result.decision.channel
                }}</span>
                <span v-else class="muted-text">-</span>
              </el-descriptions-item>
              <el-descriptions-item label="归因码">
                <template v-if="result.decision.code">
                  <span class="mono">{{ result.decision.code.code }}</span>
                  <span v-if="result.decision.code.name" class="muted-text">
                    （{{ result.decision.code.name }}）
                  </span>
                </template>
                <span v-else class="muted-text">-</span>
              </el-descriptions-item>
              <el-descriptions-item label="识别到的参数" :span="2">
                <el-space
                  v-if="result.decision.matchedParams?.length"
                  wrap
                  :size="4"
                >
                  <el-tag
                    v-for="p in result.decision.matchedParams"
                    :key="p"
                    size="small"
                    effect="plain"
                    class="mono"
                    >{{ p }}</el-tag
                  >
                </el-space>
                <span v-else class="muted-text">无</span>
              </el-descriptions-item>
              <el-descriptions-item label="浏览器" :span="2">
                <template v-if="result.decision.inAppBrowser.detected">
                  <el-tag type="warning" size="small" effect="light">
                    内置浏览器 · {{ result.decision.inAppBrowser.app }}
                  </el-tag>
                </template>
                <span v-else class="muted-text">系统浏览器</span>
                <span class="muted-text" style="margin-left: 8px">
                  {{ result.decision.inAppBrowser.os }} ·
                  {{ result.decision.inAppBrowser.deviceFamily }}
                </span>
              </el-descriptions-item>
              <el-descriptions-item
                v-if="result.decision.pathRule"
                label="命中路径规则"
                :span="2"
              >
                <span class="mono">{{ result.decision.pathRule.pattern }}</span>
                <span class="muted-text">
                  （{{ result.decision.pathRule.matchType }} →
                  {{ result.decision.pathRule.action }}）
                </span>
              </el-descriptions-item>
            </el-descriptions>

            <div
              v-if="result.decision.validationErrors?.length"
              class="block-gap"
            >
              <div class="app-card-title"
                ><span>校验失败</span><em>这些参数没有通过</em></div
              >
              <el-table
                :data="result.decision.validationErrors"
                size="small"
                border
              >
                <el-table-column prop="param" label="参数" width="140">
                  <template #default="{ row }"
                    ><span class="mono">{{ row.param }}</span></template
                  >
                </el-table-column>
                <el-table-column prop="value" label="值" min-width="140">
                  <template #default="{ row }">
                    <span class="mono">{{ row.value || '-' }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="原因" width="160">
                  <template #default="{ row }">
                    <el-tag type="danger" size="small" effect="light">
                      {{ errorReasons[row.reason] || row.reason }}
                    </el-tag>
                  </template>
                </el-table-column>
              </el-table>
            </div>

            <div class="block-gap">
              <div class="app-card-title"
                ><span>判定过程</span><em>引擎每一步的结论</em></div
              >
              <el-timeline class="detail-timeline">
                <el-timeline-item
                  v-for="(t, i) in result.decision.trace"
                  :key="i"
                  :type="
                    i === result.decision.trace.length - 1 ? 'primary' : ''
                  "
                >
                  <div class="log-content">{{
                    traceSteps[t.step] || t.step
                  }}</div>
                  <div class="log-operator mono">{{ t.result }}</div>
                </el-timeline-item>
              </el-timeline>
            </div>

            <div v-if="result.cookie" class="block-gap">
              <div class="app-card-title">
                <span>将写入的 Cookie</span
                ><em
                  >{{ result.cookie.name }}，有效
                  {{ Math.round(result.cookie.maxAge / 86400) }} 天</em
                >
              </div>
              <div class="token-box mono">{{ result.cookie.value }}</div>
            </div>
          </template>
          <el-empty
            v-else
            description="填写左侧参数后执行判定"
            :image-size="90"
          />
        </section-card>
      </el-col>
    </el-row>
  </ele-page>
</template>

<script setup>
  import { ref, reactive, watch, onMounted } from 'vue';
  import { useRoute } from 'vue-router';
  import { Refresh } from '@element-plus/icons-vue';
  import { EleMessage } from 'ele-admin-plus';
  import { useCurrentSite } from '@/utils/use-site';
  import {
    runDecisionTest,
    getSnapshotStatus,
    publishSnapshot
  } from '@/api/merchant/decision';
  import {
    contentOptions,
    touchOptions,
    confidenceOptions,
    errorReasons,
    traceSteps,
    uaPresets,
    getOptionLabel,
    getOptionType
  } from './constants';

  defineOptions({ name: 'MerchantDecisionTest' });

  const route = useRoute();
  const { sites, siteId, load: loadSites } = useCurrentSite();
  const formRef = ref(null);
  const running = ref(false);
  const publishing = ref(false);
  const result = ref(null);
  const status = ref(null);

  const form = reactive({
    siteId: Number(route.query.siteId) || siteId.value || undefined,
    url: String(route.query.url || ''),
    userAgent: uaPresets[0].value,
    cookieToken: '',
    ip: '',
    referer: ''
  });

  const rules = {
    siteId: [{ required: true, message: '请选择站点', trigger: 'change' }],
    url: [{ required: true, message: '请输入访问地址', trigger: 'blur' }]
  };

  const loadStatus = () => {
    if (!form.siteId) {
      status.value = null;
      return;
    }
    getSnapshotStatus(form.siteId)
      .then((d) => (status.value = d))
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));
  };

  const handleRun = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) return;
      running.value = true;
      runDecisionTest({ ...form })
        .then((d) => (result.value = d))
        .catch((e) => EleMessage.error({ message: e.message, plain: true }))
        .finally(() => (running.value = false));
    });
  };

  const useCookie = () => {
    form.cookieToken = result.value.cookie.value;
    EleMessage.success({
      message: '已填入令牌，去掉地址中的参数再执行即可模拟回访',
      plain: true
    });
  };

  const handleReset = () => {
    form.url = '';
    form.cookieToken = '';
    form.ip = '';
    form.referer = '';
    result.value = null;
  };

  const handlePublish = () => {
    publishing.value = true;
    publishSnapshot(form.siteId)
      .then((msg) => {
        EleMessage.success({ message: msg, plain: true });
        loadStatus();
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => (publishing.value = false));
  };

  watch(
    () => form.siteId,
    (v) => {
      if (v) siteId.value = v;
      result.value = null;
      loadStatus();
    }
  );

  onMounted(() => {
    loadSites().then(() => {
      if (!form.siteId) form.siteId = siteId.value || undefined;
      loadStatus();
    });
  });
</script>

<style scoped>
  .form-bar {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  .result-hero {
    padding: 16px 18px;
    border-radius: 10px;
    border: 1px solid var(--el-border-color-lighter);
    background: var(--el-fill-color-extra-light);
  }

  .result-hero.is-real {
    border-color: var(--el-color-success-light-7);
    background: var(--el-color-success-light-9);
  }

  .result-hero.is-review {
    border-color: var(--el-color-warning-light-7);
    background: var(--el-color-warning-light-9);
  }

  .result-hero.is-block,
  .result-hero.is-not_found {
    border-color: var(--el-color-danger-light-7);
    background: var(--el-color-danger-light-9);
  }

  .result-main {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .result-desc {
    margin-top: 8px;
    font-size: 12px;
  }

  .block-gap {
    margin-top: 18px;
  }

  .block-gap .app-card-title {
    margin-bottom: 10px;
  }

  .token-box {
    padding: 10px 12px;
    border-radius: 8px;
    background: var(--el-fill-color-light);
    word-break: break-all;
    font-size: 12px;
  }

  .side-kv > strong {
    font-weight: 600;
  }
</style>
