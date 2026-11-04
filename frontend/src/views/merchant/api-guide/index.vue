<template>
  <ele-page hide-footer>
    <ele-card
      class="detail-hero page-section"
      :body-style="{ padding: '18px 20px' }"
    >
      <div class="hero-head">
        <div>
          <div class="hero-title">
            <h3>服务端接入指南</h3>
            <el-tag type="success" effect="light">v2 签名接口</el-tag>
            <el-tag type="info" effect="plain">兼容 V1 X-API-Key</el-tag>
          </div>
          <div class="hero-desc">
            不托管域名也能用：你的服务端在渲染页面前调用判定接口，按返回决定输出真实内容还是审核内容，地址栏保持不变。
          </div>
        </div>
        <el-space>
          <el-select
            v-model="siteId"
            placeholder="选择站点"
            style="width: 220px"
          >
            <el-option
              v-for="s in sites"
              :key="s.siteId"
              :label="s.name"
              :value="s.siteId"
            />
          </el-select>
          <el-select v-model="lang" style="width: 130px">
            <el-option
              v-for="l in languages"
              :key="l.key"
              :label="l.label"
              :value="l.key"
            />
          </el-select>
        </el-space>
      </div>
      <div class="summary-strip">
        <div class="summary-item is-amount">
          <span>网关地址</span>
          <strong class="mono">{{ baseUrl }}</strong>
        </div>
        <div class="summary-item">
          <span>密钥前缀 X-Cloak-Key</span>
          <strong class="mono">{{ site?.secretPrefix || '-' }}</strong>
        </div>
        <div class="summary-item">
          <span>Cookie 名</span>
          <strong class="mono">{{ site?.cookieName || '_ck_vid' }}</strong>
        </div>
        <div class="summary-item">
          <span>IP 白名单</span>
          <strong>{{
            site?.secretIpAllowlist?.length
              ? site.secretIpAllowlist.join(', ')
              : '不限制'
          }}</strong>
        </div>
      </div>
    </ele-card>

    <el-alert
      v-if="site && !site.secretPrefix"
      type="warning"
      show-icon
      :closable="false"
      class="page-section"
      title="该站点尚无服务端密钥，请到站点配置页创建或重置密钥。"
    />

    <el-row :gutter="16">
      <el-col :lg="7" :xs="24">
        <section-card title="接入步骤" subtitle="三步完成" class="page-section">
          <el-steps
            direction="vertical"
            :active="3"
            finish-status="success"
            class="steps"
          >
            <el-step
              title="拿到密钥"
              description="站点配置页创建站点或「重置服务端密钥」，完整密钥 sk_... 只显示一次，放到服务端环境变量。"
            />
            <el-step
              title="实现签名"
              description="复制右侧「签名函数」，所有接口共用。路径不含查询串，GET 请求体按空字符串计算哈希。"
            />
            <el-step
              title="按场景调用"
              description="页面判定、只做归因、下单查归因、支付回传。接口超时建议 3 秒，失败按站点兜底处理。"
            />
          </el-steps>
        </section-card>
        <section-card
          title="错误码"
          subtitle="data.error 字段"
          class="page-section"
        >
          <div v-for="[code, desc] in errorCodes" :key="code" class="side-kv">
            <span class="mono">{{ code }}</span>
            <em>{{ desc }}</em>
          </div>
        </section-card>
        <section-card title="文档" subtitle="接口定义" class="page-section">
          <div class="side-kv"
            ><span>OpenAPI</span
            ><em class="mono">backend-go/docs/openapi/openapi.yaml</em></div
          >
          <div class="side-kv"
            ><span>探活</span
            ><em class="mono">GET {{ baseUrl }}/api/open/v2/ping</em></div
          >
          <div class="side-kv"
            ><span>限流</span><em>每密钥 50 次/秒 · 每站点 2000 次/分</em></div
          >
        </section-card>
      </el-col>

      <el-col :lg="17" :xs="24">
        <section-card
          title="签名函数"
          subtitle="所有 v2 接口共用，密钥只存服务端"
          class="page-section"
        >
          <template #extra>
            <el-button
              link
              type="primary"
              @click="copy(render(signSamples[lang]))"
              >复制</el-button
            >
          </template>
          <pre class="code-block mono">{{ render(signSamples[lang]) }}</pre>
        </section-card>

        <section-card
          v-for="s in scenarios"
          :key="s.key"
          :title="s.title"
          :subtitle="s.endpoint"
          class="page-section"
        >
          <template #extra>
            <el-button link type="primary" @click="copy(render(s.code[lang]))"
              >复制</el-button
            >
          </template>
          <p class="scene-desc">{{ s.desc }}</p>
          <pre class="code-block mono">{{ render(s.code[lang]) }}</pre>
        </section-card>

        <section-card
          title="V1 兼容调用"
          subtitle="旧插件无需改动；新接入请用 v2 签名"
          class="page-section"
        >
          <pre class="code-block mono">{{ render(v1Sample) }}</pre>
        </section-card>

        <section-card
          title="在线验证"
          subtitle="用当前站点发一次签名请求（服务端代签，密钥不会下发到浏览器）"
          class="page-section"
        >
          <el-alert
            type="info"
            show-icon
            :closable="false"
            style="margin-bottom: 12px"
          >
            这里调用的是后台「判定测试」接口，与开放接口共用同一引擎；真正的签名调用请在你的服务端完成后用
            <code>/api/open/v2/ping</code> 与 <code>decision</code> 验证。
          </el-alert>
          <el-button type="primary" plain @click="goTest"
            >打开判定测试</el-button
          >
        </section-card>
      </el-col>
    </el-row>
  </ele-page>
</template>

<script setup>
  import { ref, computed, watch, onMounted } from 'vue';
  import { useRouter } from 'vue-router';
  import { EleMessage } from 'ele-admin-plus';
  import { useCurrentSite } from '@/utils/use-site';
  import { getSite, getSiteGuide } from '@/api/merchant/site';
  import {
    languages,
    signSamples,
    scenarios,
    v1Sample,
    errorCodes
  } from './samples';

  defineOptions({ name: 'MerchantApiGuide' });

  const router = useRouter();
  const { sites, siteId, load: loadSites } = useCurrentSite();
  const lang = ref('php');
  const site = ref(null);
  const entryHost = ref('');

  const baseUrl = computed(() =>
    entryHost.value ? `https://${entryHost.value}` : 'https://<网关地址>'
  );
  const hostName = computed(() => {
    const d = site.value?.domains?.find?.((x) => x.status === 'bound');
    return d?.domain || 'shop.example.com';
  });

  const render = (tpl) =>
    String(tpl || '')
      .replaceAll('{{BASE}}', baseUrl.value)
      .replaceAll('{{PREFIX}}', site.value?.secretPrefix || 'sk_xxxxxxxxx')
      .replaceAll('{{HOST}}', hostName.value);

  const loadSite = () => {
    if (!siteId.value) {
      site.value = null;
      return;
    }
    getSite(siteId.value)
      .then((d) => (site.value = d))
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));
  };

  const copy = async (text) => {
    try {
      await navigator.clipboard.writeText(text);
      EleMessage.success({ message: '已复制', plain: true });
    } catch (e) {
      EleMessage.error({ message: '复制失败，请手动选择', plain: true });
    }
  };

  const goTest = () =>
    router.push({
      path: '/merchant/decision-test',
      query: { siteId: siteId.value }
    });

  watch(siteId, loadSite);
  onMounted(() => {
    getSiteGuide()
      .then((d) => (entryHost.value = d.entryHost))
      .catch(() => {});
    loadSites().then(loadSite);
  });
</script>

<style scoped>
  .code-block {
    margin: 0;
    padding: 14px 16px;
    border-radius: 10px;
    background: var(--el-fill-color-light);
    font-size: 12px;
    line-height: 1.65;
    overflow: auto;
    white-space: pre;
  }

  .scene-desc {
    margin: 0 0 12px;
    font-size: 13px;
    line-height: 1.7;
    color: var(--el-text-color-secondary);
  }

  .steps :deep(.el-step__description) {
    font-size: 12px;
    line-height: 1.6;
    padding-bottom: 14px;
  }

  .side-kv > em {
    font-style: normal;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-align: right;
    max-width: 62%;
  }
</style>
