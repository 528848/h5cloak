<!-- 商户详情抽屉：Hero + 摘要条 + 分区卡片 -->
<template>
  <el-drawer
    v-bind="modalProps"
    :size="mobile ? '100%' : 760"
    title="商户详情"
    destroy-on-close
  >
    <div v-loading="loading">
      <el-empty
        v-if="!loading && !data"
        :image-size="90"
        description="商户不存在"
      />
      <template v-else-if="data">
        <ele-card
          class="detail-hero page-section"
          :body-style="{ padding: '18px 20px' }"
        >
          <div class="hero-head">
            <div>
              <div class="hero-title">
                <h3>{{ data.merchantName }}</h3>
                <el-tag
                  :type="
                    getOptionType(merchantStatusOptions, data.effectiveStatus)
                  "
                  effect="light"
                >
                  {{
                    getOptionLabel(merchantStatusOptions, data.effectiveStatus)
                  }}
                </el-tag>
                <el-tag v-if="data.expireAt" type="warning" effect="plain"
                  >到期 {{ data.expireAt }}</el-tag
                >
                <el-tag v-else type="info" effect="plain">长期</el-tag>
              </div>
              <div class="hero-desc">
                商户 ID {{ data.merchantId }} · 开通于 {{ data.createdAt }} ·
                时区 {{ data.timezone }}
              </div>
            </div>
            <el-space>
              <el-button
                v-permission="'merchant:account:quota'"
                type="primary"
                plain
                @click="handleQuota"
                >额度与功能</el-button
              >
              <el-button
                v-permission="'merchant:account:edit'"
                plain
                @click="handleEdit"
                >编辑</el-button
              >
            </el-space>
          </div>
          <div class="summary-strip">
            <div class="summary-item is-amount">
              <span>判定额度 / 月</span>
              <strong>{{
                Number(data.quota.decisionQuota).toLocaleString()
              }}</strong>
            </div>
            <div class="summary-item"
              ><span>站点</span><strong>{{ data.quota.siteQuota }}</strong></div
            >
            <div class="summary-item"
              ><span>域名</span
              ><strong>{{ data.quota.domainQuota }}</strong></div
            >
            <div class="summary-item"
              ><span>分销码</span
              ><strong>{{ data.quota.codeQuota }}</strong></div
            >
            <div class="summary-item">
              <span>子账号</span>
              <strong>{{ data.userCount }} / {{ data.quota.userQuota }}</strong>
            </div>
            <div class="summary-item"
              ><span>日志保留</span
              ><strong>{{ data.quota.logRetentionDays }} 天</strong></div
            >
          </div>
        </ele-card>

        <el-row :gutter="16">
          <el-col :md="16" :xs="24">
            <section-card
              title="商户资料"
              subtitle="联系方式与管理员账号"
              class="page-section"
            >
              <el-descriptions
                :column="mobile ? 1 : 2"
                border
                class="detail-table"
              >
                <el-descriptions-item label="联系人">{{
                  data.contactName || '-'
                }}</el-descriptions-item>
                <el-descriptions-item label="联系电话">{{
                  data.contactPhone || '-'
                }}</el-descriptions-item>
                <el-descriptions-item label="联系邮箱" :span="2">{{
                  data.contactEmail || '-'
                }}</el-descriptions-item>
                <el-descriptions-item label="管理员账号">
                  <span class="mono">{{ data.adminUsername || '-' }}</span>
                </el-descriptions-item>
                <el-descriptions-item label="达到额度后">
                  {{
                    getOptionLabel(quotaPolicyOptions, data.quota.quotaPolicy)
                  }}
                </el-descriptions-item>
                <el-descriptions-item label="备注" :span="2">{{
                  data.remark || '-'
                }}</el-descriptions-item>
                <el-descriptions-item label="最近更新" :span="2">{{
                  data.updatedAt
                }}</el-descriptions-item>
              </el-descriptions>
            </section-card>
          </el-col>
          <el-col :md="8" :xs="24">
            <section-card
              title="功能开关"
              subtitle="按商户单独配置"
              class="page-section"
            >
              <div v-for="f in featureOptions" :key="f.key" class="side-kv">
                <span>{{ f.label }}</span>
                <el-tag
                  :type="data.features[f.key] ? 'success' : 'info'"
                  effect="plain"
                  size="small"
                >
                  {{ data.features[f.key] ? '开启' : '关闭' }}
                </el-tag>
              </div>
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
  import { getMerchant } from '@/api/admin/merchant';
  import {
    merchantStatusOptions,
    quotaPolicyOptions,
    featureOptions,
    getOptionLabel,
    getOptionType
  } from '../constants';

  const props = defineProps({ merchantId: { type: Number, required: true } });
  const emit = defineEmits(['done']);
  const { modalProps, openModal } = useModal();
  const { mobile } = useMobile();

  const loading = ref(true);
  const data = ref(null);

  const load = () => {
    loading.value = true;
    getMerchant(props.merchantId)
      .then((d) => (data.value = d))
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => (loading.value = false));
  };

  const afterChange = () => {
    load();
    emit('done');
  };

  const handleQuota = () =>
    openModal({
      custom: true,
      asyncComponent: () => import('./merchant-quota.vue'),
      componentProps: { data: data.value, onDone: afterChange }
    });

  const handleEdit = () =>
    openModal({
      custom: true,
      asyncComponent: () => import('./merchant-edit.vue'),
      componentProps: { data: data.value, onDone: afterChange }
    });

  onMounted(load);
</script>
