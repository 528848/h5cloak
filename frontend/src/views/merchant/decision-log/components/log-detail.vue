<!-- 决策日志详情抽屉：判定链路 → 归因 → 访客 -->
<template>
  <el-drawer
    v-bind="modalProps"
    :size="mobile ? '100%' : 820"
    title="判定详情"
    destroy-on-close
  >
    <div v-loading="loading">
      <el-empty
        v-if="!loading && !data"
        :image-size="90"
        description="日志不存在或已过期"
      />
      <template v-else-if="data">
        <ele-card
          class="detail-hero page-section"
          :body-style="{ padding: '18px 20px' }"
        >
          <div class="hero-head">
            <div>
              <div class="hero-title">
                <h3 class="mono">{{ data.path }}</h3>
                <el-tag
                  :type="getOptionType(contentOptions, data.content)"
                  effect="dark"
                >
                  {{ getOptionLabel(contentOptions, data.content) }}
                </el-tag>
                <el-tag
                  :type="getOptionType(touchOptions, data.touchType)"
                  effect="light"
                >
                  {{ getOptionLabel(touchOptions, data.touchType) }}
                  <template v-if="data.handoffKind">
                    ·
                    {{ data.handoffKind === 'token' ? '交接令牌' : '设备匹配' }}
                  </template>
                </el-tag>
                <el-tag
                  v-if="data.attributed"
                  :type="getOptionType(confidenceOptions, data.confidence)"
                  effect="plain"
                >
                  置信度
                  {{ getOptionLabel(confidenceOptions, data.confidence) }}
                </el-tag>
                <el-tag v-if="data.fallback" type="danger" effect="plain"
                  >兜底</el-tag
                >
                <el-tag v-if="data.degraded" type="warning" effect="plain"
                  >降级</el-tag
                >
              </div>
              <div class="hero-desc">
                {{ data.createdAt }} · {{ data.siteName }} · {{ data.host }} ·
                请求 <span class="mono">{{ data.requestId }}</span>
              </div>
            </div>
          </div>
          <div class="summary-strip">
            <div class="summary-item is-amount">
              <span>渠道</span>
              <strong class="mono">{{ data.channel || '-' }}</strong>
            </div>
            <div class="summary-item">
              <span>分销码</span>
              <strong class="mono">{{ data.code || '-' }}</strong>
            </div>
            <div class="summary-item">
              <span>浏览器</span>
              <strong>{{
                data.inAppBrowser ? `内置 · ${data.inAppBrowser}` : '系统浏览器'
              }}</strong>
            </div>
            <div class="summary-item">
              <span>设备 / 国家</span>
              <strong
                >{{ data.deviceFamily || '-' }}
                {{ data.country ? `· ${data.country}` : '' }}</strong
              >
            </div>
            <div class="summary-item">
              <span>响应</span>
              <strong
                >{{
                  originKindLabels[data.originKind] || data.originKind || '-'
                }}
                <template v-if="data.originStatus">
                  · {{ data.originStatus }}</template
                >
              </strong>
            </div>
            <div class="summary-item">
              <span>耗时</span>
              <strong>{{ data.latencyMs }} ms</strong>
            </div>
          </div>
        </ele-card>

        <el-row :gutter="16">
          <el-col :md="14" :xs="24">
            <section-card
              title="判定链路"
              subtitle="路径规则 → 令牌 → 参数 → 校验 → 结果"
              class="page-section"
            >
              <el-timeline class="detail-timeline">
                <el-timeline-item
                  v-for="(t, i) in data.trace || []"
                  :key="i"
                  :type="i === (data.trace || []).length - 1 ? 'primary' : ''"
                >
                  <div class="log-content">{{
                    traceSteps[t.step] || t.step
                  }}</div>
                  <div class="log-operator mono">{{ t.result }}</div>
                </el-timeline-item>
              </el-timeline>
              <template v-if="data.validationErrors?.length">
                <div class="app-card-title" style="margin: 8px 0 10px"
                  ><span>校验失败</span></div
                >
                <el-table :data="data.validationErrors" size="small" border>
                  <el-table-column prop="param" label="参数" width="140">
                    <template #default="{ row }"
                      ><span class="mono">{{ row.param }}</span></template
                    >
                  </el-table-column>
                  <el-table-column prop="value" label="值" min-width="140">
                    <template #default="{ row }"
                      ><span class="mono">{{
                        row.value || '-'
                      }}</span></template
                    >
                  </el-table-column>
                  <el-table-column label="原因" width="170">
                    <template #default="{ row }">
                      <el-tag type="danger" size="small" effect="light">{{
                        errorReasons[row.reason] || row.reason
                      }}</el-tag>
                    </template>
                  </el-table-column>
                </el-table>
              </template>
            </section-card>

            <section-card
              title="请求信息"
              subtitle="脱敏后的客户端特征"
              class="page-section"
            >
              <el-descriptions :column="1" border class="detail-table">
                <el-descriptions-item label="识别到的参数">
                  <el-space v-if="data.matchedParams?.length" wrap :size="4">
                    <el-tag
                      v-for="p in data.matchedParams"
                      :key="p"
                      size="small"
                      effect="plain"
                      class="mono"
                      >{{ p }}</el-tag
                    >
                  </el-space>
                  <span v-else class="muted-text">无</span>
                </el-descriptions-item>
                <el-descriptions-item label="命中路径规则">{{
                  data.pathRule || '-'
                }}</el-descriptions-item>
                <el-descriptions-item v-if="data.handoffKind" label="交接来源">
                  <span class="muted-text">{{
                    data.handoffKind === 'token' ? '交接令牌' : '同设备匹配'
                  }}</span>
                  <el-link
                    v-if="data.handoffFromRequestId"
                    type="primary"
                    underline="never"
                    class="mono"
                    style="margin-left: 8px"
                    @click="load(data.handoffFromRequestId)"
                  >
                    {{ data.handoffFromRequestId }}
                  </el-link>
                </el-descriptions-item>
                <el-descriptions-item label="IP 前缀"
                  ><span class="mono">{{
                    data.ipPrefix || '-'
                  }}</span></el-descriptions-item
                >
                <el-descriptions-item label="User-Agent"
                  ><span class="ua-text">{{
                    data.uaSummary || '-'
                  }}</span></el-descriptions-item
                >
                <el-descriptions-item label="Referer"
                  ><span class="ua-text">{{
                    data.refererSummary || '-'
                  }}</span></el-descriptions-item
                >
                <el-descriptions-item label="入口 / 方法"
                  >{{ data.entry }} / {{ data.method }}</el-descriptions-item
                >
              </el-descriptions>
            </section-card>
          </el-col>
          <el-col :md="10" :xs="24">
            <section-card
              title="归因"
              subtitle="首次与最近触点"
              class="page-section"
            >
              <template v-if="data.attribution">
                <div class="side-kv"
                  ><span>首次渠道</span
                  ><em class="mono">{{
                    data.attribution.firstChannel || '-'
                  }}</em></div
                >
                <div class="side-kv"
                  ><span>首次落地</span
                  ><em class="mono">{{
                    data.attribution.firstLandingPath || '-'
                  }}</em></div
                >
                <div class="side-kv"
                  ><span>首次时间</span
                  ><em>{{ data.attribution.firstTouchAt }}</em></div
                >
                <div class="side-kv"
                  ><span>最近渠道</span
                  ><em class="mono">{{
                    data.attribution.lastChannel || '-'
                  }}</em></div
                >
                <div class="side-kv"
                  ><span>最近时间</span
                  ><em>{{ data.attribution.lastTouchAt }}</em></div
                >
                <div class="side-kv"
                  ><span>触点 / 交接</span
                  ><em
                    >{{ data.attribution.touchCount }} /
                    {{ data.attribution.handoffCount }}</em
                  ></div
                >
                <div class="side-kv"
                  ><span>归因到期</span
                  ><em>{{ data.attribution.expiresAt || '-' }}</em></div
                >
                <div class="side-kv">
                  <span>首访请求</span>
                  <el-link
                    v-if="
                      data.attribution.firstRequestId &&
                      data.attribution.firstRequestId !== data.requestId
                    "
                    type="primary"
                    underline="never"
                    class="mono"
                    @click="load(data.attribution.firstRequestId)"
                  >
                    {{ data.attribution.firstRequestId }}
                  </el-link>
                  <em v-else class="mono">{{
                    data.attribution.firstRequestId || '-'
                  }}</em>
                </div>
              </template>
              <el-empty v-else :image-size="60" description="本次访问未归因" />
            </section-card>
            <section-card
              title="访客"
              subtitle="同一站点下的累计"
              class="page-section"
            >
              <template v-if="data.visitor">
                <div class="side-kv"
                  ><span>访客 ID</span
                  ><em class="mono">{{ data.visitorId }}</em></div
                >
                <div class="side-kv"
                  ><span>首次出现</span
                  ><em>{{ data.visitor.firstSeenAt }}</em></div
                >
                <div class="side-kv"
                  ><span>最近出现</span
                  ><em>{{ data.visitor.lastSeenAt }}</em></div
                >
                <div class="side-kv"
                  ><span>访问次数</span
                  ><em>{{ data.visitor.visitCount }}</em></div
                >
                <div class="side-kv"
                  ><span>首次设备</span
                  ><em>{{ data.visitor.firstDeviceFamily || '-' }}</em></div
                >
                <div class="side-kv"
                  ><span>首次浏览器</span
                  ><em>{{
                    data.visitor.firstInApp
                      ? `内置 · ${data.visitor.firstInApp}`
                      : '系统浏览器'
                  }}</em></div
                >
              </template>
              <el-empty v-else :image-size="60" description="无访客记录" />
            </section-card>
          </el-col>
        </el-row>
      </template>
    </div>
  </el-drawer>
</template>

<script setup>
  import { ref, onMounted } from 'vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useMobile } from '@/utils/use-mobile';
  import { getDecisionLog } from '@/api/merchant/decision-log';
  import {
    contentOptions,
    touchOptions,
    confidenceOptions,
    errorReasons,
    traceSteps,
    originKindLabels,
    getOptionLabel,
    getOptionType
  } from '../constants';

  const props = defineProps({ requestId: { type: String, required: true } });
  const { modalProps } = useModal();
  const { mobile } = useMobile();
  const loading = ref(true);
  const data = ref(null);

  const load = (rid) => {
    loading.value = true;
    getDecisionLog(rid || props.requestId)
      .then((d) => (data.value = d))
      .catch((e) => {
        data.value = null;
        EleMessage.error({ message: e.message, plain: true });
      })
      .finally(() => (loading.value = false));
  };

  onMounted(() => load());
</script>

<style scoped>
  .hero-title > h3 {
    font-size: 16px;
    max-width: 360px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .side-kv > em {
    font-style: normal;
    text-align: right;
    max-width: 60%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .ua-text {
    font-size: 12px;
    word-break: break-all;
    color: var(--el-text-color-secondary);
  }
</style>
