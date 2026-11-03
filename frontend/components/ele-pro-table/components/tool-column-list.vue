<template>
  <VueDraggable
    itemKey="uid"
    :animation="300"
    :modelValue="data"
    :setData="() => void 0"
    handle=".ele-tool-column-handle"
    :componentData="{ class: 'ele-tool-column-list' }"
    :forceFallback="true"
    @update:modelValue="handleSortChange"
  >
    <template #item="{ element: d }">
      <div class="ele-tool-column-item">
        <div class="ele-tool-column-item-body">
          <div v-if="sortable" class="ele-tool-column-handle">
            <ElIcon>
              <HolderOutlined />
            </ElIcon>
          </div>
          <div class="ele-tool-column-label">
            <ElCheckbox
              :label="d.label"
              :modelValue="d.checked"
              @update:modelValue="(v) => handleCheckedChange(d, v)"
            />
          </div>
          <div v-if="allowFixed" class="ele-tool-column-fixed">
            <div
              class="ele-tool-column-fixed-item"
              :class="{ 'is-active': d.fixed === true || d.fixed === 'left' }"
              @click="handleFixedLeft(d)"
              @mouseover="handleFixedLeftTooltip"
            >
              <ElIcon>
                <VerticalRightOutlined />
              </ElIcon>
            </div>
            <div
              class="ele-tool-column-fixed-item"
              :class="{ 'is-active': d.fixed === 'right' }"
              @click="handleFixedRight(d)"
              @mouseover="handleFixedRightTooltip"
            >
              <ElIcon>
                <VerticalLeftOutlined />
              </ElIcon>
            </div>
          </div>
          <div v-if="allowWidth" class="ele-tool-column-input">
            <ElInput
              size="small"
              :maxlength="12"
              :modelValue="d.width"
              :placeholder="columnWidthPlaceholder"
              @update:modelValue="(v) => handleColWidthChange(d, v)"
            />
          </div>
        </div>
        <ToolColumnList
          v-if="d.children && d.children.length"
          :data="d.children"
          :parent="d"
          :sortable="sortable"
          :allowWidth="allowWidth"
          :columnWidthPlaceholder="columnWidthPlaceholder"
          @sortChange="handleChildSortChange"
          @checkedChange="handleCheckedChange"
          @fixedLeft="handleFixedLeft"
          @fixedRight="handleFixedRight"
          @fixedLeftTooltip="handleChildFixedLeftTooltip"
          @fixedRightTooltip="handleChildFixedRightTooltip"
          @colWidthChange="handleColWidthChange"
        />
      </div>
    </template>
  </VueDraggable>
</template>

<script setup>
  import { inject } from 'vue';
  import VueDraggable from 'vuedraggable';
  import { ElCheckbox, ElIcon, ElInput } from 'element-plus';
  import {
    HolderOutlined,
    VerticalRightOutlined,
    VerticalLeftOutlined
  } from '../../icons/index';
  import { modalItemContextKey } from '../../utils/hook';

  defineOptions({ name: 'ToolColumnList' });

  const props = defineProps({
    /** 列配置数据 */
    data: Array,
    /** 父级数据 */
    parent: Object,
    /** 是否开启列拖拽排序 */
    sortable: Boolean,
    /** 是否开启开关固定列 */
    allowFixed: Boolean,
    /** 是否开启列宽设置 */
    allowWidth: Boolean,
    /** 列宽输入框提示文本 */
    columnWidthPlaceholder: String
  });

  const emit = defineEmits({
    sortChange: (_colItems, _parent) => true,
    checkedChange: (_item, _checked) => true,
    fixedLeft: (_item) => true,
    fixedRight: (_item) => true,
    fixedLeftTooltip: (_el) => true,
    fixedRightTooltip: (_el) => true,
    colWidthChange: (_item, _width) => true
  });

  const modalItem = inject(modalItemContextKey, null);

  /** 显示固定左侧提示 */
  const handleChildFixedLeftTooltip = (el) => {
    emit('fixedLeftTooltip', el);
  };

  /** 显示固定右侧提示 */
  const handleChildFixedRightTooltip = (el) => {
    emit('fixedRightTooltip', el);
  };

  /** 固定左侧按钮 hover 事件 */
  const handleFixedLeftTooltip = (e) => {
    handleChildFixedLeftTooltip(e.currentTarget);
  };

  /** 固定右侧按钮 hover 事件 */
  const handleFixedRightTooltip = (e) => {
    handleChildFixedRightTooltip(e.currentTarget);
  };

  /** 固定左侧 */
  const handleFixedLeft = (colItem) => {
    if (modalItem?.label) {
      emit('fixedLeft', colItem);
    }
  };

  /** 固定右侧 */
  const handleFixedRight = (colItem) => {
    if (modalItem?.label) {
      emit('fixedRight', colItem);
    }
  };

  /** 拖动顺序改变 */
  const handleChildSortChange = (colItems, parent) => {
    if (modalItem?.label) {
      emit('sortChange', colItems, parent);
    }
  };

  /** 选中改变 */
  const handleCheckedChange = (colItem, checked) => {
    if (modalItem?.label) {
      emit('checkedChange', colItem, checked);
    }
  };

  /** 列宽输入改变 */
  const handleColWidthChange = (item, width) => {
    emit('colWidthChange', item, width);
  };

  /** 拖动顺序改变 */
  const handleSortChange = (colItems) => {
    handleChildSortChange(colItems, props.parent);
  };
</script>
