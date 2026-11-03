<!-- 提及 -->
<template>
  <ElMention
    v-bind="omit($props, ['options'])"
    ref="mentionRef"
    :options="optionData"
    @update:modelValue="emitMethods['update:modelValue']"
    @search="emitMethods['search']"
    @select="emitMethods['select']"
    @focus="emitMethods['focus']"
    @blur="emitMethods['blur']"
  >
    <template v-for="name in Object.keys($slots)" #[name]="slotProps">
      <slot :name="name" v-bind="slotProps || {}"></slot>
    </template>
  </ElMention>
</template>

<script setup>
  import { ref } from 'vue';
  import { ElMention } from 'element-plus';
  import { omit } from '../utils/common';
  import { useComponentEvents, useProOptions } from '../utils/hook';
  import { mentionProps, mentionEmits } from './props';

  defineOptions({ name: 'EleMention' });

  const props = defineProps(mentionProps);

  const emit = defineEmits(mentionEmits);

  const { emitMethods } = useComponentEvents(mentionEmits, emit);
  const { optionData, reloadOptions } = useProOptions(props);

  /** 组件引用 */
  const mentionRef = ref(null);

  defineExpose({
    reloadOptions,
    mentionRef
  });
</script>
