<!-- 属性设置表单 -->
<template>
  <component
    :is="proFormComponent || EleProForm"
    ref="propsProFormRef"
    size="small"
    labelPosition="top"
    :model="propsFormData"
    :items="propsFormItems"
    :itemTypeData="itemTypeData"
    :httpRequest="httpRequest"
    class="ele-pro-form-builder-props-form"
    @updateValue="handleUpdateItem"
  >
    <template
      v-if="!$slots.proFormBuilderOptionsEdit"
      #proFormBuilderOptionsEdit="{ item, modelValue, updateValue }"
    >
      <OptionsEdit
        :codeEditerComponent="codeEditerComponent"
        v-bind="item.props || {}"
        :modelValue="modelValue"
        @update:modelValue="updateValue"
      />
    </template>
    <template
      v-if="!$slots.proFormBuilderEventEdit"
      #proFormBuilderEventEdit="{ item, modelValue, updateValue }"
    >
      <EventEdit
        :codeEditerComponent="codeEditerComponent"
        v-bind="item.props || {}"
        :modelValue="modelValue"
        @update:modelValue="updateValue"
      />
    </template>
    <template
      v-if="!$slots.proFormBuilderHtmlEdit"
      #proFormBuilderHtmlEdit="{ item, modelValue, updateValue }"
    >
      <HtmlEdit
        :htmlEditerComponent="htmlEditerComponent"
        v-bind="item.props || {}"
        :modelValue="modelValue"
        @update:modelValue="updateValue"
      />
    </template>
    <template
      v-if="!$slots.proFormBuilderIfEdit"
      #proFormBuilderIfEdit="{ item, modelValue, updateValue }"
    >
      <IfEdit
        :codeEditerComponent="codeEditerComponent"
        v-bind="item.props || {}"
        :modelValue="modelValue"
        @update:modelValue="updateValue"
      />
    </template>
    <template
      v-if="!$slots.proFormBuilderSourceEdit"
      #proFormBuilderSourceEdit="{ item, model, updatePropValue }"
    >
      <SourceEdit
        :jsonEditerComponent="jsonEditerComponent"
        v-bind="item.props || {}"
        :modelValue="model"
        @update:modelValue="(val) => updatePropValue('', val)"
      />
    </template>
    <template v-if="!$slots.proFormBuilderTypeEdit" #proFormBuilderTypeEdit>
      <TypeEdit
        :key="currentFormItemId"
        :formItem="currentFormItem"
        :componentData="componentData"
        @openComponentPicker="handleOpenComponentPicker"
      />
    </template>
    <template
      v-if="!$slots.proFormBuilderChildrenEdit"
      #proFormBuilderChildrenEdit="{ item }"
    >
      <ChildrenEdit
        :key="currentFormItemId"
        v-bind="item.props || {}"
        :formItem="currentFormItem"
        :componentData="componentData"
        @updateChildLabel="handleUpdateChildLabel"
        @sortChildren="handleSortChildren"
        @deleteChildren="handleDeleteChildren"
        @addChildren="handleAddChildren"
        @update:currentFormItemId="handleUpdateCurrentFormItemId"
      />
    </template>
    <template
      v-if="!$slots.proFormBuilderStyleEdit"
      #proFormBuilderStyleEdit="{ item, modelValue, updateValue }"
    >
      <StyleEdit
        :key="currentFormItemId"
        v-bind="item.props || {}"
        :modelValue="modelValue"
        @update:modelValue="updateValue"
      />
    </template>
    <template
      v-if="!$slots.proFormBuilderJsonInput"
      #proFormBuilderJsonInput="{ item, modelValue, updateValue }"
    >
      <JsonInput
        :key="currentFormItemId"
        v-bind="item.props || {}"
        :modelValue="modelValue"
        @update:modelValue="updateValue"
      />
    </template>
    <template
      v-if="!$slots.proFormBuilderIconInput"
      #proFormBuilderIconInput="{ modelValue, updateValue }"
    >
      <ElInput
        size="small"
        :clearable="true"
        :modelValue="modelValue"
        @update:modelValue="updateValue"
      />
    </template>
    <template v-for="name in Object.keys($slots)" #[name]="slotProps">
      <slot :name="name" v-bind="slotProps || {}"></slot>
    </template>
  </component>
  <ElEmpty
    v-if="!currentFormItem"
    :imageSize="58"
    description="选中组件配置属性"
    class="ele-pro-form-builder-form-empty"
  />
</template>

<script setup>
  import { ref, reactive, watch } from 'vue';
  import { ElEmpty, ElInput } from 'element-plus';
  import { findTree } from '../../utils/common';
  import EleProForm from '../../ele-pro-form/index.vue';
  import { getFormDataAndItems, fixedChildTypes } from './build-core';
  import { generateAddChildData } from './build-util';
  import OptionsEdit from './options-edit.vue';
  import EventEdit from './event-edit.vue';
  import HtmlEdit from './html-edit.vue';
  import IfEdit from './if-edit.vue';
  import SourceEdit from './source-edit.vue';
  import TypeEdit from './type-edit.vue';
  import ChildrenEdit from './children-edit.vue';
  import StyleEdit from './style-edit.vue';
  import JsonInput from './json-input.vue';

  defineOptions({ name: 'PropsForm' });

  const props = defineProps({
    /** 选中的表单项 id */
    currentFormItemId: [String, Number],
    /** 表单属性 */
    formProps: Object,
    /** 表单属性设置的组件预设属性值 */
    configFormPresetProps: Object,
    /** 组件库数据 */
    componentData: Array,
    /** 高级表单组件 */
    proFormComponent: [String, Object, Function],
    /** 代码编辑器组件 */
    codeEditerComponent: [String, Object, Function],
    /** JSON 编辑器组件 */
    jsonEditerComponent: [String, Object, Function],
    /** 富文本编辑器组件 */
    htmlEditerComponent: [String, Object, Function],
    /** 高级表单组件类型数据 */
    itemTypeData: Array,
    /** 远程数据源请求工具 */
    httpRequest: [Object, Function]
  });

  const emit = defineEmits({
    'update:currentFormItemId': (_formItemId) => true,
    updateItem: (_formItemId, _field, _value) => true,
    updateItems: (_result) => true,
    sortItemChildren: (_childIds, _formItemId) => true,
    openComponentPicker: (_formItemId, _formItemType) => true
  });

  /** 表单组件 */
  const propsProFormRef = ref(null);

  /** 当前选中的表单项数据 */
  const currentFormItem = ref();

  /** 当前选中的表单项对应的属性设置表单数据 */
  const propsFormData = reactive({});

  /** 当前选中的表单项对应的属性设置表单项配置 */
  const propsFormItems = ref([]);

  /** 更新表单项 */
  const handleUpdateItem = (field, value) => {
    if (props.currentFormItemId != null) {
      emit('updateItem', props.currentFormItemId, field, value);
    }
  };

  /** 更新表单项子级名称 */
  const handleUpdateChildLabel = (label, child, field) => {
    emit('updateItem', child.key, field, label);
  };

  /** 更新表单项子级顺序 */
  const handleSortChildren = (children) => {
    if (props.currentFormItemId != null) {
      emit(
        'sortItemChildren',
        children.map((c) => c.key),
        props.currentFormItemId
      );
    }
  };

  /** 表单项删除子级 */
  const handleDeleteChildren = (child) => {
    emit('updateItems', {
      addItems: [],
      updateItems: [],
      deleteItemIds: [child.key]
    });
  };

  /** 打开组件选择弹窗 */
  const handleOpenComponentPicker = (item, isAdd) => {
    emit('openComponentPicker', item.key, isAdd ? void 0 : item.type);
  };

  /** 表单项添加子级 */
  const handleAddChildren = (parent) => {
    if (parent.type && fixedChildTypes.some((d) => d.type === parent.type)) {
      const result = generateAddChildData(
        parent,
        void 0,
        void 0,
        void 0,
        props.formProps?.items,
        void 0,
        props.componentData
      );
      emit('updateItems', result);
    } else {
      handleOpenComponentPicker(parent, true);
    }
  };

  /** 选中表单项子级 */
  const handleUpdateCurrentFormItemId = (formItemId) => {
    emit('update:currentFormItemId', formItemId);
  };

  /** 重置属性表单数据 */
  const resetPropsFormData = (data) => {
    Object.keys(propsFormData).forEach((k) => {
      propsFormData[k] = void 0;
    });
    if (data) {
      Object.assign(propsFormData, data);
    }
  };

  /** 同步选中表单项更新 */
  watch(
    [() => props.formProps, () => props.currentFormItemId],
    () => {
      if (props.currentFormItemId == null) {
        resetPropsFormData();
        propsFormItems.value = [];
        currentFormItem.value = void 0;
        return;
      }
      const temp = findTree(
        props.formProps?.items,
        (item) => item.key === props.currentFormItemId
      );
      const { data, items } = getFormDataAndItems(temp, props.componentData);
      resetPropsFormData(
        Object.assign({}, data, {
          itemProps: Object.assign(
            {
              labelWidth:
                props.formProps?.labelWidth ??
                props.configFormPresetProps?.labelWidth,
              labelPosition:
                props.formProps?.labelPosition ??
                props.configFormPresetProps?.labelPosition
            },
            data.itemProps
          )
        })
      );
      if (JSON.stringify(propsFormItems.value) !== JSON.stringify(items)) {
        propsFormItems.value = items;
      }
      if (currentFormItem.value !== temp) {
        currentFormItem.value = temp;
        if (propsProFormRef.value) {
          propsProFormRef.value.$el.scrollTop = 0;
        }
      }
    },
    {
      immediate: true,
      deep: true
    }
  );
</script>
