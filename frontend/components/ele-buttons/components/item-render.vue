<template>
  <ElDivider
    v-if="divider"
    direction="vertical"
    class="ele-buttons-divider"
    v-bind="typeof divider === 'object' ? divider : {}"
  />
  <ElLink
    v-if="(item.type ?? type) === 'link'"
    underline="never"
    :type="linkType"
    :icon="icon"
    class="ele-buttons-item ele-buttons-link"
    :class="{
      'ele-buttons-icon-link': !!icon,
      'ele-buttons-wrap': wrapClass
    }"
    v-bind="item.props || {}"
    @click="(e) => handleItemClick(e, item)"
    @mouseenter="(e) => handleOpenDropdown(e, item)"
  >
    <span>{{ title }}</span>
    <ElIcon
      v-if="item.dropdownItems && item.dropdownItems.length"
      class="ele-buttons-arrow"
    >
      <ArrowDown />
    </ElIcon>
  </ElLink>
  <ElButton
    v-else
    :type="btnType"
    :icon="icon"
    class="ele-buttons-item ele-buttons-btn"
    :class="{
      'ele-buttons-icon-btn': !!icon,
      'ele-buttons-wrap': wrapClass
    }"
    v-bind="item.props || {}"
    @click="(e) => handleItemClick(e, item)"
    @mouseenter="(e) => handleOpenDropdown(e, item)"
  >
    <span>{{ title }}</span>
    <ElIcon
      v-if="item.dropdownItems && item.dropdownItems.length"
      class="ele-buttons-arrow"
    >
      <ArrowDown />
    </ElIcon>
  </ElButton>
</template>

<script setup>
  import { computed } from 'vue';
  import { ElDivider, ElLink, ElIcon, ElButton } from 'element-plus';
  import {
    ArrowDown,
    ArrowUp,
    SearchOutlined,
    SyncOutlined,
    PlusOutlined,
    DeleteOutlined,
    EditOutlined,
    UploadOutlined,
    DownloadOutlined,
    ColumnHeightOutlined,
    VerticalAlignMiddleOutlined,
    EyeOutlined
  } from '../../icons/index';
  import { mapTree } from '../../utils/common';
  import { useDropdown } from '../../ele-dropdown-provider/util';
  import { usePopconfirm } from '../../ele-popconfirm-provider/util';
  const presetIcon = {
    search: SearchOutlined,
    reset: SyncOutlined,
    add: PlusOutlined,
    delBatch: DeleteOutlined,
    edit: EditOutlined,
    del: DeleteOutlined,
    import: UploadOutlined,
    export: DownloadOutlined,
    upload: UploadOutlined,
    download: DownloadOutlined,
    expand: ColumnHeightOutlined,
    fold: VerticalAlignMiddleOutlined,
    searchExpand: ArrowDown,
    searchFold: ArrowUp,
    detail: EyeOutlined
  };
  const dangerLinkPreset = ['del', 'delBatch'];
  const dangerBtnPreset = ['del', 'delBatch'];
  const warningBtnPreset = ['edit'];
  const defaultBtnPreset = [
    'reset',
    'import',
    'export',
    'expand',
    'fold',
    'detail',
    'more',
    'cancel'
  ];

  defineOptions({ name: 'ItemRender' });

  const props = defineProps({
    /** 按钮数据 */
    item: {
      type: Object,
      required: true
    },
    /** 是否显示分割线 */
    divider: [Boolean, Object],
    /** 组件类型 */
    type: String,
    /** 是否增加包裹类名 */
    wrapClass: Boolean,
    /** 文案 */
    lang: {
      type: Object,
      required: true
    }
  });

  const emit = defineEmits({
    itemClick: (_command, _e) => true
  });

  /** 下拉菜单操作 */
  const dropdown = useDropdown();

  /** 气泡确认框操作 */
  const popconfirm = usePopconfirm();

  /** 获取文本 */
  const getItemTitle = (item, lang) => {
    if (item.title != null) {
      return item.title;
    }
    if (!item.preset) {
      return;
    }
    return lang[item.preset];
  };

  /** 获取图标 */
  const getItemIcon = (item) => {
    if (item.icon != null) {
      return item.icon;
    }
    if (!item.preset) {
      return;
    }
    return presetIcon[item.preset];
  };

  /** 文本 */
  const title = computed(() => {
    return getItemTitle(props.item, props.lang);
  });

  /** 图标 */
  const icon = computed(() => {
    return getItemIcon(props.item);
  });

  /** 链接颜色类型 */
  const linkType = computed(() => {
    const preset = props.item.preset;
    if (preset && dangerLinkPreset.includes(preset)) {
      return 'danger';
    }
    return 'primary';
  });

  /** 按钮颜色类型 */
  const btnType = computed(() => {
    const preset = props.item.preset;
    if (preset) {
      if (dangerBtnPreset.includes(preset)) {
        return 'danger';
      }
      if (warningBtnPreset.includes(preset)) {
        return 'warning';
      }
      if (defaultBtnPreset.includes(preset)) {
        return 'default';
      }
    }
    return 'primary';
  });

  /** 触发操作项点击事件 */
  const triggerItemClick = (command, e) => {
    emit('itemClick', command, e);
  };

  /** 下拉菜单点击事件 */
  const handleDropdownCommand = (command) => {
    triggerItemClick(command);
  };

  /** 打开下拉菜单 */
  const handleOpenDropdown = (e, item) => {
    if (
      !item.dropdownItems ||
      !item.dropdownItems.length ||
      !dropdown ||
      !dropdown.openDropdown
    ) {
      return;
    }
    dropdown.openDropdown(
      e.currentTarget,
      mapTree(item.dropdownItems, (d) => {
        return {
          ...d,
          title: getItemTitle(d, props.lang),
          icon: getItemIcon(d)
        };
      }),
      {
        iconSize: 'small',
        onCommand: handleDropdownCommand,
        popperOptions: { strategy: 'fixed' },
        ...(item.dropdownProps || {}),
        trigger: 'hover'
      }
    );
  };

  /** 打开气泡确认框 */
  const handleOpenPopconfirm = (e, item) => {
    if (!item.popconfirmProps) {
      return;
    }
    if (!popconfirm || !popconfirm.openPopconfirm) {
      // 降级为直接触发确认事件
      const onConfirm = item.popconfirmProps.onConfirm;
      onConfirm && onConfirm(e);
      return;
    }
    popconfirm.openPopconfirm(e.currentTarget, {
      popperOptions: { strategy: 'fixed' },
      ...(item.popconfirmProps || {}),
      trigger: 'click'
    });
  };

  /** 点击事件 */
  const handleItemClick = (e, item) => {
    if (item.popconfirmProps) {
      handleOpenPopconfirm(e, item);
    }
    item.onClick && item.onClick(e);
    triggerItemClick(item.command, e);
  };
</script>
