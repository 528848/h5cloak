<template>
  <ele-page hide-footer>
    <el-row :gutter="16">
      <el-col :lg="14" :xs="24">
        <section-card
          title="链接参数"
          subtitle="选择站点与域名，填上归因码和来源参数，生成可直接投放的链接"
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
            <el-form-item label="域名" prop="baseUrl">
              <el-select
                v-model="form.baseUrl"
                filterable
                allow-create
                default-first-option
                placeholder="选择已绑定域名，或输入地址"
                class="ele-fluid"
              >
                <el-option
                  v-for="d in options.domains"
                  :key="d.domain"
                  :label="`${d.domain}${d.status === 'bound' ? '' : '（未绑定）'}`"
                  :value="`https://${d.domain}`"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="落地路径">
              <el-input
                v-model.trim="form.path"
                placeholder="/ 或 /landing"
                class="mono"
              />
            </el-form-item>
            <el-form-item :label="codeLabel">
              <el-input
                v-model.trim="form.code"
                placeholder="已登记的归因码（分销码 / 活动码），留空则不带码"
                class="mono"
              />
              <div v-if="options.codeParam" class="field-hint">
                分销码参数名 <span class="mono">{{ options.codeParam }}</span>
                <template v-if="options.campaignParam">
                  ，活动码参数名
                  <span class="mono">{{ options.campaignParam }}</span>
                </template>
              </div>
              <div v-else class="field-hint">
                站点未启用分销码参数，请先在「归因规则」配置
              </div>
            </el-form-item>
            <el-divider content-position="left">来源参数</el-divider>
            <el-row :gutter="12">
              <el-col v-for="p in sourceParams" :key="p.name" :span="12">
                <el-form-item :label="p.name" label-width="110px">
                  <el-input
                    v-model.trim="form.params[p.name]"
                    :placeholder="p.placeholder"
                  />
                </el-form-item>
              </el-col>
            </el-row>
            <el-divider content-position="left">附加参数</el-divider>
            <div v-for="(row, i) in extras" :key="i" class="extra-row">
              <el-input
                v-model.trim="row.key"
                placeholder="参数名"
                class="mono"
              />
              <el-input v-model.trim="row.value" placeholder="参数值" />
              <el-button link type="danger" @click="extras.splice(i, 1)"
                >删除</el-button
              >
            </div>
            <el-button
              plain
              size="small"
              @click="extras.push({ key: '', value: '' })"
            >
              添加参数
            </el-button>
          </el-form>
          <template #footer>
            <div class="form-bar">
              <el-button plain @click="handleReset">重置</el-button>
              <el-button
                type="primary"
                :loading="building"
                @click="handleBuild"
              >
                生成链接
              </el-button>
            </div>
          </template>
        </section-card>
      </el-col>
      <el-col :lg="10" :xs="24">
        <section-card
          title="生成结果"
          subtitle="复制后投放到对应渠道"
          class="page-section"
        >
          <template v-if="result">
            <div class="result-box mono">{{ result.url }}</div>
            <el-alert
              v-for="(w, i) in result.warnings"
              :key="i"
              type="warning"
              show-icon
              :closable="false"
              :title="w"
              style="margin-top: 10px"
            />
            <div class="result-actions">
              <el-button type="primary" plain @click="copy(result.url)"
                >复制链接</el-button
              >
              <el-link
                :href="result.url"
                target="_blank"
                type="primary"
                underline="never"
              >
                在新窗口打开
              </el-link>
            </div>
          </template>
          <el-empty
            v-else
            description="填写左侧参数后点击「生成链接」"
            :image-size="80"
          />
        </section-card>
        <section-card
          title="说明"
          subtitle="链接如何被归因"
          class="page-section"
        >
          <div class="side-kv"
            ><span>归因码</span><em>须在归因码列表中登记且状态有效</em></div
          >
          <div class="side-kv"
            ><span>点击参数</span
            ><em>由广告平台在跳转时自动追加，无需手填</em></div
          >
          <div class="side-kv"
            ><span>来源参数</span
            ><em>utm_source 若配置了允许来源列表，须在列表内</em></div
          >
          <div class="side-kv"
            ><span>地址栏</span
            ><em>访客看到的链接保持不变，只是返回的内容不同</em></div
          >
        </section-card>
      </el-col>
    </el-row>
  </ele-page>
</template>

<script setup>
  import { ref, reactive, computed, watch, onMounted } from 'vue';
  import { useRoute } from 'vue-router';
  import { EleMessage } from 'ele-admin-plus';
  import { useCurrentSite } from '@/utils/use-site';
  import { getLinkOptions, buildLink } from '@/api/merchant/link';

  defineOptions({ name: 'MerchantLink' });

  const route = useRoute();
  const { sites, siteId, load: loadSites } = useCurrentSite();
  const formRef = ref(null);
  const building = ref(false);
  const result = ref(null);
  const options = ref({
    domains: [],
    codeParam: '',
    campaignParam: '',
    sourceParams: []
  });
  const extras = ref([]);

  const placeholders = {
    utm_source: '流量来源，如 tiktok',
    utm_medium: '媒介，如 cpc、social',
    utm_campaign: '活动名',
    utm_content: '素材',
    utm_term: '关键词'
  };

  const form = reactive({
    siteId: Number(route.query.siteId) || siteId.value || undefined,
    baseUrl: '',
    path: '/',
    code: String(route.query.code || ''),
    params: {}
  });

  const rules = {
    siteId: [{ required: true, message: '请选择站点', trigger: 'change' }],
    baseUrl: [
      { required: true, message: '请选择域名或填写地址', trigger: 'change' }
    ]
  };

  const codeLabel = computed(() => '归因码');
  const sourceParams = computed(() =>
    (options.value.sourceParams || []).map((p) => ({
      name: p.name,
      placeholder: placeholders[p.name] || ''
    }))
  );

  const loadOptions = () => {
    if (!form.siteId) return;
    getLinkOptions(form.siteId)
      .then((d) => {
        options.value = d;
        const bound =
          (d.domains || []).find((x) => x.status === 'bound') || d.domains?.[0];
        if (!form.baseUrl && bound) form.baseUrl = `https://${bound.domain}`;
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));
  };

  const handleBuild = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) return;
      const params = { ...form.params };
      extras.value.forEach((r) => {
        if (r.key && r.value) params[r.key] = r.value;
      });
      building.value = true;
      buildLink({
        siteId: form.siteId,
        baseUrl: form.baseUrl,
        path: form.path,
        code: form.code,
        params
      })
        .then((d) => (result.value = d))
        .catch((e) => EleMessage.error({ message: e.message, plain: true }))
        .finally(() => (building.value = false));
    });
  };

  const handleReset = () => {
    form.path = '/';
    form.code = '';
    form.params = {};
    extras.value = [];
    result.value = null;
  };

  const copy = async (text) => {
    try {
      await navigator.clipboard.writeText(text);
      EleMessage.success({ message: '已复制', plain: true });
    } catch (e) {
      EleMessage.error({ message: '复制失败，请手动选择', plain: true });
    }
  };

  watch(
    () => form.siteId,
    (v) => {
      if (v) siteId.value = v;
      form.baseUrl = '';
      result.value = null;
      loadOptions();
    }
  );

  onMounted(() => {
    loadSites().then(() => {
      if (!form.siteId) form.siteId = siteId.value || undefined;
      loadOptions();
    });
  });
</script>

<style scoped>
  .field-hint {
    width: 100%;
    margin-top: 4px;
    font-size: 12px;
    line-height: 18px;
    color: var(--el-text-color-secondary);
  }

  .extra-row {
    display: grid;
    grid-template-columns: 1fr 1.4fr auto;
    gap: 8px;
    margin-bottom: 8px;
  }

  .form-bar {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  .result-box {
    padding: 12px;
    border-radius: 8px;
    background: var(--el-fill-color-light);
    word-break: break-all;
    line-height: 1.6;
  }

  .result-actions {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-top: 12px;
  }

  .side-kv > em {
    font-style: normal;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-align: right;
  }
</style>
