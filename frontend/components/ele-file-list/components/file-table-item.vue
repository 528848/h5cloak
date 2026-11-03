<!-- 表格布局条目 -->
<template>
  <div
    class="ele-file-list-item"
    :class="[
      { 'is-checked': selected },
      { 'is-active': ctxMenuDropdownVisible }
    ]"
    @click.stop="handleClick"
  >
    <div class="ele-file-list-item-body" @contextmenu="handleContextmenu">
      <!-- 复选框列 -->
      <div v-if="selectable" class="ele-file-list-item-checkbox">
        <i
          class="ele-file-list-checkbox"
          :class="[
            { 'is-radio': selectionType === 'radio' },
            { 'is-checked': selected }
          ]"
          @click.stop="handleCheckChange"
        ></i>
      </div>
      <!-- 标题列 -->
      <div class="ele-file-list-item-name">
        <div class="ele-file-list-item-icon">
          <slot name="icon" :icon="icon" :item="item">
            <img
              :src="icon"
              :class="{ 'ele-file-list-item-image': !!item.thumbnail }"
            />
          </slot>
          <slot name="title" :item="item">
            <div :title="item.name" class="ele-file-list-item-title">
              {{ item.name }}
            </div>
          </slot>
        </div>
        <div v-if="$slots.tool" class="ele-file-list-item-tools">
          <slot name="tool" :item="item"></slot>
        </div>
      </div>
      <!-- 自定义列 -->
      <div
        v-for="col in columns"
        :key="col.prop"
        :style="col.style"
        class="ele-file-list-item-cell"
      >
        <template
          v-if="
            col.slot &&
            !['icon', 'title', 'tool', 'contextMenu'].includes(col.slot) &&
            $slots[col.slot]
          "
        >
          <slot :name="col.slot" :item="item" :col="col"></slot>
        </template>
        <template v-else-if="item && col && col.prop">
          {{ item[col.prop] }}
        </template>
      </div>
    </div>
    <!-- 右键菜单 -->
    <div
      ref="dropdownTriggerRef"
      class="ele-file-list-item-dropdown"
      @click.stop=""
    ></div>
  </div>
</template>

<script setup>
  import { ref, computed } from 'vue';

  defineOptions({ name: 'FileTableItem' });

  const props = defineProps({
    /** 数据 */
    item: {
      type: Object,
      required: true
    },
    /** 选择框类型 */
    selectionType: String,
    /** 多选选中数据 */
    selections: Array,
    /** 单选选中数据 */
    current: Object,
    /** 后缀对应的图标 */
    icons: Array,
    /** 自定义列配置 */
    columns: Array,
    /** 右键下拉菜单是否显示 */
    ctxMenuDropdownVisible: Boolean
  });

  const emit = defineEmits({
    click: (_item) => true,
    checkChange: (_item) => true,
    contextOpen: (_option) => true
  });

  /** 右键菜单触发节点 */
  const dropdownTriggerRef = ref(null);

  /** 是否可以选中 */
  const selectable = computed(() => {
    return (
      props.selectionType === 'checkbox' || props.selectionType === 'radio'
    );
  });

  /** 是否选中 */
  const selected = computed(() => {
    if (!selectable.value || !props.item) {
      return false;
    }
    if (props.selectionType === 'checkbox') {
      return !!(
        props.selections &&
        props.selections.some((t) => t.key === props.item.key)
      );
    }
    return !!(props.current && props.current.key === props.item.key);
  });

  /** 对应图标 */
  const icon = computed(() => {
    // 有缩略图返回文件缩略图
    if (props.item.thumbnail) {
      return props.item.thumbnail;
    }
    if (!props.icons) {
      return;
    }
    // 文件夹
    if (props.item.isDirectory) {
      return props.icons.find((d) => d.type === 'dir')?.icon;
    }
    // 匹配后缀对应的图标
    if (props.item.name) {
      const icon = props.icons.find((d) => {
        if (!d.suffixes) {
          return false;
        }
        return !!d.suffixes.find((t) => !!props.item.name.endsWith(t));
      })?.icon;
      if (icon) {
        return icon;
      }
    }
    return props.icons.find((d) => d.type === 'file')?.icon;
  });

  /** 点击事件 */
  const handleClick = () => {
    emit('click', props.item);
  };

  /** 复选框点击事件 */
  const handleCheckChange = () => {
    emit('checkChange', props.item);
  };

  /** 触发右键菜单 */
  const handleContextmenu = (e) => {
    if (dropdownTriggerRef.value != null) {
      emit('contextOpen', {
        item: props.item,
        triggerEl: dropdownTriggerRef.value,
        e
      });
    }
  };
</script>
