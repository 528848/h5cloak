<!-- 分区卡片（UI 规范 §2.2）：标题 + 副标题 + 右侧操作 -->
<template>
  <ele-card class="app-section-card" v-bind="$attrs">
    <template #header>
      <div class="section-card-header">
        <div class="app-card-title">
          <span>{{ title }}</span>
          <em v-if="subtitle">{{ subtitle }}</em>
        </div>
        <div v-if="$slots.extra" class="section-card-extra">
          <slot name="extra"></slot>
        </div>
      </div>
    </template>
    <slot></slot>
    <template v-if="$slots.footer" #footer>
      <slot name="footer"></slot>
    </template>
  </ele-card>
</template>

<script setup>
  defineOptions({ name: 'SectionCard', inheritAttrs: false });

  defineProps({
    title: { type: String, required: true },
    subtitle: { type: String, default: '' }
  });
</script>

<style scoped>
  .section-card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }
</style>
