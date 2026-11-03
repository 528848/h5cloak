<template>
  <div class="ele-message-box-icon" v-show="type || icon">
    <component v-if="icon" :is="icon" />
    <CheckCircleFilled v-else-if="type === 'success'" />
    <ExclamationCircleFilled v-else-if="type === 'warning'" />
    <CloseCircleFilled v-else-if="type === 'error'" />
    <InfoCircleFilled v-else-if="type === 'info'" />
  </div>
</template>

<script setup>
  import { onBeforeUnmount, onMounted, getCurrentInstance } from 'vue';
  import {
    InfoCircleFilled,
    CheckCircleFilled,
    ExclamationCircleFilled,
    CloseCircleFilled
  } from '../../icons/index';

  defineOptions({ name: 'MessageBoxIcon' });

  const props = defineProps({
    /** 类型 */
    type: String,
    /** 图标 */
    icon: [String, Object, Function],
    /** 标识id */
    boxId: String
  });

  const emit = defineEmits({
    boxDestroy: (_boxId) => true,
    boxMounted: (_) => true
  });

  const ins = getCurrentInstance?.();

  onBeforeUnmount(() => {
    emit('boxDestroy', props.boxId);
  });

  onMounted(() => {
    emit('boxMounted', {
      boxId: props.boxId,
      doClose: ins?.ctx?.$root?.doClose
    });
  });
</script>
