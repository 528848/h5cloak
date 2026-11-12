<template>
  <ele-page hide-footer>
    <ele-card
      v-loading="loading"
      class="detail-hero page-section"
      :body-style="{ padding: '18px 20px' }"
    >
      <div class="hero-head">
        <div>
          <div class="hero-title">
            <h3>{{ detail?.name || '新建站点' }}</h3>
            <el-tag
              v-if="detail"
              :type="getOptionType(siteStatusOptions, detail.status)"
              effect="light"
            >
              {{ getOptionLabel(siteStatusOptions, detail.status) }}
            </el-tag>
            <el-tag
              v-if="detail"
              :type="getOptionType(siteTypeOptions, detail.siteType)"
              effect="plain"
            >
              {{ getOptionLabel(siteTypeOptions, detail.siteType) }}
            </el-tag>
          </div>
          <div class="hero-desc">
            <template v-if="detail">
              站点 ID {{ detail.siteId }} · 创建于 {{ detail.createdAt }} ·
              同一访问地址按归因结果返回真实内容或审核内容
            </template>
            <template v-else>
              先填写基本设置并创建，再配置源站、审核内容和域名
            </template>
          </div>
        </div>
        <el-space>
          <el-button plain @click="goBack">返回列表</el-button>
          <el-button
            v-if="canSave"
            type="primary"
            :loading="saving"
            @click="handleSave"
          >
            {{ siteId ? '保存设置' : '创建站点' }}
          </el-button>
        </el-space>
      </div>
      <div v-if="detail" class="summary-strip">
        <div class="summary-item is-amount">
          <span>公钥</span>
          <strong class="mono">{{ detail.publishableKey }}</strong>
        </div>
        <div class="summary-item">
          <span>服务端密钥</span>
          <strong class="mono">{{ detail.secretPrefix }}***</strong>
        </div>
        <div class="summary-item">
          <span>归因窗口</span>
          <strong>{{ detail.attributionTtlDays }} 天</strong>
        </div>
        <div class="summary-item">
          <span>无归因时</span>
          <strong>{{
            getOptionLabel(noAttributionOptions, detail.noAttributionAction)
          }}</strong>
        </div>
      </div>
    </ele-card>

    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="120px"
      :disabled="!canSave"
      @submit.prevent=""
    >
      <section-card
        title="基本信息"
        subtitle="名称、接入方式与备注"
        class="page-section"
      >
        <el-row :gutter="16">
          <el-col :md="12" :xs="24">
            <el-form-item label="站点名称" prop="name">
              <el-input
                v-model.trim="form.name"
                maxlength="64"
                placeholder="1-64 字"
              />
            </el-form-item>
          </el-col>
          <el-col :md="12" :xs="24">
            <el-form-item label="接入方式" prop="siteType">
              <el-select v-model="form.siteType" class="ele-fluid">
                <el-option
                  v-for="o in siteTypeOptions"
                  :key="o.value"
                  :label="o.label"
                  :value="o.value"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="备注">
              <el-input
                v-model="form.remark"
                type="textarea"
                :rows="2"
                maxlength="200"
                placeholder="请输入备注"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </section-card>

      <section-card
        title="Cookie 与归因窗口"
        subtitle="访客再次打开同一地址时如何认出他"
        class="page-section"
      >
        <el-row :gutter="16">
          <el-col :md="12" :xs="24">
            <el-form-item label="Cookie 名">
              <el-input v-model.trim="form.cookieName" placeholder="_ck_vid" />
            </el-form-item>
          </el-col>
          <el-col :md="12" :xs="24">
            <el-form-item label="Cookie 域">
              <el-input
                v-model.trim="form.cookieDomain"
                placeholder="留空则使用当前访问域名"
              />
            </el-form-item>
          </el-col>
          <el-col :md="12" :xs="24">
            <el-form-item label="归因窗口">
              <el-input-number
                v-model="form.attributionTtlDays"
                :min="1"
                :max="365"
                controls-position="right"
              />
              <span class="field-hint">天，最长 365</span>
            </el-form-item>
          </el-col>
          <el-col :md="12" :xs="24">
            <el-form-item label="续期方式">
              <el-radio-group v-model="form.renewMode">
                <el-radio
                  v-for="o in renewModeOptions"
                  :key="o.value"
                  :value="o.value"
                >
                  {{ o.label }}
                </el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
        </el-row>
      </section-card>

      <section-card
        title="无归因与失败兜底"
        subtitle="没有归因参数，或判定服务异常时返回什么"
        class="page-section"
      >
        <el-row :gutter="16">
          <el-col :md="12" :xs="24">
            <el-form-item label="无归因时">
              <el-select v-model="form.noAttributionAction" class="ele-fluid">
                <el-option
                  v-for="o in noAttributionOptions"
                  :key="o.value"
                  :label="o.label"
                  :value="o.value"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :md="12" :xs="24">
            <el-form-item label="失败兜底">
              <el-select v-model="form.failMode" class="ele-fluid">
                <el-option
                  v-for="o in failModeOptions"
                  :key="o.value"
                  :label="o.label"
                  :value="o.value"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :md="12" :xs="24">
            <el-form-item label="最低置信度">
              <el-select v-model="form.minConfidence" class="ele-fluid">
                <el-option
                  v-for="o in confidenceOptions"
                  :key="o.value"
                  :label="o.label"
                  :value="o.value"
                />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
      </section-card>

      <section-card
        title="跨浏览器接力"
        subtitle="从社媒内置浏览器换到系统浏览器后，如何接上同一次访问"
        class="page-section"
      >
        <el-row :gutter="16">
          <el-col :md="12" :xs="24">
            <el-form-item label="打开提示条">
              <el-switch v-model="form.handoffBanner" />
            </el-form-item>
          </el-col>
          <el-col :md="12" :xs="24">
            <el-form-item label="改写外链">
              <el-switch v-model="form.handoffLinkRewrite" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="提示文案">
              <el-input
                v-model.trim="form.handoffBannerText"
                maxlength="80"
                placeholder="在浏览器中打开，继续访问"
              />
            </el-form-item>
          </el-col>
          <el-col v-if="form.handoffBanner" :span="24">
            <el-form-item label="效果">
              <el-button plain @click="bannerPreview = true">预览提示条</el-button>
              <span class="field-hint">浮层弹框，没有遮罩，访客仍可操作源站</span>
            </el-form-item>
          </el-col>
          <el-col :md="8" :xs="24">
            <el-form-item label="接力有效期">
              <el-input-number
                v-model="form.handoffTokenTtlHours"
                :min="1"
                :max="168"
                controls-position="right"
              />
              <span class="field-hint">小时</span>
            </el-form-item>
          </el-col>
          <el-col :md="8" :xs="24">
            <el-form-item label="最多使用">
              <el-input-number
                v-model="form.handoffTokenMaxUses"
                :min="1"
                :max="20"
                controls-position="right"
              />
              <span class="field-hint">次</span>
            </el-form-item>
          </el-col>
          <el-col :md="8" :xs="24">
            <el-form-item label="模糊匹配">
              <el-switch v-model="form.handoffMatchEnabled" />
            </el-form-item>
          </el-col>
          <el-col v-if="form.handoffMatchEnabled" :md="12" :xs="24">
            <el-form-item label="匹配时间窗">
              <el-input-number
                v-model="form.handoffMatchMinutes"
                :min="5"
                :max="60"
                controls-position="right"
              />
              <span class="field-hint"
                >分钟，5-60。按 IP 与设备族做概率匹配</span
              >
            </el-form-item>
          </el-col>
          <el-col v-if="form.handoffMatchEnabled" :md="12" :xs="24">
            <el-form-item label="只匹配系统浏览器">
              <el-switch v-model="form.handoffMatchSysOnly" />
              <span class="field-hint"
                >开启时新请求须来自 Safari/Chrome 等系统浏览器，避免同一 App
                内两位访客互相匹配</span
              >
            </el-form-item>
          </el-col>
        </el-row>
      </section-card>

      <section-card
        title="密钥"
        subtitle="公钥可公开，服务端密钥只在创建或重置时出现一次"
        class="page-section"
      >
        <el-form-item label="IP 白名单">
          <el-input
            v-model="form.secretIpAllowlistText"
            type="textarea"
            :rows="3"
            placeholder="每行一个 IP 或 CIDR，留空表示不限制。例如 10.0.0.0/8"
          />
        </el-form-item>
        <el-alert
          v-if="detail"
          type="info"
          show-icon
          :closable="false"
          :title="`当前密钥前缀 ${detail.secretPrefix}，完整密钥无法再次查看`"
        />
      </section-card>
    </el-form>

    <template v-if="detail">
      <section-card
        title="源站"
        subtitle="真实内容与审核内容的回源地址，地址栏保持访客原来的链接"
        class="page-section"
      >
        <el-row :gutter="16">
          <el-col
            v-for="item in originItems"
            :key="item.kind"
            :md="12"
            :xs="24"
          >
            <div class="origin-box">
              <div class="origin-head">
                <strong>{{ item.label }}</strong>
                <el-tag
                  :type="
                    getOptionType(
                      healthOptions,
                      origins[item.kind].healthStatus
                    )
                  "
                  effect="light"
                  size="small"
                >
                  {{
                    getOptionLabel(
                      healthOptions,
                      origins[item.kind].healthStatus
                    )
                  }}
                </el-tag>
              </div>
              <el-form label-width="88px" :disabled="!canEdit">
                <el-form-item label="源站地址">
                  <el-input
                    v-model.trim="origins[item.kind].url"
                    placeholder="https://origin.example.com"
                  />
                </el-form-item>
                <el-form-item label="回源 Host">
                  <el-input
                    v-model.trim="origins[item.kind].hostHeader"
                    placeholder="填客户主域名可沿用已有证书，如 example.com"
                  />
                </el-form-item>
                <el-form-item label="超时">
                  <el-input-number
                    v-model="origins[item.kind].timeoutMs"
                    :min="1000"
                    :max="60000"
                    :step="1000"
                    controls-position="right"
                  />
                  <span class="field-hint">毫秒</span>
                </el-form-item>
              </el-form>
              <div class="origin-actions">
                <el-button
                  v-if="canEdit"
                  type="primary"
                  plain
                  @click="handleSaveOrigin(item.kind)"
                >
                  保存
                </el-button>
                <el-button v-if="canEdit" plain @click="handleProbe(item.kind)">
                  检测
                </el-button>
                <span class="muted-text">{{
                  origins[item.kind].healthMessage
                }}</span>
              </div>
            </div>
          </el-col>
        </el-row>
      </section-card>

      <section-card
        title="审核内容"
        subtitle="没有归因时，同一链接返回这里配置的内容"
        class="page-section"
      >
        <el-radio-group v-model="review.source" :disabled="!canEdit">
          <el-radio
            v-for="o in reviewSourceOptions"
            :key="o.value"
            :value="o.value"
          >
            {{ o.label }}
          </el-radio>
        </el-radio-group>
        <div v-if="review.source === 'template'" class="block-gap">
          <el-form label-width="88px" :disabled="!canEdit">
            <el-form-item label="模板">
              <el-select
                v-model="review.templateId"
                class="ele-fluid"
                placeholder="请选择模板"
              >
                <el-option
                  v-for="t in templates"
                  :key="t.templateId"
                  :label="`${t.name}（${templateKindLabel(t.kind)}）`"
                  :value="t.templateId"
                />
              </el-select>
            </el-form-item>
            <template v-if="selectedTemplateKind === 'html'">
              <el-form-item label="站点名">
                <el-input
                  v-model.trim="review.siteName"
                  placeholder="展示在模板标题"
                />
              </el-form-item>
              <el-form-item label="口号">
                <el-input
                  v-model.trim="review.slogan"
                  placeholder="展示在模板副标题"
                />
              </el-form-item>
            </template>
            <el-alert
              v-else-if="selectedTemplateKind"
              type="info"
              show-icon
              :closable="false"
              title="内容由平台模板提供，站点不能修改模板正文。"
            />
          </el-form>
          <el-button
            v-if="canEdit && selectedTemplateKind === 'html'"
            plain
            @click="handlePreview"
            >预览</el-button
          >
        </div>
        <div v-else-if="review.source === 'origin'" class="block-gap">
          <el-alert
            type="info"
            show-icon
            :closable="false"
            title="将回源到上方配置的审核源站。请先保存审核源站地址。"
          />
        </div>
        <div v-else class="block-gap">
          <el-upload
            :auto-upload="false"
            :limit="1"
            accept=".zip"
            :on-change="onPackage"
            :on-remove="() => (packageFile = null)"
          >
            <el-button :disabled="!canEdit">选择 zip 静态包</el-button>
          </el-upload>
          <p class="muted-text"
            >小于 20MB，包内需要包含 index.html。上传后即作为审核内容。</p
          >
          <el-button
            v-if="canEdit"
            type="primary"
            plain
            :disabled="!packageFile"
            :loading="uploading"
            @click="handleUpload"
          >
            上传
          </el-button>
          <el-link
            v-if="previewUrl"
            type="primary"
            :href="previewUrl"
            target="_blank"
            underline="never"
          >
            打开静态包预览
          </el-link>
        </div>
        <div class="origin-actions">
          <el-button
            v-if="canEdit"
            type="primary"
            plain
            @click="handleSaveReview"
          >
            保存审核内容
          </el-button>
        </div>
      </section-card>

      <section-card
        title="路径规则"
        subtitle="命中后跳过归因，或强制返回真实内容、审核内容"
        class="page-section"
      >
        <el-table :data="pathRules" border size="small">
          <el-table-column label="路径" min-width="160">
            <template #default="{ row }">
              <el-input v-model.trim="row.pattern" :disabled="!canEdit" />
            </template>
          </el-table-column>
          <el-table-column label="匹配" width="120">
            <template #default="{ row }">
              <el-select v-model="row.matchType" :disabled="!canEdit">
                <el-option
                  v-for="o in matchTypeOptions"
                  :key="o.value"
                  :label="o.label"
                  :value="o.value"
                />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="动作" width="140">
            <template #default="{ row }">
              <el-select v-model="row.action" :disabled="!canEdit">
                <el-option
                  v-for="o in pathActionOptions"
                  :key="o.value"
                  :label="o.label"
                  :value="o.value"
                />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="排序" width="90">
            <template #default="{ row }">
              <el-input-number
                v-model="row.sort"
                :min="0"
                :controls="false"
                :disabled="!canEdit"
                class="ele-fluid"
              />
            </template>
          </el-table-column>
          <el-table-column label="备注" min-width="120">
            <template #default="{ row }">
              <el-input v-model="row.remark" :disabled="!canEdit" />
            </template>
          </el-table-column>
          <el-table-column label="操作" width="70" align="center">
            <template #default="{ $index }">
              <el-button
                v-if="canEdit"
                link
                type="danger"
                @click="pathRules.splice($index, 1)"
              >
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="origin-actions">
          <el-button v-if="canEdit" plain @click="addRule">添加规则</el-button>
          <el-button
            v-if="canEdit"
            type="primary"
            plain
            @click="handleSaveRules"
          >
            保存路径规则
          </el-button>
        </div>
      </section-card>

      <section-card
        title="域名"
        subtitle="托管域名需按解析指引指向网关，自有域名只校验归属"
        class="page-section"
      >
        <template #extra>
          <el-button
            v-permission="'site:domain:add'"
            link
            type="primary"
            @click="handleAddDomain"
          >
            添加域名
          </el-button>
        </template>
        <el-table :data="domains" border size="small">
          <el-table-column prop="domain" label="域名" min-width="160" />
          <el-table-column label="类型" width="100">
            <template #default="{ row }">
              {{ getOptionLabel(domainTypeOptions, row.domainType) }}
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag
                :type="getOptionType(domainStatusOptions, row.status)"
                effect="light"
                size="small"
              >
                {{ getOptionLabel(domainStatusOptions, row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column
            prop="verifyMessage"
            label="校验结果"
            min-width="180"
          />
          <el-table-column label="操作" width="140" align="center">
            <template #default="{ row }">
              <el-button link type="primary" @click="openGuide(row)"
                >指引</el-button
              >
              <el-button
                v-permission="'site:domain:verify'"
                link
                type="primary"
                @click="handleVerify(row)"
              >
                校验
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </section-card>

      <section-card
        title="密钥操作"
        subtitle="服务端密钥用于接口签名；访客令牌密钥用于签发 Cookie"
        class="page-section"
      >
        <el-row :gutter="16">
          <el-col :md="12" :xs="24">
            <div class="key-box">
              <div class="key-head">
                <strong>服务端密钥</strong>
                <span class="muted-text">前缀 {{ detail.secretPrefix }}</span>
              </div>
              <p class="muted-text"
                >重置后旧密钥立即失效，使用旧密钥的服务端判定会失败。</p
              >
              <el-button
                v-permission="'site:site:secret'"
                type="danger"
                plain
                @click="handleResetSecret"
              >
                重置服务端密钥
              </el-button>
            </div>
          </el-col>
          <el-col :md="12" :xs="24">
            <div class="key-box">
              <div class="key-head">
                <strong>访客令牌密钥</strong>
                <span class="muted-text"
                  >当前版本 v{{ detail.tokenKeyVersion || '-' }}</span
                >
              </div>
              <p class="muted-text">
                轮换后新访客用新密钥；旧 Cookie
                在宽限期内仍可放行并自动换发。宽限期为 0
                等于强制全部访客重新归因。
              </p>
              <el-space>
                <el-button
                  v-permission="'site:site:secret'"
                  type="warning"
                  plain
                  @click="handleRotateKey(72)"
                >
                  轮换（宽限 72 小时）
                </el-button>
                <el-button
                  v-permission="'site:site:secret'"
                  type="danger"
                  plain
                  @click="handleRotateKey(0)"
                >
                  强制全部重新归因
                </el-button>
              </el-space>
            </div>
          </el-col>
        </el-row>
      </section-card>
    </template>

    <ele-card>
      <div class="form-bar">
        <el-button plain @click="goBack">返回列表</el-button>
        <el-button
          v-if="canSave"
          type="primary"
          :loading="saving"
          @click="handleSave"
        >
          {{ siteId ? '保存设置' : '创建站点' }}
        </el-button>
      </div>
    </ele-card>

    <el-dialog
      v-model="secretVisible"
      title="请立即保存服务端密钥"
      width="520px"
      :close-on-click-modal="false"
    >
      <el-alert
        type="warning"
        show-icon
        :closable="false"
        title="密钥只显示这一次。关闭后只能看到前缀，无法再查看完整密钥。"
        style="margin-bottom: 12px"
      />
      <div class="secret-box mono">{{ serverSecret }}</div>
      <template #footer>
        <el-button @click="copySecret">复制</el-button>
        <el-button type="primary" @click="secretVisible = false"
          >我已保存</el-button
        >
      </template>
    </el-dialog>

    <el-dialog v-model="bannerPreview" title="提示条预览" width="640px">
      <div class="banner-stage">
        <div class="banner-page">
          <div class="banner-line"></div>
          <div class="banner-line short"></div>
          <div class="banner-line"></div>
          <div class="banner-line mid"></div>
        </div>
        <div class="ck-preview">
          <button type="button" class="ck-x" @click="bannerPreview = false">×</button>
          <div class="ck-text">{{ form.handoffBannerText || '在浏览器中打开，继续访问' }}</div>
          <span class="ck-open">打开</span>
        </div>
      </div>
      <p class="banner-note">弹框浮在源站上，没有遮罩。访客可以关掉它，也可以继续点击页面里的内容。</p>
    </el-dialog>

    <el-dialog v-model="previewVisible" title="模板预览" width="860px">
      <iframe class="preview-frame" :srcdoc="previewHtml"></iframe>
    </el-dialog>
  </ele-page>
</template>

<script setup>
  import { ref, reactive, computed, watch } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { usePermission } from '@/utils/use-permission';
  import {
    getSite,
    addSite,
    updateSite,
    resetSiteSecret,
    saveSiteOrigin,
    probeSiteOrigin,
    saveSiteReview,
    listReviewTemplates,
    previewReviewTemplate,
    uploadSitePackage,
    saveSitePathRules
  } from '@/api/merchant/site';
  import {
    pageMerchantDomains,
    verifyMerchantDomain
  } from '@/api/merchant/domain';
  import { rotateSiteKey } from '@/api/merchant/decision';
  import {
    siteStatusOptions,
    siteTypeOptions,
    domainStatusOptions,
    domainTypeOptions,
    healthOptions,
    renewModeOptions,
    noAttributionOptions,
    failModeOptions,
    confidenceOptions,
    reviewSourceOptions,
    matchTypeOptions,
    pathActionOptions,
    getOptionLabel,
    getOptionType,
    emptySiteForm,
    siteToPayload
  } from '../constants';

  defineOptions({ name: 'MerchantSiteForm' });

  const route = useRoute();
  const router = useRouter();
  const { openModal } = useModal();
  const { hasPermission } = usePermission();
  const siteId = computed(() => Number(route.query.id) || 0);
  const canEdit = computed(() => hasPermission('site:site:edit'));
  const canSave = computed(() =>
    siteId.value ? canEdit.value : hasPermission('site:site:add')
  );

  const loading = ref(false);
  const saving = ref(false);
  const uploading = ref(false);
  const detail = ref(null);
  const formRef = ref(null);
  const form = reactive(emptySiteForm());
  const rules = {
    name: [{ required: true, message: '请输入站点名称', trigger: 'blur' }]
  };
  const templates = ref([]);
  const domains = ref([]);
  const pathRules = ref([]);
  const packageFile = ref(null);
  const previewUrl = ref('');
  const previewHtml = ref('');
  const previewVisible = ref(false);
  const secretVisible = ref(false);
  const bannerPreview = ref(false);
  const serverSecret = ref('');
  const review = reactive({
    source: 'template',
    templateId: undefined,
    siteName: '',
    slogan: ''
  });
  const templateKindLabel = (kind) =>
    ({ html: 'HTML', url: '网址', static: '静态项目' })[kind] || 'HTML';
  const selectedTemplateKind = computed(() => {
    const row = templates.value.find((t) => t.templateId === review.templateId);
    return row?.kind || 'html';
  });
  const origins = reactive({
    real: emptyOrigin(),
    review: emptyOrigin()
  });
  const originItems = [
    { kind: 'real', label: '真实源站' },
    { kind: 'review', label: '审核源站' }
  ];

  function emptyOrigin() {
    return {
      url: '',
      hostHeader: '',
      timeoutMs: 15000,
      healthStatus: 'unknown',
      healthMessage: '',
      lastHealthAt: ''
    };
  }

  const fillForm = (data) => {
    Object.assign(form, emptySiteForm(), {
      name: data.name,
      siteType: data.siteType,
      remark: data.remark || '',
      cookieName: data.cookieName,
      cookieDomain: data.cookieDomain || '',
      attributionTtlDays: data.attributionTtlDays,
      renewMode: data.renewMode,
      noAttributionAction: data.noAttributionAction,
      failMode: data.failMode,
      minConfidence: data.minConfidence,
      handoffBanner: data.handoffBanner,
      handoffBannerText: data.handoffBannerText,
      handoffLinkRewrite: data.handoffLinkRewrite,
      handoffTokenTtlHours: data.handoffTokenTtlHours,
      handoffTokenMaxUses: data.handoffTokenMaxUses,
      handoffMatchEnabled: data.handoffMatchEnabled,
      handoffMatchMinutes: data.handoffMatchMinutes,
      handoffMatchSysOnly: data.handoffMatchSysOnly ?? true,
      secretIpAllowlistText: (data.secretIpAllowlist || []).join('\n')
    });
    Object.assign(origins.real, emptyOrigin());
    Object.assign(origins.review, emptyOrigin());
    (data.origins || []).forEach((o) => {
      if (origins[o.kind]) Object.assign(origins[o.kind], o);
    });
    const rv = data.review || {};
    review.source = rv.source || 'template';
    review.templateId = rv.templateId || undefined;
    review.siteName = rv.templateVars?.siteName || '';
    review.slogan = rv.templateVars?.slogan || '';
    pathRules.value = (data.pathRules || []).map((r) => ({ ...r }));
  };

  const loadDomains = () => {
    if (!siteId.value) return;
    pageMerchantDomains({ siteId: siteId.value, pageNum: 1, pageSize: 50 })
      .then((res) => (domains.value = res.rows || []))
      .catch(() => {});
  };

  const load = () => {
    if (!siteId.value) {
      detail.value = null;
      Object.assign(form, emptySiteForm());
      return;
    }
    loading.value = true;
    getSite(siteId.value)
      .then((data) => {
        detail.value = data;
        fillForm(data);
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => (loading.value = false));
    loadDomains();
  };

  watch(siteId, load, { immediate: true });

  listReviewTemplates()
    .then((rows) => (templates.value = rows || []))
    .catch(() => {});

  const goBack = () => router.push('/merchant/site');

  const showSecret = (secret) => {
    serverSecret.value = secret;
    secretVisible.value = true;
  };

  const copySecret = async () => {
    try {
      await navigator.clipboard.writeText(serverSecret.value);
      EleMessage.success({ message: '已复制', plain: true });
    } catch (e) {
      EleMessage.error({ message: '复制失败，请手动选择密钥', plain: true });
    }
  };

  const handleSave = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) return;
      saving.value = true;
      const payload = siteToPayload(form);
      const req = siteId.value
        ? updateSite(siteId.value, payload)
        : addSite(payload);
      req
        .then((res) => {
          saving.value = false;
          if (siteId.value) {
            EleMessage.success({ message: res, plain: true });
            load();
            return;
          }
          showSecret(res.data.serverSecret);
          router.replace({
            path: '/merchant/site/form',
            query: { id: res.data.siteId }
          });
        })
        .catch((e) => {
          saving.value = false;
          EleMessage.error({ message: e.message, plain: true });
        });
    });
  };

  const handleSaveOrigin = (kind) => {
    const row = origins[kind];
    saveSiteOrigin(siteId.value, {
      kind,
      url: row.url,
      hostHeader: row.hostHeader,
      timeoutMs: row.timeoutMs
    })
      .then((msg) => EleMessage.success({ message: msg, plain: true }))
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));
  };

  const handleProbe = (kind) => {
    const loadingMsg = EleMessage.loading({ message: '检测中..', plain: true });
    probeSiteOrigin(siteId.value, kind)
      .then((data) => {
        loadingMsg.close();
        Object.assign(origins[kind], data);
        EleMessage.success({
          message: data.healthMessage || '检测完成',
          plain: true
        });
      })
      .catch((e) => {
        loadingMsg.close();
        EleMessage.error({ message: e.message, plain: true });
      });
  };

  const handleSaveReview = () => {
    const payload = { source: review.source };
    if (review.source === 'template') {
      payload.templateId = review.templateId;
      payload.templateVars = {
        siteName: review.siteName,
        slogan: review.slogan
      };
    }
    saveSiteReview(siteId.value, payload)
      .then((msg) => EleMessage.success({ message: msg, plain: true }))
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));
  };

  const handlePreview = () => {
    if (!review.templateId) {
      EleMessage.error({ message: '请选择模板', plain: true });
      return;
    }
    previewReviewTemplate(review.templateId, {
      siteName: review.siteName,
      slogan: review.slogan
    })
      .then((html) => {
        previewHtml.value = html;
        previewVisible.value = true;
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));
  };

  const onPackage = (file) => {
    packageFile.value = file.raw;
  };

  const handleUpload = () => {
    if (!packageFile.value) return;
    uploading.value = true;
    uploadSitePackage(siteId.value, packageFile.value)
      .then((res) => {
        uploading.value = false;
        previewUrl.value = res.data.previewUrl;
        review.source = 'static';
        EleMessage.success({ message: res.msg, plain: true });
      })
      .catch((e) => {
        uploading.value = false;
        EleMessage.error({ message: e.message, plain: true });
      });
  };

  const addRule = () => {
    pathRules.value.push({
      pattern: '/',
      matchType: 'prefix',
      action: 'bypass',
      sort: (pathRules.value.length + 1) * 10,
      remark: ''
    });
  };

  const handleSaveRules = () => {
    saveSitePathRules(siteId.value, pathRules.value)
      .then((msg) => EleMessage.success({ message: msg, plain: true }))
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));
  };

  const openGuide = (row) => {
    openModal({
      custom: true,
      asyncComponent: () => import('../../domain/components/domain-guide.vue'),
      componentProps: { data: row }
    });
  };

  const handleAddDomain = () => {
    openModal({
      custom: true,
      asyncComponent: () => import('../../domain/components/domain-edit.vue'),
      componentProps: {
        siteId: siteId.value,
        onDone: (row) => {
          loadDomains();
          if (row) openGuide(row);
        }
      }
    });
  };

  const handleVerify = (row) => {
    const loadingMsg = EleMessage.loading({ message: '校验中..', plain: true });
    verifyMerchantDomain(row.domainId)
      .then((res) => {
        loadingMsg.close();
        EleMessage.success({ message: res.msg, plain: true });
        loadDomains();
        openGuide(res.data);
      })
      .catch((e) => {
        loadingMsg.close();
        EleMessage.error({ message: e.message, plain: true });
      });
  };

  const handleRotateKey = (graceHours) => {
    const text =
      graceHours > 0
        ? `轮换访客令牌密钥，旧 Cookie 在 ${graceHours} 小时内仍可放行。是否继续？`
        : '旧 Cookie 立即失效，所有访客需要重新通过归因参数访问才能看到真实内容。是否继续？';
    ElMessageBox.confirm(text, '系统提示', { type: 'warning', draggable: true })
      .then(() => rotateSiteKey(siteId.value, graceHours))
      .then((res) => {
        if (!res) return;
        EleMessage.success({ message: res.msg, plain: true });
        load();
      })
      .catch((e) => {
        if (e?.message) EleMessage.error({ message: e.message, plain: true });
      });
  };

  const handleResetSecret = () => {
    ElMessageBox.confirm(
      '重置后旧密钥立即失效，使用旧密钥的服务端判定会失败。是否继续？',
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => {
        const allow = siteToPayload(form).secretIpAllowlist;
        return resetSiteSecret(siteId.value, allow);
      })
      .then((res) => {
        if (!res) return;
        showSecret(res.data.serverSecret);
        load();
      })
      .catch((e) => {
        if (e?.message) EleMessage.error({ message: e.message, plain: true });
      });
  };
</script>

<style scoped>
  .field-hint {
    margin-left: 8px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .banner-stage {
    position: relative;
    height: 280px;
    border-radius: 10px;
    background: #f5f6f8;
    overflow: hidden;
  }

  .banner-page {
    padding: 28px 24px;
  }

  .banner-line {
    height: 12px;
    margin-bottom: 12px;
    border-radius: 6px;
    background: #e4e7ed;
  }

  .banner-line.short {
    width: 46%;
  }

  .banner-line.mid {
    width: 72%;
  }

  .ck-preview {
    position: absolute;
    left: 50%;
    top: 18%;
    transform: translateX(-50%);
    width: min(360px, calc(100% - 32px));
    box-sizing: border-box;
    padding: 16px 16px 14px;
    background: #fff;
    color: #1f2329;
    border-radius: 12px;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.22);
    font-size: 14px;
    line-height: 1.5;
  }

  .ck-x {
    position: absolute;
    top: 8px;
    right: 10px;
    border: 0;
    background: none;
    color: #8a8f99;
    font-size: 20px;
    line-height: 1;
    padding: 0;
    cursor: pointer;
  }

  .ck-text {
    margin: 0 22px 14px 0;
  }

  .ck-open {
    display: inline-block;
    background: #1677ff;
    color: #fff;
    border-radius: 6px;
    padding: 8px 14px;
    font-weight: 600;
  }

  .banner-note {
    margin: 12px 0 0;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .key-box {
    padding: 14px 16px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 10px;
    height: 100%;
  }

  .key-box .key-head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 6px;
  }

  .key-box p {
    margin: 0 0 12px;
    font-size: 12px;
    line-height: 1.6;
  }

  .origin-box {
    padding: 12px 12px 4px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 10px;
    margin-bottom: 12px;
  }

  .origin-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  .origin-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 12px;
    flex-wrap: wrap;
  }

  .block-gap {
    margin-top: 16px;
  }

  .form-bar {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  .secret-box {
    padding: 12px;
    border-radius: 8px;
    background: var(--el-fill-color-light);
    word-break: break-all;
  }

  .preview-frame {
    width: 100%;
    height: 480px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }
</style>
