<!-- 选项卡和折叠面板和栅格的子级编辑和拖拽排序 -->
<template>
  <VueDraggable
    v-if="formItem.children"
    itemKey="key"
    :animation="150"
    :setData="() => void 0"
    :modelValue="formItem.children"
    handle=".ele-pro-form-builder-children-edit-item-handle"
    class="ele-pro-form-builder-children-edit-list"
    @update:modelValue="handleUpdateChildren"
  >
    <template #item="{ element }">
      <div class="ele-pro-form-builder-children-edit-item">
        <ElIcon class="ele-pro-form-builder-children-edit-item-handle">
          <DragOutlined />
        </ElIcon>
        <div
          v-if="formItem.type && formItem.type === 'tabs'"
          class="ele-pro-form-builder-children-edit-item-body"
        >
          <ElInput
            size="small"
            :modelValue="element.props?.label"
            @update:modelValue="
              (value) => handleUpdateLabel(value, element, 'props.label')
            "
          />
        </div>
        <div
          v-else-if="formItem.type && formItem.type === 'collapse'"
          class="ele-pro-form-builder-children-edit-item-body"
        >
          <ElInput
            size="small"
            :modelValue="element.props?.title"
            @update:modelValue="
              (value) => handleUpdateLabel(value, element, 'props.title')
            "
          />
        </div>
        <div
          v-else
          class="ele-pro-form-builder-children-edit-item-body is-clickable"
          @click.stop="handleClick(element)"
        >
          <ComponentName
            :itemType="element.type"
            :componentData="componentData"
            class="ele-pro-form-builder-outline-item-type-tag"
          />
          <div class="ele-pro-form-builder-children-edit-item-text">
            {{ element.label || element.prop }}
          </div>
        </div>
        <ElIcon
          class="ele-pro-form-builder-children-edit-item-del-btn"
          title="删除"
          @click.stop="handleDelete(element)"
        >
          <DeleteOutlined />
        </ElIcon>
      </div>
    </template>
  </VueDraggable>
  <ElButton
    size="small"
    :icon="isFixedChildType ? PlusSquareDashOutlined : PlusOutlined"
    class="ele-pro-form-builder-props-fluid-btn is-small-icon"
    @click="handleAdd"
  >
    <template v-if="addBtnText" #default>
      {{ addBtnText }}
    </template>
  </ElButton>
</template>

<script setup>
  import { computed } from 'vue';
  import VueDraggable from 'vuedraggable';
  import { ElIcon, ElInput, ElButton } from 'element-plus';
  import {
    DragOutlined,
    DeleteOutlined,
    PlusOutlined,
    PlusSquareDashOutlined
  } from '../../icons/index';
  import { fixedChildTypes } from './build-core';
  import ComponentName from './component-name.vue';

  defineOptions({ name: 'ChildrenEdit' });

  const props = defineProps({
    /** 添加按钮文本 */
    addBtnText: String,
    /** 表单项数据 */
    formItem: {
      type: Object,
      required: true
    },
    /** 组件库数据 */
    componentData: {
      type: Array,
      required: true
    }
  });

  const emit = defineEmits({
    'update:currentFormItemId': (_formItemId) => true,
    updateChildLabel: (_label, _child, _field) => true,
    sortChildren: (_children) => true,
    deleteChildren: (_child) => true,
    addChildren: (_parent) => true
  });

  /** 是否是有固定的子级的类型 */
  const isFixedChildType = computed(() => {
    const parentType = props.formItem?.type;
    return !!(parentType && fixedChildTypes.some((d) => d.type === parentType));
  });

  /** 更新名称 */
  const handleUpdateLabel = (value, item, field) => {
    emit('updateChildLabel', value, item, field);
  };

  /** 更新数据排序 */
  const handleUpdateChildren = (children) => {
    emit('sortChildren', children);
  };

  /** 删除 */
  const handleDelete = (item) => {
    emit('deleteChildren', item);
  };

  /** 点击选中 */
  const handleClick = (item) => {
    emit('update:currentFormItemId', item.key);
  };

  /** 添加 */
  const handleAdd = () => {
    emit('addChildren', props.formItem);
  };
</script>
