<!-- 表头行 -->
<template>
  <HeaderCell
    v-for="(column, index) in headerCols"
    :key="getKey(column, index)"
    :column="column"
    :columnIndex="index"
    :headerIndex="headerIndex"
    :colSize="headerColSizes ? headerColSizes[index] : void 0"
    :sortBy="sortBy"
    :filterValue="getFiltered(column)"
    :isCheckAll="isCheckAll"
    :isIndeterminate="isIndeterminate"
    :disabledCheckbox="disabledCheckbox"
    :tableSize="tableSize"
    :headerCellClass="headerCellClass"
    :headerCellStyle="headerCellStyle"
    :headerEllipsis="headerEllipsis"
    :rowHeight="rowHeight"
    @checkedChange="handleCellCheckedChange"
    @filterChange="handleCellFilterChange"
    @click="handleCellClick"
    @contextmenu="handleCellContextmenu"
    @mouseenter="handleCellMouseenter"
  >
    <template v-for="name in Object.keys($slots)" #[name]="slotProps">
      <slot :name="name" v-bind="slotProps || {}"></slot>
    </template>
  </HeaderCell>
</template>

<script setup>
  import { getColFiltered } from '../util';
  import HeaderCell from './header-cell.vue';

  defineOptions({ name: 'HeaderRow' });

  const props = defineProps({
    /** 表头行配置 */
    headerCols: Array,
    /** 表头所有行列宽 */
    headerColSizes: Array,
    /** 当前行索引 */
    headerIndex: Number,
    /** 表格排序状态 */
    sortBy: Object,
    /** 表格筛选值 */
    filtered: Object,
    /** 表格是否是全选状态 */
    isCheckAll: Boolean,
    /** 表格是否是半选状态 */
    isIndeterminate: Boolean,
    /** 是否禁用表头选择框 */
    disabledCheckbox: Boolean,
    /** 表格尺寸 */
    tableSize: String,
    /** 表头单元格类名自定义 */
    headerCellClass: [String, Function],
    /** 表头单元样式自定义 */
    headerCellStyle: [Object, Function],
    /** 表头单元格是否溢出省略 */
    headerEllipsis: Boolean,
    /** 表格行高 */
    rowHeight: Number
  });

  const emit = defineEmits({
    /** 多选框选中改变事件 */
    checkedChange: (_checked) => true,
    /** 列筛选改变事件 */
    filterChange: (_params) => true,
    /** 单元格点击事件 */
    cellClick: (_col, _e) => true,
    /** 单元格右键事件 */
    cellContextmenu: (_col, _e) => true,
    /** 单元格鼠标移入事件 */
    cellMouseenter: (_col, _e) => true
  });

  /** 获取列唯一值 */
  const getKey = (column, columnIndex) => {
    return `${props.headerIndex}-${String(column.key ?? columnIndex)}`;
  };

  /** 获取列筛选值 */
  const getFiltered = (column) => {
    return getColFiltered(column.originalCol, props.filtered);
  };

  /** 多选框选中改变事件 */
  const handleCellCheckedChange = (checked) => {
    emit('checkedChange', checked);
  };

  /** 列筛选改变事件 */
  const handleCellFilterChange = (params) => {
    emit('filterChange', params);
  };

  /** 单元格点击事件 */
  const handleCellClick = (col, e) => {
    emit('cellClick', col, e);
  };

  /** 单元格右键事件 */
  const handleCellContextmenu = (col, e) => {
    emit('cellContextmenu', col, e);
  };

  /** 单元格鼠标移入事件 */
  const handleCellMouseenter = (col, e) => {
    emit('cellMouseenter', col, e);
  };
</script>
