<!-- 标签页下拉按钮 -->
<template>
  <EleTabTool>
    <EleDropdown
      placement="bottom-end"
      popperClass="ele-tab-popup"
      :popperOptions="{
        modifiers: [{ name: 'offset', options: { offset: [12, 8] } }]
      }"
      :validateEvent="false"
      v-bind="dropdownProps || {}"
      ref="dropdownRef"
      :items="items"
      componentType="pro"
      @command="handleCommand"
    >
      <div :style="{ textAlign: 'center', outline: 'none' }">
        <slot name="icon">
          <ElIcon class="ele-tab-icon" :style="{ verticalAlign: '-3px' }">
            <ArrowDown />
          </ElIcon>
        </slot>
      </div>
    </EleDropdown>
  </EleTabTool>
</template>

<script setup>
  import { ref } from 'vue';
  import { ElIcon } from 'element-plus';
  import { ArrowDown } from '../../icons/index';
  import { useTimer } from '../../utils/hook';
  import EleDropdown from '../../ele-dropdown/index.vue';
  import EleTabTool from '../../ele-tab-tool/index.vue';

  defineOptions({ name: 'TabDropdown' });

  const props = defineProps({
    /** 右键菜单 */
    items: Array,
    /** 右键菜单属性 */
    dropdownProps: Object,
    /** 是否点击后自动关闭菜单 */
    autoClose: {
      type: Boolean,
      default: true
    }
  });

  const emit = defineEmits({
    menuClick: (_command) => true
  });

  const [startAutoCloseTimer] = useTimer(150);

  /** 下拉菜单组件 */
  const dropdownRef = ref(null);

  /** 下拉菜单点击事件 */
  const handleCommand = (command) => {
    emit('menuClick', command);
    if (props.autoClose) {
      startAutoCloseTimer(() => {
        if (dropdownRef.value) {
          dropdownRef.value.handleClose();
        }
      });
    }
  };

  defineExpose({
    dropdownRef
  });
</script>
