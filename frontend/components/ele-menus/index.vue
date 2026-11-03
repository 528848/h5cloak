<!-- 菜单 -->
<template>
  <ElMenu
    v-bind="menuProps"
    ref="menuRef"
    :ellipsis="false"
    :mode="isCompact ? 'vertical' : mode"
    :collapse="isCompact ? true : collapse"
    class="ele-menu"
    :class="[
      { 'ele-menu-dark': theme === 'dark' },
      { 'is-night': theme === 'dark' },
      { 'ele-menu-colorful': colorful },
      { 'is-colorful': colorful },
      { 'is-compact': isCompact },
      { 'is-compact-collapse': isCompact && collapse }
    ]"
    @open="handleOpen"
    @close="handleClose"
    @select="handleSelect"
  >
    <MenuItems
      v-if="menuItems && menuItems.length"
      :items="menuItems"
      :first="true"
      :tipDisabled="collapseTooltipDisabled"
      :parentIsGroup="false"
      :theme="theme"
      :popTheme="popupTheme"
      :colorful="colorful"
      :popupColorful="popupColorful"
      :firstPopClass="firstPopperClass"
      :webkit="isWebkit"
      @itemClick="handleItemClick"
      @itemMouseenter="handleItemMouseenter"
      @itemMouseleave="handleItemMouseleave"
      @parentMouseenter="handleParentMouseenter"
      @parentMouseleave="handleParentMouseleave"
    >
      <template v-for="name in Object.keys($slots)" #[name]="slotProps">
        <slot :name="name" v-bind="slotProps || {}"></slot>
      </template>
    </MenuItems>
    <ElSubMenu
      v-if="moreMenuItems && moreMenuItems.length"
      key="sub-menu-more"
      :teleported="true"
      v-bind="ellipsisProps || {}"
      ref="ellipsisRef"
      index="sub-menu-more"
      :popperClass="
        getPopperClass(
          ellipsisProps?.popperClass,
          theme,
          popupTheme,
          colorful,
          popupColorful,
          firstPopperClass,
          true,
          isWebkit
        )
      "
      class="ele-sub-menu-ellipsis"
    >
      <template #title>
        <ElIcon v-bind="ellipsisProps?.iconProps || {}">
          <component
            :is="ellipsisProps?.icon ?? EllipsisOutlined"
            :style="ellipsisProps?.iconStyle"
          />
        </ElIcon>
      </template>
      <MenuItems
        :items="moreMenuItems"
        :first="false"
        :tipDisabled="collapseTooltipDisabled"
        :parentIsGroup="false"
        :theme="theme"
        :popTheme="popupTheme"
        :colorful="colorful"
        :popupColorful="popupColorful"
        :firstPopClass="firstPopperClass"
        :webkit="isWebkit"
        @itemClick="handleItemClick"
        @itemMouseenter="handleItemMouseenter"
        @itemMouseleave="handleItemMouseleave"
        @parentMouseenter="handleParentMouseenter"
        @parentMouseleave="handleParentMouseleave"
      >
        <template v-for="name in Object.keys($slots)" #[name]="slotProps">
          <slot :name="name" v-bind="slotProps || {}"></slot>
        </template>
      </MenuItems>
    </ElSubMenu>
    <EleTooltip
      v-if="textEllipsisTooltip"
      trigger="click"
      placement="right"
      :fallbackPlacements="[
        'top-end',
        'top',
        'top-start',
        'bottom-end',
        'bottom',
        'bottom-start',
        'left'
      ]"
      :persistent="false"
      :enterable="false"
      :triggerKeys="[]"
      v-bind="textEllipsisTooltip === true ? {} : textEllipsisTooltip"
      :virtualTriggering="true"
      :virtualRef="tooltipVirtualRef"
      :content="tooltipContent"
      :visible="tooltipVisible"
    />
  </ElMenu>
</template>

<script setup>
  import { ref, shallowRef, computed, watch, onMounted } from 'vue';
  import { ElMenu, ElSubMenu, ElIcon } from 'element-plus';
  import { EllipsisOutlined } from '../icons/index';
  import { pick, contentIsEllipsis } from '../utils/common';
  import EleTooltip from '../ele-tooltip/index.vue';
  import MenuItems from './components/menu-items.vue';
  import { getMenuItems, getPopperClass, useMenuEllipsis } from './util';
  import { menusProps, menusEmits, menuPropKeys } from './props';

  defineOptions({ name: 'EleMenus' });

  const props = defineProps(menusProps);

  const emit = defineEmits(menusEmits);

  /** 是否是谷歌内核浏览器 */
  const isWebkit = navigator?.userAgent?.includes?.('WebKit');

  /** 菜单组件 */
  const menuRef = ref(null);

  /** 省略菜单子组件 */
  const ellipsisRef = ref(null);

  /** 菜单省略位置 */
  const sliceIndex = ref(-1);

  /** 菜单数据 */
  const menuItems = shallowRef([]);

  /** 省略的菜单数据 */
  const moreMenuItems = shallowRef([]);

  /** 文本溢出提示单例目标 */
  const tooltipVirtualRef = ref();

  /** 文本溢出提示文本 */
  const tooltipContent = ref('');

  /** 文本溢出提示是否显示 */
  const tooltipVisible = ref(false);

  /** 是否是水平菜单 */
  const isHorizontal = computed(() => props.mode === 'horizontal');

  /** 是否是紧凑菜单 */
  const isCompact = computed(() => props.mode === 'compact');

  /** 是否禁用折叠的提示组件 */
  const collapseTooltipDisabled = computed(() =>
    isCompact.value ? !props.collapse : props.tooltipDisabled
  );

  /** 菜单组件属性 */
  const menuProps = computed(() => pick(props, menuPropKeys));

  /** 展开指定的子菜单 */
  const open = (index) => {
    if (menuRef.value) {
      menuRef.value.open(index);
    }
  };

  /** 关闭指定的子菜单 */
  const close = (index) => {
    if (menuRef.value) {
      menuRef.value.open(index);
    }
  };

  /** 激活指定菜单 */
  const updateActiveIndex = (index) => {
    if (menuRef.value) {
      menuRef.value.updateActiveIndex(index);
    }
  };

  /** 关闭文本溢出提示 */
  const hideTooltip = () => {
    tooltipVisible.value = false;
  };

  /** 触发文本溢出提示 */
  const triggerTooltip = (e) => {
    if (props.textEllipsisTooltip) {
      const itemEl = e.currentTarget?.parentNode;
      if (itemEl) {
        const titleEl = itemEl.querySelector('.ele-menu-title');
        const text = titleEl?.innerText;
        if (text && contentIsEllipsis(titleEl, 'horizontal')) {
          tooltipVirtualRef.value = itemEl;
          tooltipContent.value = text;
          tooltipVisible.value = true;
          return;
        }
      }
    }
    hideTooltip();
  };

  /** 父菜单展开事件 */
  const handleOpen = (index, indexPath) => {
    emit('open', index, indexPath);
  };

  /** 父菜单关闭事件 */
  const handleClose = (index, indexPath) => {
    emit('close', index, indexPath);
  };

  /** 菜单项选中事件 */
  const handleSelect = (index, indexPath, item, routerResult) => {
    emit('select', index, indexPath, item, routerResult);
  };

  /** 子菜单项点击事件 */
  const handleItemClick = (item, e, type) => {
    emit('itemClick', item, e, type);
  };

  /** 子菜单项鼠标进入事件 */
  const handleItemMouseenter = (item, e) => {
    triggerTooltip(e);
    emit('itemMouseenter', item, e);
  };

  /** 子菜单项鼠标离开事件 */
  const handleItemMouseleave = (item, e) => {
    hideTooltip();
    emit('itemMouseleave', item, e);
  };

  /** 父级菜单项鼠标进入事件 */
  const handleParentMouseenter = (item, e) => {
    triggerTooltip(e);
    emit('parentMouseenter', item, e);
  };

  /** 父级菜单项鼠标离开事件 */
  const handleParentMouseleave = (item, e) => {
    hideTooltip();
    emit('parentMouseleave', item, e);
  };

  /** 滚动到选中的菜单节点 */
  const scrollToActive = () => {
    const menuEl = menuRef.value?.$el;
    if (menuEl) {
      const el =
        menuEl.querySelector('.el-menu-item.is-active') ||
        menuEl.querySelector('.el-sub-menu.is-active');
      if (el) {
        if (typeof el['scrollIntoViewIfNeeded'] === 'function') {
          el.scrollIntoViewIfNeeded(true);
        } else {
          el.scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
      }
    }
  };

  /** 监听菜单尺寸变化并溢出省略 */
  const { observe, unobserve, computedEllipsis } = useMenuEllipsis({
    getMenuEl: () => menuRef.value?.$el,
    getMoreEl: () => ellipsisRef.value?.$el,
    onEllipsis: (index) => {
      if (sliceIndex.value !== index) {
        sliceIndex.value = index;
      }
    }
  });

  onMounted(() => {
    if (props.ellipsis && isHorizontal.value && menuRef.value?.$el) {
      computedEllipsis();
      observe();
    }
  });

  watch(
    [
      () => props.ellipsis,
      isHorizontal,
      () => menuRef.value?.$el,
      () => props.items
    ],
    () => {
      if (props.ellipsis && isHorizontal.value && menuRef.value?.$el) {
        observe();
        return;
      }
      unobserve();
      sliceIndex.value = -1;
    }
  );

  /** 分割溢出的菜单 */
  watch(
    [() => props.items, sliceIndex, isHorizontal],
    () => {
      const { items, moreItems } = getMenuItems(
        props.items,
        sliceIndex.value,
        isHorizontal.value
      );
      menuItems.value = items;
      moreMenuItems.value = moreItems;
    },
    {
      immediate: true,
      deep: true
    }
  );

  defineExpose({
    menuRef,
    ellipsisRef,
    open,
    close,
    updateActiveIndex,
    scrollToActive
  });
</script>
