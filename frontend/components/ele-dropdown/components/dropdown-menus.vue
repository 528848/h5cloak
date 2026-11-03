<template>
  <div
    class="ele-dropdown-menu"
    :class="[
      { 'ele-dropdown-icon-small': iconSize === 'small' },
      { 'is-small': size === 'small' },
      { 'is-large': size === 'large' }
    ]"
    :style="menuStyle"
  >
    <template
      v-for="item in items"
      :key="JSON.stringify(item.key ?? item.command ?? item.title)"
    >
      <DropdownMenuItem
        :item="item"
        :selected="selected"
        :iconProps="iconProps"
        @itemClick="handleItemClick"
        @wrapperContext="handleWrapperContext"
      >
        <template
          v-for="name in Object.keys($slots).filter(
            (k) => 'default' !== k && 'subMenus' !== k
          )"
          #[name]="slotProps"
        >
          <slot :name="name" v-bind="slotProps || {}"></slot>
        </template>
        <template v-if="item.children && item.children.length" #subMenus>
          <DropdownMenus
            :items="item.children"
            :selected="selected"
            :menuStyle="menuStyle"
            :iconProps="iconProps"
            :size="size"
            @itemClick="handleItemClick"
            @wrapperContext="handleWrapperContext"
          >
            <template
              v-for="name in Object.keys($slots).filter(
                (k) => 'default' !== k && 'subMenus' !== k
              )"
              #[name]="slotProps"
            >
              <slot :name="name" v-bind="slotProps || {}"></slot>
            </template>
          </DropdownMenus>
        </template>
      </DropdownMenuItem>
    </template>
  </div>
</template>

<script setup>
  import DropdownMenuItem from './dropdown-menu-item.vue';

  defineOptions({ name: 'DropdownMenus' });

  defineProps({
    /** 下拉菜单数据 */
    items: {
      type: Array,
      required: true
    },
    /** 选中的菜单 */
    selected: [String, Number, Object],
    /** 自定义下拉菜单样式 */
    menuStyle: Object,
    /** 自定义图标属性 */
    iconProps: Object,
    /** 图标尺寸 */
    iconSize: String,
    /** 尺寸 */
    size: String
  });

  const emit = defineEmits({
    itemClick: (_item, _e) => true,
    wrapperContext: (_e) => true
  });

  /** 菜单项点击事件 */
  const handleItemClick = (item, e) => {
    if (item.disabled) {
      return;
    }
    emit('itemClick', item, e);
  };

  /** 菜单容器右键事件 */
  const handleWrapperContext = (e) => {
    emit('wrapperContext', e);
  };
</script>
