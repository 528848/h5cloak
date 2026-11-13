<!-- API 调用详情：调用结果 + 判定参数名，每项可复制 -->
<template>
  <el-drawer
    v-bind="modalProps"
    :size="mobile ? '100%' : 560"
    title="调用详情"
    destroy-on-close
  >
    <div v-loading="loading">
      <el-empty
        v-if="!loading && !data"
        :image-size="90"
        description="日志不存在或已过期"
      />
      <template v-else-if="data">
        <div class="copy-all">
          <el-button type="primary" plain @click="copy(allText)">
            复制全部
          </el-button>
        </div>
        <section-card title="调用结果" subtitle="这次开放接口的结果">
          <div v-for="item in callItems" :key="item.label" class="kv">
            <span class="kv-label">{{ item.label }}</span>
            <span class="kv-value mono">{{ item.value || '-' }}</span>
            <el-tooltip v-if="item.value" content="复制" placement="top">
              <el-button
                link
                type="primary"
                :icon="DocumentCopy"
                @click="copy(item.value)"
              />
            </el-tooltip>
          </div>
        </section-card>
        <section-card
          title="请求参数"
          subtitle="只记录参数名，不保存点击参数和令牌明文"
          class="page-section"
        >
          <el-empty
            v-if="!hasParams"
            :image-size="64"
            description="这次调用没有关联的判定参数。鉴权失败、限流、转化回传和归因码维护只记录调用结果。"
          />
          <template v-else>
            <div v-for="item in paramItems" :key="item.label" class="kv">
              <span class="kv-label">{{ item.label }}</span>
              <span class="kv-value mono">{{ item.value || '-' }}</span>
              <el-tooltip v-if="item.value" content="复制" placement="top">
                <el-button
                  link
                  type="primary"
                  :icon="DocumentCopy"
                  @click="copy(item.value)"
                />
              </el-tooltip>
            </div>
            <div v-if="data.queryParamNames?.length" class="param-block">
              <div class="kv-label">URL 参数</div>
              <div
                v-for="name in data.queryParamNames"
                :key="'q-' + name"
                class="kv"
              >
                <span class="kv-value mono">{{ name }}</span>
                <el-tooltip content="复制" placement="top">
                  <el-button
                    link
                    type="primary"
                    :icon="DocumentCopy"
                    @click="copy(name)"
                  />
                </el-tooltip>
              </div>
            </div>
            <div v-if="data.matchedParams?.length" class="param-block">
              <div class="kv-label">命中并放行的参数</div>
              <div
                v-for="name in data.matchedParams"
                :key="'m-' + name"
                class="kv"
              >
                <span class="kv-value mono">{{ name }}</span>
                <el-tooltip content="复制" placement="top">
                  <el-button
                    link
                    type="primary"
                    :icon="DocumentCopy"
                    @click="copy(name)"
                  />
                </el-tooltip>
              </div>
            </div>
          </template>
        </section-card>
      </template>
    </div>
  </el-drawer>
</template>

<script setup>
  import { computed, ref, onMounted } from 'vue';
  import { DocumentCopy } from '@element-plus/icons-vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useMobile } from '@/utils/use-mobile';
  import { getAdminApiLog } from '@/api/admin/api-log';
  import { getMerchantApiLog } from '@/api/merchant/api-log';

  const props = defineProps({
    id: [Number, String],
    portal: { type: String, default: 'admin' }
  });

  const { modalProps } = useModal();
  const { mobile } = useMobile();
  const loading = ref(true);
  const data = ref(null);

  const text = (v) => (v == null || v === '' ? '' : String(v));

  const callItems = computed(() => {
    const d = data.value || {};
    const items = [
      { label: '请求 ID', value: text(d.requestId) },
      { label: '调用时间', value: text(d.createdAt) },
      { label: '接口', value: text(d.endpoint) },
      { label: '状态码', value: text(d.status) },
      { label: '错误码', value: text(d.errorCode) },
      { label: '耗时(ms)', value: text(d.latencyMs) },
      { label: 'IP', value: text(d.ip) },
      { label: '密钥前缀', value: text(d.keyPrefix) },
      { label: '站点', value: text(d.siteName) }
    ];
    if (props.portal === 'admin') {
      items.splice(2, 0, { label: '商户', value: text(d.merchantName) });
    }
    return items;
  });

  const paramItems = computed(() => {
    const d = data.value || {};
    return [
      { label: '方法', value: text(d.method) },
      { label: '主机', value: text(d.host) },
      { label: '路径', value: text(d.path) },
      { label: '归因码', value: text(d.code) },
      { label: '渠道', value: text(d.channel) },
      { label: '访客 ID', value: text(d.visitorId) }
    ].filter((item) => item.value);
  });

  const hasParams = computed(
    () =>
      paramItems.value.length > 0 ||
      (data.value?.queryParamNames || []).length > 0 ||
      (data.value?.matchedParams || []).length > 0
  );

  const allText = computed(() => {
    const d = data.value || {};
    const lines = callItems.value.map((item) => `${item.label}: ${item.value || '-'}`);
    lines.push(`URL 参数: ${(d.queryParamNames || []).join(', ') || '-'}`);
    lines.push(`命中参数: ${(d.matchedParams || []).join(', ') || '-'}`);
    paramItems.value.forEach((item) => lines.push(`${item.label}: ${item.value}`));
    return lines.join('\n');
  });

  const copy = async (value) => {
    const textValue = text(value);
    if (!textValue) return;
    try {
      await navigator.clipboard.writeText(textValue);
      EleMessage.success({ message: '已复制', plain: true });
    } catch (e) {
      EleMessage.error({ message: '复制失败', plain: true });
    }
  };

  onMounted(() => {
    const load = props.portal === 'merchant' ? getMerchantApiLog : getAdminApiLog;
    load(props.id)
      .then((row) => (data.value = row))
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => (loading.value = false));
  });
</script>

<style scoped>
  .copy-all {
    display: flex;
    justify-content: flex-end;
    margin-bottom: 12px;
  }

  .kv {
    display: flex;
    align-items: center;
    gap: 8px;
    min-height: 32px;
    padding: 4px 0;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .kv-label {
    width: 96px;
    flex-shrink: 0;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .kv-value {
    flex: 1;
    min-width: 0;
    word-break: break-all;
  }

  .param-block {
    margin-top: 12px;
  }
</style>
