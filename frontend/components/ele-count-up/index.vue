<!-- 滚动数字 -->
<template>
  <span ref="rootRef"></span>
</template>

<script setup>
  import { ref, unref, watch, onMounted, onBeforeUnmount } from 'vue';
  import { CountUp } from 'countup.js';
  import { countUpProps, countUpEmits } from './props';

  defineOptions({ name: 'EleCountUp' });

  const props = defineProps(countUpProps);

  const emit = defineEmits(countUpEmits);

  const isFunction = (value) => typeof value === 'function';

  /** 实例 */
  let instance;

  /** 根节点 */
  const rootRef = ref(null);

  /** 创建 */
  const create = () => {
    const elem = unref(rootRef);
    if (instance || !elem) {
      return;
    }
    const ins = new CountUp(elem, props.endVal ?? 0, props.options);
    if (ins.error) {
      console.warn(ins);
      return;
    }
    instance = ins;
  };

  /** 打印值 */
  const printValue = (value) => {
    if (instance && isFunction(instance.printValue)) {
      return instance.printValue(value);
    }
  };

  /** 开启动画 */
  const start = (callback) => {
    if (instance && isFunction(instance.start)) {
      return instance.start(callback);
    }
  };

  /** 暂停/恢复 */
  const pauseResume = () => {
    if (instance && isFunction(instance.pauseResume)) {
      return instance.pauseResume();
    }
  };

  /** 重置 */
  const reset = () => {
    if (instance && isFunction(instance.reset)) {
      return instance.reset();
    }
  };

  /** 更新结束值 */
  const update = (newEndVal) => {
    if (instance && isFunction(instance.update)) {
      return instance.update(newEndVal ?? 0);
    }
  };

  /** 销毁 */
  const destroy = () => {
    reset();
    instance = null;
  };

  watch(
    () => props.endVal,
    (value) => {
      update(value);
    }
  );

  watch(
    () => props.options,
    () => {
      destroy();
      create();
      emit('ready', instance);
    }
  );

  onMounted(() => {
    create();
    if (props.delay < 0) {
      emit('ready', instance);
      return;
    }
    setTimeout(() => {
      if (instance && isFunction(instance.start)) {
        instance.start(() => {
          emit('ready', instance);
        });
      }
    }, props.delay);
  });

  onBeforeUnmount(() => {
    destroy();
  });

  defineExpose({
    printValue,
    start,
    pauseResume,
    reset,
    update
  });
</script>
