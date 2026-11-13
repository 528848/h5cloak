<!-- 归因码单元格：有值时带复制按钮，没有值时显示 -。 -->
<template>
  <span v-if="value" class="code-cell">
    <span class="mono">{{ value }}</span>
    <el-tooltip content="复制" placement="top">
      <el-button
        link
        type="primary"
        :icon="DocumentCopy"
        aria-label="复制"
        @click.stop="copy"
      />
    </el-tooltip>
  </span>
  <span v-else class="muted-text">-</span>
</template>

<script setup>
  import { computed } from 'vue';
  import { DocumentCopy } from '@element-plus/icons-vue';
  import { EleMessage } from 'ele-admin-plus';

  const props = defineProps({
    text: { type: [String, Number], default: '' }
  });

  const value = computed(() => String(props.text ?? '').trim());

  const copy = async () => {
    if (!value.value) return;
    try {
      await navigator.clipboard.writeText(value.value);
      EleMessage.success({ message: '已复制', plain: true });
    } catch (e) {
      EleMessage.error({ message: '复制失败', plain: true });
    }
  };
</script>

<style scoped>
  .code-cell {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    max-width: 100%;
  }

  .code-cell .mono {
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
