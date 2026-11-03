<!-- 表尾行 -->
<template>
  <div
    class="ele-table-foot-tr"
    :style="{ width: (sumWidth || 0) + 'px', height: (rowHeight || 0) + 'px' }"
  >
    <div
      v-for="col in sumCols"
      :key="col.key"
      class="ele-table-td is-ellipsis"
      :class="[
        { 'is-align-left': col.align === 'left' },
        { 'is-align-center': col.align === 'center' },
        { 'is-align-right': col.align === 'right' },
        { 'is-fixed-left': col.fixed === 'left' || col.fixed === true },
        { 'is-fixed-right': col.fixed === 'right' },
        { 'is-fixed-left-last': col.isFixedLeftLast },
        { 'is-fixed-right-first': col.isFixedRightFirst }
      ]"
      :style="{
        width: col.width + 'px',
        left: col.fixedLeft,
        right: col.fixedRight
      }"
      @mouseenter="handleMouseenter"
    >
      <div class="ele-table-cell">
        <CellRender :render="() => col.text" :params="[]" />
      </div>
    </div>
  </div>
</template>

<script setup>
  import { computed } from 'vue';
  import { getSumValue, getUserSums, CellRender } from '../util';

  defineOptions({ name: 'FooterRow' });

  const props = defineProps({
    /** 表格主体列配置 */
    bodyCols: {
      type: Array,
      required: true
    },
    /** 表格数据 */
    tableData: {
      type: Array,
      required: true
    },
    /** 表格主体列宽 */
    colSizes: {
      type: Array,
      required: true
    },
    /** 表格主体所有列合计宽度 */
    sumWidth: Number,
    /** 表格行高 */
    rowHeight: Number,
    /** 合计行文本 */
    sumText: String,
    /** 合计行自定义方法 */
    summaryMethod: Function
  });

  const emit = defineEmits({
    /** 单元格鼠标移入事件 */
    mouseenter: (_e) => true
  });

  /** 表格原始数据 */
  const data = computed(() => {
    return props.tableData.map((row) => row.rowData);
  });

  /** 自定义的合计列 */
  const userSums = computed(() => {
    return getUserSums(props.summaryMethod, props.bodyCols, data.value);
  });

  /** 合计列 */
  const sumCols = computed(() => {
    const sums = userSums.value;
    return props.bodyCols.map((column, i) => {
      const colSize = props.colSizes[i];
      const sumValue = getSumValue(data.value, column.dataKey);
      const col = column.originalCol;
      return {
        key: `${i}-${column.key}`,
        width: (colSize && colSize.width) || 0,
        fixedLeft: colSize ? colSize.fixedLeft : void 0,
        fixedRight: colSize ? colSize.fixedRight : void 0,
        text: sums == null ? (i === 0 ? props.sumText : sumValue) : sums[i],
        align: col ? col.align : void 0,
        fixed: col ? col.fixed : void 0,
        isFixedLeftLast: column.isFixedLeftLast,
        isFixedRightFirst: column.isFixedRightFirst
      };
    });
  });

  /** 单元格鼠标移入事件 */
  const handleMouseenter = (e) => {
    emit('mouseenter', e);
  };
</script>
