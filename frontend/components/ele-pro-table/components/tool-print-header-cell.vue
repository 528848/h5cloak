<!-- 打印表格表头单元格 -->
<template>
  <th
    :colspan="col.colspan"
    :rowspan="col.rowspan"
    :style="cellStyle"
    :class="cellClass"
  >
    <CellRender v-bind="renderOpt" />
  </th>
</template>

<script setup>
  import { computed, useSlots } from 'vue';
  import { CellRender } from '../../ele-virtual-table/util';

  defineOptions({ name: 'ToolPrintHeaderCell' });

  const props = defineProps({
    /** 列数据 */
    col: {
      type: Object,
      required: true
    },
    /** 列索引 */
    columnIndex: Number,
    /** 单元格样式 */
    headerCellStyle: [Object, Function],
    /** 单元格类名自定义 */
    headerCellClass: [String, Function]
  });

  const slots = useSlots();

  /** 自定义渲染组件属性 */
  const renderOpt = computed(() => {
    const { text, column } = props.col;
    const params = [{ column, $index: props.columnIndex }];
    const slotName = column
      ? column.printHeaderSlot || column.headerSlot
      : void 0;
    if (column && slotName && typeof slots[slotName] === 'function') {
      return { render: slots[slotName], params };
    }
    return { render: () => text, params };
  });

  /** 自定义方法参数 */
  const cellParam = computed(() => {
    return {
      column: props.col.column,
      columnIndex: props.columnIndex,
      rowIndex: props.col.index
    };
  });

  /** 样式 */
  const cellStyle = computed(() => {
    if (typeof props.headerCellStyle === 'function') {
      if (cellParam.value.column == null) {
        return;
      }
      return props.headerCellStyle(cellParam.value);
    }
    return props.headerCellStyle;
  });

  /** 类名 */
  const cellClass = computed(() => {
    const classes = [];
    const column = cellParam.value.column;
    if (column) {
      // 对齐方式
      const align = column.headerAlign || column.align;
      if (align) {
        classes.push(`is-align-${align}`);
      }
      // 自定义类名
      if (typeof props.headerCellClass === 'function') {
        const temp = props.headerCellClass(cellParam.value);
        if (temp) {
          classes.push(temp);
        }
      } else if (props.headerCellClass) {
        classes.push(props.headerCellClass);
      }
      if (column.labelClassName) {
        classes.push(column.labelClassName);
      }
    }
    return classes.join(' ');
  });
</script>
