<!-- 步骤条 -->
<template>
  <ElSteps
    v-bind="omit($props, ['items', 'type'])"
    ref="stepsRef"
    class="ele-steps"
    :class="{ 'is-inline': direction === 'horizontal' && type === 'inline' }"
  >
    <ElStep v-for="(item, index) in optionData" :key="index" v-bind="item">
      <template
        v-for="name in Object.keys($slots).filter((k) => 'default' !== k)"
        #[name]
      >
        <slot :name="name" :index="index" :item="item"></slot>
      </template>
    </ElStep>
  </ElSteps>
</template>

<script setup>
  import { ref } from 'vue';
  import { ElSteps, ElStep } from 'element-plus';
  import { omit } from '../utils/common';
  import { useProOptions } from '../utils/hook';
  import { stepsProps } from './props';

  defineOptions({ name: 'EleSteps' });

  const props = defineProps(stepsProps);

  const { optionData, reloadOptions } = useProOptions(props, 'items');

  /** 组件引用 */
  const stepsRef = ref(null);

  defineExpose({
    reloadOptions,
    stepsRef
  });
</script>
