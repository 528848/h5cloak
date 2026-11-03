<!-- 树下拉 -->
<template>
  <ElTreeSelect
    v-bind="omit($props, ['data'])"
    ref="treeSelectRef"
    :data="optionData"
    @update:modelValue="emitMethods['update:modelValue']"
  >
    <template v-for="name in Object.keys($slots)" #[name]="slotProps">
      <slot :name="name" v-bind="slotProps || {}"></slot>
    </template>
  </ElTreeSelect>
</template>

<script setup>
  import { ref } from 'vue';
  import { ElTreeSelect } from 'element-plus';
  import { omit } from '../utils/common';
  import { useComponentEvents, useProOptions } from '../utils/hook';
  import { selectTreeProps, selectTreeEmits } from './props';

  defineOptions({ name: 'EleSelectTree' });

  const props = defineProps(selectTreeProps);

  const emit = defineEmits(selectTreeEmits);

  const { emitMethods } = useComponentEvents(selectTreeEmits, emit);
  const { optionData, reloadOptions } = useProOptions(props, 'data');

  /** 组件引用 */
  const treeSelectRef = ref(null);

  defineExpose({
    reloadOptions,
    treeSelectRef
  });
</script>
