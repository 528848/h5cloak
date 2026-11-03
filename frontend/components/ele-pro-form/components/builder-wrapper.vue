<template>
  <div
    class="ele-pro-form-builder-item-wrapper"
    :class="[
      { 'is-div-type': item.type === 'div' },
      { 'is-active': activeItemKey != null && activeItemKey === item.key }
    ]"
    @click.stop="handleItemBuilderWrapperClick"
  >
    <slot></slot>
    <BuilderToolWrapper
      :item="item"
      :activeItemKey="activeItemKey"
      :handle="handle"
    >
      <template v-for="name in Object.keys($slots)" #[name]="slotProps">
        <slot :name="name" v-bind="slotProps || {}"></slot>
      </template>
    </BuilderToolWrapper>
  </div>
</template>

<script setup>
  import BuilderToolWrapper from './builder-tool-wrapper.vue';

  defineOptions({ name: 'BuilderWrapper' });

  const props = defineProps({
    /** 表单项 */
    item: {
      type: Object,
      required: true
    },
    /** 编辑模式选中的表单项 */
    activeItemKey: [String, Number],
    /** 是否需要拖拽手柄 */
    handle: Boolean
  });

  const emit = defineEmits({
    'update:activeItemKey': (_activeKey) => true
  });

  /** 表单项构建容器点击事件 */
  const handleItemBuilderWrapperClick = () => {
    emit('update:activeItemKey', props.item.key);
  };
</script>
