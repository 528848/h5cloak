<!-- API Key 创建成功 -->
<template>
  <ele-modal
    :width="520"
    title="API 斗篷规则创建成功"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    v-bind="modalProps"
  >
    <div class="api-key-secret-body">
      <el-icon class="api-key-secret-icon" :size="22">
        <CircleCheckFilled />
      </el-icon>
      <div class="api-key-secret-content">
        <p class="api-key-secret-tip">
          请妥善保存 Token，关闭后无法再次查看完整密钥：
        </p>
        <div class="api-key-secret-row">
          <span class="api-key-secret-text">{{ secret }}</span>
          <ele-copyable
            :text="secret"
            :tooltip="false"
            class="api-key-secret-copy"
            @copy="handleCopied"
          />
        </div>
      </div>
    </div>
    <template #footer>
      <btn-items
        :items="[
          { title: '我已保存', type: 'primary', onClick: () => closeModal() }
        ]"
      />
    </template>
  </ele-modal>
</template>

<script setup>
  import { CircleCheckFilled } from '@element-plus/icons-vue';
  import { EleMessage, useModal } from 'ele-admin-plus';

  defineProps({
    secret: {
      type: String,
      required: true
    }
  });

  const { modalProps, closeModal } = useModal();

  const handleCopied = (error) => {
    if (error) {
      EleMessage.error({ message: '复制失败', plain: true });
      return;
    }
    EleMessage.success({ message: '复制成功', plain: true });
  };
</script>

<style scoped>
  .api-key-secret-body {
    display: flex;
    align-items: flex-start;
    gap: 12px;
  }

  .api-key-secret-icon {
    flex-shrink: 0;
    margin-top: 2px;
    color: var(--el-color-success);
  }

  .api-key-secret-content {
    flex: 1;
    min-width: 0;
  }

  .api-key-secret-tip {
    margin: 0 0 10px;
    line-height: 1.6;
    color: var(--el-text-color-regular);
  }

  .api-key-secret-row {
    display: flex;
    align-items: flex-start;
    gap: 8px;
  }

  .api-key-secret-text {
    flex: 1;
    min-width: 0;
    word-break: break-all;
    line-height: 1.6;
    color: var(--el-text-color-primary);
    font-family: Consolas, Monaco, 'Courier New', monospace;
    font-size: 13px;
  }

  .api-key-secret-copy {
    flex-shrink: 0;
    margin-top: 2px;
  }
</style>
