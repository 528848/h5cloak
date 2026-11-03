<!-- 默认工具按钮 -->
<template>
  <template v-for="(tool, index) in tools">
    <!-- 刷新 -->
    <EleTool
      v-if="tool === 'reload'"
      :key="index + '-reload'"
      :placement="placement"
      :title="lang.refresh"
      @click="reload"
    >
      <ElIcon>
        <ReloadOutlined />
      </ElIcon>
    </EleTool>
    <!-- 导出 -->
    <EleTool
      v-else-if="tool === 'export'"
      :key="index + '-export'"
      :title="lang.export"
      :placement="placement"
      :clickHideTooltip="true"
      @click="openExportModal"
    >
      <ElIcon :style="{ transform: 'scale(1.1)', transformOrigin: 'bottom' }">
        <DownloadOutlined />
      </ElIcon>
    </EleTool>
    <!-- 打印 -->
    <EleTool
      v-else-if="tool === 'print'"
      :key="index + '-print'"
      :title="lang.print"
      :placement="placement"
      :clickHideTooltip="true"
      @click="openPrintModal"
    >
      <ElIcon>
        <PrinterOutlined />
      </ElIcon>
    </EleTool>
    <!-- 密度设置 -->
    <EleTool
      v-else-if="tool === 'size'"
      :key="index + '-size'"
      :placement="placement"
      :title="lang.sizes"
    >
      <EleDropdown
        trigger="click"
        placement="bottom"
        :validateEvent="false"
        popperClass="ele-tool-size-popper"
        :modelValue="size"
        :items="sizeDropdownItems"
        :popperOptions="{
          modifiers: [{ name: 'offset', options: { offset: [0, 10] } }]
        }"
        @command="updateSize"
      >
        <ElIcon>
          <ColumnHeightOutlined />
        </ElIcon>
      </EleDropdown>
    </EleTool>
    <!-- 列设置 -->
    <ToolColumn
      v-else-if="tool === 'columns'"
      :key="index + '-columns'"
      :placement="placement"
      :locale="lang"
      :columns="columns"
      :sortable="columnSortable"
      :allowFixed="columnFixed"
      :cacheKey="cacheKey"
      @update:columns="updateColumns"
    />
    <!-- 全屏切换 -->
    <EleTool
      v-else-if="tool === 'maximized'"
      :key="index + '-maximized'"
      :placement="placement"
      :title="lang.maximized"
      :clickHideTooltip="true"
      @click="toggleMaximized"
    >
      <ElIcon>
        <FullscreenExitOutlined v-if="maximized" />
        <FullscreenOutlined v-else />
      </ElIcon>
    </EleTool>
    <!-- 自定义 -->
    <template v-else-if="tool && !ownSlots.includes(tool) && $slots[tool]">
      <slot :name="tool"></slot>
    </template>
  </template>
</template>

<script setup>
  import { computed } from 'vue';
  import { ElIcon } from 'element-plus';
  import {
    ReloadOutlined,
    DownloadOutlined,
    PrinterOutlined,
    ColumnHeightOutlined,
    SizeSlackOutlined,
    SizeMiddleOutlined,
    SizeCompactOutlined,
    FullscreenOutlined,
    FullscreenExitOutlined
  } from '../../icons/index';
  import EleDropdown from '../../ele-dropdown/index.vue';
  import { getSizeCacheKey } from '../util';
  import EleTool from '../../ele-tool/index.vue';
  import ToolColumn from './tool-column.vue';
  import { markRaw } from 'vue';
  const ownSlots = ['default', 'printTop', 'printBottom'];

  const props = defineProps({
    /** 工具按钮布局 */
    tools: {
      type: Array,
      required: true
    },
    /** 表格尺寸 */
    size: String,
    /** 表格列数据 */
    columns: Array,
    /** 是否开启列拖拽排序 */
    columnSortable: Boolean,
    /** 是否开启开关固定列 */
    columnFixed: Boolean,
    /** 是否最大化 */
    maximized: Boolean,
    /** 本地缓存的名称 */
    cacheKey: String,
    /** 国际化 */
    lang: {
      type: Object,
      required: true
    }
  });

  const emit = defineEmits({
    reload: () => true,
    'update:size': (_size) => true,
    'update:columns': (_columns, _tableColumns, _isReset) => true,
    'update:maximized': (_maximized) => true,
    openExportModal: () => true,
    openPrintModal: () => true
  });

  /** 提示方向 */
  const placement = computed(() => (props.maximized ? 'bottom' : 'top'));

  /** 尺寸下拉选择数据 */
  const sizeDropdownItems = computed(() => {
    return [
      {
        title: props.lang.sizeLarge,
        command: 'large',
        icon: markRaw(SizeSlackOutlined)
      },
      {
        title: props.lang.sizeDefault,
        command: 'default',
        icon: markRaw(SizeMiddleOutlined)
      },
      {
        title: props.lang.sizeSmall,
        command: 'small',
        icon: markRaw(SizeCompactOutlined)
      }
    ];
  });

  /** 刷新表格 */
  const reload = () => {
    emit('reload');
  };

  /** 修改表格尺寸 */
  const updateSize = (size) => {
    if (props.cacheKey) {
      localStorage.setItem(getSizeCacheKey(props.cacheKey), size);
    }
    emit('update:size', size);
  };

  /** 修改列配置 */
  const updateColumns = (columns, tableColumns, isReset) => {
    emit('update:columns', columns, tableColumns, isReset);
  };

  /** 全屏切换 */
  const toggleMaximized = () => {
    emit('update:maximized', !props.maximized);
  };

  /** 打开导出弹窗 */
  const openExportModal = () => {
    emit('openExportModal');
  };

  /** 打开打印弹窗 */
  const openPrintModal = () => {
    emit('openPrintModal');
  };
</script>
