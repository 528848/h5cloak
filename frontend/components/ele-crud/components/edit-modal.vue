<!-- 编辑弹窗 -->
<template>
  <EleModal
    :width="460"
    :form="true"
    :title="title"
    :destroyOnClose="true"
    :loading="loading"
    v-bind="editConfig?.modalProps || {}"
    :modelValue="modelValue"
    @update:modelValue="handleUpdateVisible"
  >
    <component
      :itemTypeData="itemTypeData"
      :httpRequest="httpRequest"
      :screenSize="screenSize"
      v-bind="editFormProps"
      :is="proFormComponent || EleProForm"
      ref="editFormRef"
      :model="editForm"
      @updateValue="handleUpdateEditForm"
      @submit="handleSubmitEditForm"
      @reset="handleResetEditForm()"
    >
      <template
        v-for="(slotName, compSlotName) in getSlotsMap(
          $slots,
          editConfig?.formSlots,
          [],
          [],
          true
        )"
        #[compSlotName]="slotProps"
      >
        <slot :name="slotName" v-bind="slotProps || {}"></slot>
      </template>
    </component>
    <template v-if="btnItems.length" #footer>
      <EleButtons :items="btnItems" />
    </template>
    <template
      v-for="(slotName, compSlotName) in getSlotsMap(
        $slots,
        editConfig?.modalSlots,
        ['default', 'footer']
      )"
      #[compSlotName]="slotProps"
    >
      <slot :name="slotName" v-bind="slotProps || {}"></slot>
    </template>
  </EleModal>
</template>

<script setup>
  import { ref, reactive, computed, watch } from 'vue';
  import { getSlotsMap } from '../../utils/common';
  import EleModal from '../../ele-modal/index.vue';
  import EleProForm from '../../ele-pro-form/index.vue';
  import {
    setValue,
    mergeValue,
    getFormInitValue
  } from '../../ele-pro-form/util';
  import EleButtons from '../../ele-buttons/index.vue';
  import { codeStringPrefix } from '../util';

  defineOptions({ name: 'EditModal' });

  const props = defineProps({
    /** 弹窗是否打开 */
    modelValue: Boolean,
    /** 编辑弹窗数据 */
    data: Object,
    /** 弹窗标题 */
    title: String,
    /** 编辑配置 */
    editConfig: Object,
    /** 字段数据 */
    fields: Array,
    /** 获取字段数据对应的表单项的方法 */
    getFieldsEditFormItems: Function,
    /** 获取字段数据对应的表单项的方法 */
    getAndCacheCode: {
      type: Function,
      required: true
    },
    /** 高级表单组件 */
    proFormComponent: [String, Object, Function],
    /** 修改数据接口 */
    editApi: [Function, String],
    /** 高级表单组件类型数据 */
    itemTypeData: Array,
    /** 远程数据源请求工具 */
    httpRequest: [Object, Function],
    /** 屏幕尺寸 */
    screenSize: String
  });

  const emit = defineEmits({
    /** 更新弹窗打开状态事件 */
    'update:modelValue': (_visible) => true,
    /** 编辑保存失败事件 */
    editError: (_error) => true,
    /** 编辑保存成功事件 */
    editDone: (_msg) => true
  });

  /** 弹窗保存按钮加载状态 */
  const loading = ref(false);

  /** 编辑表单组件 */
  const editFormRef = ref(null);

  /** 编辑表单数据 */
  const editForm = reactive({});

  /** 编辑表单属性 */
  const editFormProps = computed(() => {
    const formProps = props.editConfig?.formProps || {};
    return {
      ...formProps,
      items: formProps.items?.length
        ? formProps.items
        : props.getFieldsEditFormItems?.(props.fields)
    };
  });

  /** 更新弹窗打开状态 */
  const handleUpdateVisible = (visible) => {
    emit('update:modelValue', visible);
  };

  /** 保存编辑 */
  const handleSave = () => {
    editFormRef.value?.validate?.((valid) => {
      if (valid) {
        handleSubmitEditForm(editForm);
      }
    });
  };

  /** 关闭弹窗 */
  const handleCancel = () => {
    handleUpdateVisible(false);
    loading.value = false;
  };

  /** 更新编辑表单数据 */
  const handleUpdateEditForm = (field, value) => {
    setValue(editForm, field, value);
  };

  /** 重置编辑表单数据 */
  const handleResetEditForm = (item) => {
    Object.keys(editForm).forEach((key) => {
      editForm[key] = void 0;
    });
    mergeValue(
      editForm,
      getFormInitValue(
        editFormProps.value.items,
        editFormProps.value.itemTypeData || props.itemTypeData,
        true
      ),
      item
    );
  };

  /** 提交编辑表单 */
  const handleSubmitEditForm = (form) => {
    if (!props.editApi) {
      return;
    }
    const editApiFunc =
      typeof props.editApi === 'function'
        ? props.editApi
        : props.editApi.startsWith(codeStringPrefix)
          ? props.getAndCacheCode(props.editApi)
          : void 0;
    if (!editApiFunc) {
      return;
    }
    loading.value = true;
    editApiFunc(form)
      .then((msg) => {
        handleCancel();
        emit('editDone', msg);
      })
      .catch((e) => {
        loading.value = false;
        emit('editError', e);
      });
  };

  /** 操作按钮 */
  const btnItems = computed(() => {
    const items = [];
    if (props.editConfig?.cancelBtnProps !== false) {
      items.push({
        preset: 'cancel',
        props:
          props.editConfig?.cancelBtnProps === true
            ? void 0
            : props.editConfig?.cancelBtnProps,
        onClick: handleCancel
      });
    }
    if (props.editConfig?.saveBtnProps !== false) {
      items.push({
        preset: 'save',
        props: {
          loading: loading.value,
          ...((props.editConfig?.saveBtnProps === true
            ? void 0
            : props.editConfig?.saveBtnProps) || {})
        },
        onClick: handleSave
      });
    }
    return items;
  });

  /** 监听弹窗打开 */
  watch(
    () => props.modelValue,
    (visible) => {
      if (visible) {
        handleResetEditForm(props.data);
      }
    }
  );
</script>
