<!-- 表单设置表单 -->
<template>
  <component
    :is="proFormComponent || EleProForm"
    size="small"
    labelPosition="top"
    :model="configFormData"
    :items="configFormItems || []"
    :itemTypeData="itemTypeData"
    :httpRequest="httpRequest"
    class="ele-pro-form-builder-props-form"
    @updateValue="handleUpdateFormProp"
  >
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
    <template
      v-if="!$slots.proFormBuilderStyleEdit"
      #proFormBuilderStyleEdit="{ item, modelValue, updateValue }"
    >
      <StyleEdit
        v-bind="item.props || {}"
        :modelValue="modelValue"
        @update:modelValue="updateValue"
      />
    </template>
    <template v-for="name in Object.keys($slots)" #[name]="slotProps">
      <slot :name="name" v-bind="slotProps || {}"></slot>
    </template>
  </component>
</template>

<script setup>
  import { reactive, watch } from 'vue';
  import EleProForm from '../../ele-pro-form/index.vue';
  import { deepCloneObject } from './build-core';
  import SourceEdit from './source-edit.vue';
  import StyleEdit from './style-edit.vue';

  defineOptions({ name: 'ConfigForm' });

  const props = defineProps({
    /** 表单属性 */
    formProps: Object,
    /** 表单属性设置的表单项配置 */
    configFormItems: Array,
    /** 表单属性设置的组件预设属性值 */
    configFormPresetProps: Object,
    /** 高级表单组件 */
    proFormComponent: [String, Object, Function],
    /** JSON 编辑器组件 */
    jsonEditerComponent: [String, Object, Function],
    /** 高级表单组件类型数据 */
    itemTypeData: Array,
    /** 远程数据源请求工具 */
    httpRequest: [Object, Function]
  });

  const emit = defineEmits({
    updateFormProp: (_field, _value) => true
  });

  /** 表单属性数据 */
  const configFormData = reactive({});

  /** 更新表单属性 */
  const handleUpdateFormProp = (field, value) => {
    emit('updateFormProp', field, value);
  };

  /** 同步表单属性数据 */
  watch(
    () => props.formProps,
    (formProps) => {
      const data = Object.assign(
        {},
        configFormData,
        props.configFormPresetProps || {},
        deepCloneObject(formProps),
        { items: void 0 }
      );
      Object.assign(configFormData, data, {
        items: void 0,
        footerProps: Object.assign(
          { labelWidth: data.footerProps?.labelWidth ?? data.labelWidth },
          data.footerProps
        )
      });
    },
    {
      immediate: true,
      deep: true
    }
  );
</script>
