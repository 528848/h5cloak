<!-- 斗篷详情弹窗 -->
<template>
  <ele-modal
    :width="720"
    title="斗篷详情"
    v-bind="modalProps"
  >
    <el-descriptions :column="2" border class="cloak-detail-desc">
      <el-descriptions-item label="斗篷名称">
        {{ data.cloakName || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="所属商户">
        {{ data.merchantName || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="关联域名">
        {{ data.domain || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="状态">
        <el-tag :type="data.status === 'enabled' ? 'success' : 'info'" size="small">
          {{ data.status === 'enabled' ? '启用' : '停用' }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="模式">
        {{ modeText[data.mode] || data.mode || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="投放地区">
        {{ data.targetRegionsText || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="真实链接" :span="2">
        <div class="cloak-detail-links">
          <div
            v-for="(url, index) in realUrls"
            :key="index"
            class="cloak-detail-link-row"
          >
            <span class="cloak-detail-link-inline">
              <span class="cloak-detail-link-text">{{ url }}</span>
              <ele-copyable
                v-if="isCopyable(url)"
                :text="url"
                :tooltip="false"
                class="cloak-detail-copy"
                @copy="handleCopied"
              />
            </span>
          </div>
        </div>
      </el-descriptions-item>
      <el-descriptions-item label="审核链接" :span="2">
        <div class="cloak-detail-link-row">
          <span class="cloak-detail-link-inline">
            <span class="cloak-detail-link-text">{{ data.whiteUrl || '-' }}</span>
            <ele-copyable
              v-if="isCopyable(data.whiteUrl)"
              :text="data.whiteUrl"
              :tooltip="false"
              class="cloak-detail-copy"
              @copy="handleCopied"
            />
          </span>
        </div>
      </el-descriptions-item>
      <el-descriptions-item label="跳转短链" :span="2">
        <div class="cloak-detail-link-row">
          <span class="cloak-detail-link-inline">
            <span class="cloak-detail-link-text">{{ data.shortLinkUrl || '-' }}</span>
            <ele-copyable
              v-if="isCopyable(data.shortLinkUrl)"
              :text="data.shortLinkUrl"
              :tooltip="false"
              class="cloak-detail-copy"
              @copy="handleCopied"
            />
          </span>
        </div>
      </el-descriptions-item>
      <el-descriptions-item label="设备白名单">
        {{ data.deviceWhitelist || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="允许访问来源">
        {{ data.allowedSourcesText || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="屏蔽 PC">
        {{ data.blockPc || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="屏蔽代理">
        {{ data.blockProxy || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="中间页">
        {{ data.intermediatePage || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="今日访问">
        {{ data.todayVisits ?? 0 }}
      </el-descriptions-item>
      <el-descriptions-item label="审核链接次数">
        {{ data.whiteHits ?? 0 }}
      </el-descriptions-item>
      <el-descriptions-item label="真实链接次数">
        {{ data.blackHits ?? 0 }}
      </el-descriptions-item>
      <el-descriptions-item label="创建时间">
        {{ data.createdAt || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="更新时间">
        {{ data.updatedAt || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="备注" :span="2">
        {{ data.remark || '-' }}
      </el-descriptions-item>
    </el-descriptions>
    <template #footer>
      <btn-items
        :items="[{ preset: 'cancel', title: '关闭', onClick: () => closeModal() }]"
      />
    </template>
  </ele-modal>
</template>

<script setup>
  import { computed } from 'vue';
  import { EleMessage, useModal } from 'ele-admin-plus';

  const props = defineProps({
    data: {
      type: Object,
      required: true
    }
  });

  const { modalProps, closeModal } = useModal();

  const modeText = {
    cloak: '斗篷',
    green: '绿色',
    full_open: '全开',
    review: '审核'
  };

  const realUrls = computed(() => {
    if (Array.isArray(props.data?.realUrls) && props.data.realUrls.length) {
      return props.data.realUrls;
    }
    return props.data?.blackUrl ? [props.data.blackUrl] : ['-'];
  });

  const isCopyable = (value) => {
    return !!value && value !== '-';
  };

  const handleCopied = (error) => {
    if (error) {
      EleMessage.error({ message: '复制失败', plain: true });
      return;
    }
    EleMessage.success({ message: '复制成功', plain: true });
  };
</script>

<style scoped>
  .cloak-detail-desc {
    width: 100%;
  }

  .cloak-detail-links {
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: 100%;
  }

  .cloak-detail-link-row {
    line-height: 22px;
  }

  .cloak-detail-link-inline {
    display: inline;
    word-break: break-all;
  }

  .cloak-detail-link-text {
    word-break: break-all;
  }

  .cloak-detail-copy {
    margin-left: 4px;
  }

  .cloak-detail-link-inline :deep(.ele-copyable) {
    display: inline-flex;
    vertical-align: middle;
  }
</style>
