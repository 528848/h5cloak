<template>
  <span>{{ typeName }}</span>
</template>

<script setup>
  import { computed } from 'vue';
  import { getComponentItemByType } from './build-core';

  defineOptions({ name: 'ComponentName' });

  const props = defineProps({
    /** 表单项组件类型 */
    itemType: String,
    /** 组件库数据 */
    componentData: Array
  });

  /** 组件类型名称 */
  const typeName = computed(() => {
    const type = props.itemType;
    return getComponentItemByType(type, props.componentData)?.name ?? type;
  });
</script>
