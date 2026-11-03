<!-- 概览卡片行（UI 规范 §2.3）。items: { key, label, value, desc, type, alert, clickable } -->
<template>
  <el-row :gutter="16" class="summary-row page-section">
    <el-col
      v-for="item in items"
      :key="item.key"
      :xs="24"
      :sm="12"
      :md="colMd"
      :lg="colLg"
    >
      <ele-card
        :body-style="{ padding: '16px 18px' }"
        class="summary-card"
        :class="[
          `is-${item.type || 'primary'}`,
          {
            'is-alert': item.alert,
            'is-clickable': item.clickable ?? clickable,
            'is-active': activeKey != null && activeKey === item.key
          }
        ]"
        @click="handleClick(item)"
      >
        <div class="summary-head">
          <span class="summary-dot"></span>
          <span class="summary-label">{{ item.label }}</span>
        </div>
        <div class="summary-value">{{ format(item.value) }}</div>
        <div v-if="item.desc" class="summary-desc">{{ item.desc }}</div>
      </ele-card>
    </el-col>
  </el-row>
</template>

<script setup>
  import { computed } from 'vue';

  defineOptions({ name: 'SummaryCards' });

  const props = defineProps({
    items: { type: Array, default: () => [] },
    /** 每行列数（lg 及以上），默认按数量自适应，最多 6 */
    cols: { type: Number, default: 0 },
    clickable: { type: Boolean, default: false },
    activeKey: { type: [String, Number], default: null }
  });

  const emit = defineEmits(['select']);

  const colLg = computed(() => {
    const n = props.cols || Math.min(Math.max(props.items.length, 1), 6);
    return Math.max(Math.floor(24 / n), 4);
  });
  const colMd = computed(() => Math.max(colLg.value, 8));

  const format = (v) => {
    if (v == null || v === '') return '-';
    if (typeof v === 'number') return v.toLocaleString();
    return v;
  };

  const handleClick = (item) => {
    if (item.clickable ?? props.clickable) {
      emit('select', item);
    }
  };
</script>
