<!-- 列表额外功能 -->
<template>
  <EditModal
    v-if="addConfig !== false"
    :data="addData"
    :modelValue="addVisible"
    :title="lang.add"
    :editConfig="addConfig === true ? {} : addConfig"
    :fields="fields"
    :getFieldsEditFormItems="getFieldsAddFormItems"
    :getAndCacheCode="getAndCacheCode"
    :proFormComponent="proFormComponent"
    :editApi="addApi"
    :itemTypeData="itemTypeData"
    :httpRequest="httpRequest"
    :screenSize="screenSize"
    @editError="handleAddError"
    @editDone="handleAddDone"
    @update:modelValue="handleUpdateAddVisible"
  >
    <template
      v-for="name in Object.keys($slots).filter(
        (k) => !slotExcludes.includes(k)
      )"
      #[name]="slotProps"
    >
      <slot :name="name" v-bind="slotProps || void 0"></slot>
    </template>
  </EditModal>
  <EditModal
    v-if="editConfig !== false"
    :data="editData"
    :modelValue="editVisible"
    :title="lang.edit"
    :editConfig="editConfig === true ? {} : editConfig"
    :fields="fields"
    :getFieldsEditFormItems="getFieldsEditFormItems"
    :getAndCacheCode="getAndCacheCode"
    :proFormComponent="proFormComponent"
    :editApi="editApi"
    :itemTypeData="itemTypeData"
    :httpRequest="httpRequest"
    :screenSize="screenSize"
    @editError="handleEditError"
    @editDone="handleEditDone"
    @update:modelValue="handleUpdateEditVisible"
  >
    <template
      v-for="name in Object.keys($slots).filter(
        (k) => !slotExcludes.includes(k)
      )"
      #[name]="slotProps"
    >
      <slot :name="name" v-bind="slotProps || {}"></slot>
    </template>
  </EditModal>
  <slot></slot>
</template>

<script setup>
  import { getFieldsAddFormItems, getFieldsEditFormItems } from '../util';
  import EditModal from './edit-modal.vue';
  const slotExcludes = ['default'];

  defineOptions({ name: 'TableExtra' });

  defineProps({
    /** 添加弹窗是否打开 */
    addVisible: Boolean,
    /** 添加弹窗数据 */
    addData: Object,
    /** 修改弹窗是否打开 */
    editVisible: Boolean,
    /** 修改弹窗数据 */
    editData: Object,
    /** 添加配置 */
    addConfig: [Object, Boolean],
    /** 修改配置 */
    editConfig: [Object, Boolean],
    /** 字段数据 */
    fields: Array,
    /** 获取字段数据对应的表单项的方法 */
    getAndCacheCode: {
      type: Function,
      required: true
    },
    /** 高级表单组件 */
    proFormComponent: [String, Object, Function],
    /** 添加数据接口 */
    addApi: [Function, String],
    /** 修改数据接口 */
    editApi: [Function, String],
    /** 高级表单组件类型数据 */
    itemTypeData: Array,
    /** 远程数据源请求工具 */
    httpRequest: [Object, Function],
    /** 屏幕尺寸 */
    screenSize: String,
    /** 国际化 */
    lang: {
      type: Object,
      required: true
    }
  });

  const emit = defineEmits({
    /** 更新添加弹窗打开状态事件 */
    'update:addVisible': (_visible) => true,
    /** 更新修改弹窗打开状态事件 */
    'update:editVisible': (_visible) => true,
    /** 操作按钮点击事件 */
    btnClick: (_action, _e, _item) => true,
    /** 添加保存失败事件 */
    addError: (_e) => true,
    /** 添加保存成功事件 */
    addDone: (_msg) => true,
    /** 修改保存失败事件 */
    editError: (_e) => true,
    /** 修改保存成功事件 */
    editDone: (_msg) => true
  });

  /** 添加保存失败事件 */
  const handleAddError = (e) => {
    emit('addError', e);
  };

  /** 添加保存成功事件 */
  const handleAddDone = (msg) => {
    emit('addDone', msg);
  };

  /** 修改保存失败事件 */
  const handleEditError = (e) => {
    emit('editError', e);
  };

  /** 修改保存成功事件 */
  const handleEditDone = (msg) => {
    emit('editDone', msg);
  };

  /** 更新添加弹窗打开状态 */
  const handleUpdateAddVisible = (visible) => {
    emit('update:addVisible', visible);
  };

  /** 更新修改弹窗打开状态 */
  const handleUpdateEditVisible = (visible) => {
    emit('update:editVisible', visible);
  };
</script>
