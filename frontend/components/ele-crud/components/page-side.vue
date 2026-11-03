<!-- 侧栏 -->
<template>
  <EleLoading
    :loading="loading"
    class="ele-crud-tree-wrapper"
    v-bind="sideConfig?.loadingProps || {}"
  >
    <ElTree
      ref="treeRef"
      :data="data"
      :highlightCurrent="true"
      :expandOnClickNode="false"
      :defaultExpandAll="true"
      :filterNodeMethod="filterNode"
      class="ele-crud-tree"
      :nodeKey="defaultTreeNodeKey"
      v-bind="sideConfig?.treeProps || {}"
      @node-click="handleNodeClick"
    >
      <template #empty>
        <ElEmpty
          :imageSize="68"
          :description="errorMessage"
          v-bind="sideConfig?.emptyProps || {}"
        />
      </template>
      <template
        v-for="(slotName, compSlotName) in getSlotsMap(
          $slots,
          sideConfig?.treeSlots,
          ['empty']
        )"
        #[compSlotName]="slotProps"
      >
        <slot :name="slotName" v-bind="slotProps || {}"></slot>
      </template>
    </ElTree>
  </EleLoading>
</template>

<script setup>
  import { ref, watch, onMounted } from 'vue';
  import { ElTree, ElEmpty } from 'element-plus';
  import { getSlotsMap } from '../../utils/common';
  import EleLoading from '../../ele-loading/index.vue';
  import {
    defaultTreeNodeKey,
    getTreeValueField,
    getTreeLabelField
  } from '../util';

  defineOptions({ name: 'PageSide' });

  const props = defineProps({
    /** 侧栏配置 */
    sideConfig: Object,
    /** 数据 */
    data: Array,
    /** 加载状态 */
    loading: Boolean,
    /** 加载错误信息 */
    errorMessage: String,
    /** 选中值 */
    selectedValue: [String, Number, Boolean, Object, Array],
    /** 侧栏树搜索关键字 */
    keywords: String,
    /** 国际化 */
    lang: {
      type: Object,
      required: true
    }
  });

  const emit = defineEmits({
    /** 侧栏树点击事件 */
    treeNodeClick: (_nodeValue) => true
  });

  /** 树组件 */
  const treeRef = ref(null);

  /** 设置树选中 */
  const setTreeCurrentKey = (key) => {
    if (key != null) {
      try {
        treeRef.value?.setCurrentKey?.(key);
      } catch (e) {
        console.error(e);
      }
    }
  };

  /** 选择数据 */
  const handleNodeClick = (item) => {
    if (item != null) {
      emit('treeNodeClick', item[getTreeValueField(props.sideConfig)]);
    }
  };

  /** 树过滤方法 */
  const filterNode = (value, item) => {
    if (value) {
      const itemLabel = item[getTreeLabelField(props.sideConfig)];
      return !!(itemLabel && itemLabel.includes(value));
    }
    return true;
  };

  /** 树过滤 */
  watch(
    () => props.keywords,
    (value) => {
      treeRef.value?.filter?.(value);
    }
  );

  /** 更新选中 */
  watch(
    () => props.selectedValue,
    (selected) => {
      setTreeCurrentKey(selected);
    }
  );

  /** 初始选中 */
  onMounted(() => {
    setTreeCurrentKey(props.selectedValue);
  });
</script>
