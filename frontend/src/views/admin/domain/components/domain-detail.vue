<!-- 域名详情弹窗 -->
<template>
  <ele-modal :width="640" title="域名详情" v-bind="modalProps">
    <el-descriptions :column="2" border class="domain-detail-desc">
      <el-descriptions-item label="域名" :span="2">
        <div class="domain-detail-link-row">
          <span class="domain-detail-link-inline">
            <span class="domain-detail-link-text">{{ data.domain || '-' }}</span>
            <ele-copyable
              v-if="isCopyable(data.domain)"
              :text="data.domain"
              :tooltip="false"
              class="domain-detail-copy"
              @copy="handleCopied"
            />
          </span>
        </div>
      </el-descriptions-item>
      <el-descriptions-item label="所属商户">
        {{ data.merchantName || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="状态">
        <el-tag :type="statusType[data.status] || 'info'" size="small">
          {{ statusText[data.status] || data.status || '-' }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="解析类型">
        {{ data.verifyType || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="关联斗篷">
        {{ data.cloakCount ?? 0 }}
      </el-descriptions-item>
      <el-descriptions-item label="解析值" :span="2">
        <div class="domain-detail-link-row">
          <span class="domain-detail-link-inline">
            <span class="domain-detail-link-text">{{ data.verifyValue || '-' }}</span>
            <ele-copyable
              v-if="isCopyable(data.verifyValue)"
              :text="data.verifyValue"
              :tooltip="false"
              class="domain-detail-copy"
              @copy="handleCopied"
            />
          </span>
        </div>
      </el-descriptions-item>
      <el-descriptions-item label="最后检测">
        {{ data.lastCheckedAt || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="创建时间">
        {{ data.createdAt || '-' }}
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
  import { EleMessage, useModal } from 'ele-admin-plus';

  defineProps({
    data: {
      type: Object,
      required: true
    },
    statusText: {
      type: Object,
      required: true
    },
    statusType: {
      type: Object,
      required: true
    }
  });

  const { modalProps, closeModal } = useModal();

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
  .domain-detail-desc {
    width: 100%;
  }

  .domain-detail-link-row {
    line-height: 22px;
  }

  .domain-detail-link-inline {
    display: inline;
    word-break: break-all;
  }

  .domain-detail-link-text {
    word-break: break-all;
  }

  .domain-detail-copy {
    margin-left: 4px;
  }

  .domain-detail-link-inline :deep(.ele-copyable) {
    display: inline-flex;
    vertical-align: middle;
  }
</style>
