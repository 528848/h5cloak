<!-- 主体表单区 -->
<template>
  <div
    class="ele-pro-form-builder-body"
    :class="[
      { 'is-pc': currentScreen === 'pc' },
      { 'is-pad': currentScreen === 'pad' },
      { 'is-phone': currentScreen === 'phone' }
    ]"
  >
    <component
      v-bind="formProps || {}"
      :is="proFormComponent || EleProForm"
      ref="proFormRef"
      :model="formData"
      :activeItemKey="currentFormItemId"
      :editable="true"
      :screenSize="currentScreen"
      :footer="false"
      :validateOnRuleChange="false"
      :scrollToError="false"
      :showMessage="false"
      :itemTypeData="itemTypeData"
      :httpRequest="httpRequest"
      class="ele-pro-form-builder-body-form"
      @updateValue="setBuilderFormDataFieldValue"
      @update:items="handleUpdateFormItems"
      @update:activeItemKey="handleUpdateCurrentFormItemId"
    >
      <template #builderItemHandleContent="{ item }">
        <ComponentName :itemType="item.type" :componentData="componentData" />
      </template>
      <template #builderItemTools="{ item }">
        <BuilderTools
          :itemType="item.type"
          @delete="handleDeleteItem(item)"
          @copy="handleCopyItem(item)"
          @add="handleAddChildrenItem(item)"
          @addTableRow="handleAddChildrenItem(item, 'addTableRow')"
          @addTableCol="handleAddChildrenItem(item, 'addTableCol')"
          @openTableTool="(e) => handleOpenTableTool(item, e)"
        />
      </template>
      <template
        v-for="name in Object.keys($slots).filter((k) => !ownSlots.includes(k))"
        #[name]="slotProps"
      >
        <slot :name="name" v-bind="slotProps || {}"></slot>
      </template>
    </component>
    <ElEmpty
      v-if="!formProps || !formProps.items || !formProps.items.length"
      :imageSize="80"
      description="拖拽左侧组件到此"
      class="ele-pro-form-builder-form-empty"
    />
  </div>
</template>

<script setup>
  import { ref, reactive, watch } from 'vue';
  import { ElEmpty } from 'element-plus';
  import { eachTree } from '../../utils/common';
  import {
    setValue,
    mergeValue,
    getFormInitValue
  } from '../../ele-pro-form/util';
  import EleProForm from '../../ele-pro-form/index.vue';
  import { generateAddChildData, generateCopyItemData } from './build-util';
  import ComponentName from './component-name.vue';
  import BuilderTools from './builder-tools.vue';
  const ownSlots = ['builderItemHandleContent', 'builderItemTools'];

  defineOptions({ name: 'BodyForm' });

  const props = defineProps({
    /** 表单属性 */
    formProps: Object,
    /** 选中的表单项 id */
    currentFormItemId: [String, Number],
    /** 当前选中屏幕尺寸 */
    currentScreen: String,
    /** 组件库数据 */
    componentData: Array,
    /** 高级表单组件 */
    proFormComponent: [String, Object, Function],
    /** 高级表单组件类型数据 */
    itemTypeData: Array,
    /** 远程数据源请求工具 */
    httpRequest: [Object, Function]
  });

  const emit = defineEmits({
    'update:currentFormItemId': (_formItemId) => true,
    updateItems: (_result) => true,
    openTableTool: (_formItemId, _el) => true,
    updateFormItems: (_items) => true
  });

  /** 表单组件 */
  const proFormRef = ref(null);

  /** 表单数据 */
  const formData = reactive({});

  /** 缓存的表单数据 */
  const cachebuilderFormData = reactive({});

  /** 更新表单项数据 */
  const handleUpdateItems = (result) => {
    emit('updateItems', result);
  };

  /** 更新选中的表单项 */
  const handleUpdateCurrentFormItemId = (formItemId) => {
    emit('update:currentFormItemId', formItemId);
  };

  /** 删除表单项 */
  const handleDeleteItem = (item) => {
    if (item.key != null) {
      handleUpdateItems({
        deleteItemIds: [item.key],
        addItems: [],
        updateItems: []
      });
    }
  };

  /** 表单项添加子级 */
  const handleAddChildrenItem = (formItem, action) => {
    if (formItem.key != null) {
      eachTree(props.formProps?.items, (item, cIndex, parent) => {
        if (item.key === formItem.key) {
          const result = generateAddChildData(
            item,
            parent,
            cIndex,
            action,
            props.formProps?.items,
            void 0,
            props.componentData
          );
          handleUpdateItems(result);
          return false;
        }
      });
    }
  };

  /** 复制表单项 */
  const handleCopyItem = (item) => {
    if (item.key != null) {
      handleUpdateItems(generateCopyItemData(item.key, props.formProps?.items));
    }
  };

  /** 打开表格更多操作 */
  const handleOpenTableTool = (item, e) => {
    if (item.key != null) {
      emit('openTableTool', item.key, e.currentTarget);
    }
  };

  /** 更新表单数据 */
  const setBuilderFormDataFieldValue = (field, value) => {
    setValue(formData, field, value);
    setValue(cachebuilderFormData, field, value);
  };

  /** 拖拽排序更新 */
  const handleUpdateFormItems = (items) => {
    emit('updateFormItems', items);
  };

  /** 选项卡和折叠面板组件自动切换到选中的选项卡子级表单项 */
  watch(
    () => props.currentFormItemId,
    (currentFormItemId) => {
      eachTree(props.formProps?.items, (item, _cIndex, parent) => {
        if (item.key === currentFormItemId) {
          if (
            item.type &&
            ['tabPane', 'collapseItem'].includes(item.type) &&
            parent &&
            parent.prop
          ) {
            setBuilderFormDataFieldValue(parent.prop, item.prop);
          }
          return false;
        }
      });
    }
  );

  /** 同步更新表单项数据 */
  watch(
    () => props.formProps,
    () => {
      Object.keys(formData).forEach((k) => {
        formData[k] = void 0;
      });
      mergeValue(
        formData,
        getFormInitValue(props.formProps?.items, props.itemTypeData, true),
        cachebuilderFormData
      );
    },
    {
      deep: true,
      immediate: true
    }
  );
</script>
