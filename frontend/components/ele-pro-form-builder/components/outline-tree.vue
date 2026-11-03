<!-- 大纲树列表 -->
<template>
  <OutlineList
    v-if="formItems"
    :items="formItems"
    :currentFormItemId="currentFormItemId"
    :collapseItemIds="collapseItemIds"
    :componentData="componentData"
    :itemTypeData="itemTypeData"
    @update:currentFormItemId="handleUpdateCurrentFormItemId"
    @toggleItemCollapse="handleToggleItemCollapse"
    @deleteItem="handleDeleteItem"
    @copyItem="handleCopyItem"
    @addChildren="handleAddChildren"
    @openTableTool="handleOpenTableTool"
    @updateItemChildren="handleUpdateItemChildren"
  />
  <ElEmpty
    v-if="!formItems || !formItems.length"
    :imageSize="58"
    class="ele-pro-form-builder-form-empty"
  />
</template>

<script setup>
  import { ref, watch } from 'vue';
  import { ElEmpty } from 'element-plus';
  import { findTree } from '../../utils/common';
  import { fixedChildTypes } from './build-core';
  import { generateCopyItemData, generateAddChildData } from './build-util';
  import OutlineList from './outline-list.vue';

  defineOptions({ name: 'OutlineTree' });

  const props = defineProps({
    /** 全部表单项 */
    formItems: Array,
    /** 选中的表单项 */
    currentFormItemId: [String, Number],
    /** 组件库数据 */
    componentData: Array,
    /** 高级表单组件类型数据 */
    itemTypeData: Array
  });

  const emit = defineEmits({
    /** 更新选中的表单项事件 */
    'update:currentFormItemId': (_formItemId) => true,
    /** 更新表单项数据事件 */
    updateItems: (_result) => true,
    /** 更新表单项排序事件 */
    updateItemChildren: (_children, _parentKey) => true,
    /** 打开表格更多操作事件 */
    openTableTool: (_formItemId, _el) => true,
    /** 打开子组件选择器事件 */
    openComponentPicker: (_formItemId) => true
  });

  /** 折叠的表单项 */
  const collapseItemIds = ref([]);

  /** 更新选中的表单项 */
  const handleUpdateCurrentFormItemId = (itemId) => {
    emit('update:currentFormItemId', itemId);
  };

  /** 更新表单项数据 */
  const handleUpdateItems = (result) => {
    emit('updateItems', result);
  };

  /** 删除表单项 */
  const handleDeleteItem = (formItemId) => {
    handleUpdateItems({
      deleteItemIds: [formItemId],
      addItems: [],
      updateItems: []
    });
  };

  /** 复制表单项 */
  const handleCopyItem = (formItemId) => {
    handleUpdateItems(generateCopyItemData(formItemId, props.formItems));
  };

  /** 添加子级表单项 */
  const handleAddChildren = (triggerItem, action) => {
    if (
      triggerItem.type &&
      fixedChildTypes.some((d) => d.type === triggerItem.type)
    ) {
      const result = generateAddChildData(
        triggerItem,
        void 0,
        void 0,
        action,
        props.formItems,
        void 0,
        props.componentData
      );
      handleUpdateItems(result);
    } else {
      emit('openComponentPicker', triggerItem.key);
    }
  };

  /** 打开表格更多操作 */
  const handleOpenTableTool = (item, e) => {
    const el = e.currentTarget;
    const triggerEl = el?.parentElement?.querySelector?.(
      '.ele-pro-form-builder-outline-item-table-tool-trigger'
    );
    emit('openTableTool', item.key, triggerEl);
  };

  /** 更新表单项排序 */
  const handleUpdateItemChildren = (children, parentKey) => {
    emit('updateItemChildren', children, parentKey);
  };

  /** 表单项折叠切换 */
  const handleToggleItemCollapse = (formItemId) => {
    const index = collapseItemIds.value.indexOf(formItemId);
    if (index !== -1) {
      collapseItemIds.value.splice(index, 1);
    } else {
      collapseItemIds.value.push(formItemId);
    }
  };

  /** 表单项数据更新后移除不存在的折叠数据 */
  watch(
    () => props.formItems,
    (items) => {
      for (let i = collapseItemIds.value.length - 1; i >= 0; i--) {
        if (!findTree(items, (item) => collapseItemIds.value[i] === item.key)) {
          collapseItemIds.value.splice(i, 1);
        }
      }
    },
    { deep: true }
  );
</script>
