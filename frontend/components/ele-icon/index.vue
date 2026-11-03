<!-- 图标 -->
<template>
  <ElIcon ref="iconRef" :size="size" :color="color">
    <component
      v-if="iconComponent"
      :is="iconComponent"
      :class="iconComponent === defaultIconTag ? name : void 0"
    />
    <slot></slot>
  </ElIcon>
</template>

<script setup>
  import { ref, computed } from 'vue';
  import { ElIcon } from 'element-plus';
  import { iconProps } from './props';
  const defaultIconTag = 'i';

  defineOptions({ name: 'EleIcon' });

  const props = defineProps(iconProps);

  /** 组件引用 */
  const iconRef = ref(null);

  /** 图标名称对应的组件 */
  const iconComponent = computed(() => {
    if (
      props.name != null &&
      (props.iconType === 'fontClass' ||
        (typeof props.name === 'string' && props.name.includes(' ')))
    ) {
      return defaultIconTag;
    }
    return props.name;
  });

  defineExpose({
    iconRef
  });
</script>
