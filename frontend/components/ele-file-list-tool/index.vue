<!-- 文件列表表格操作按钮 -->
<template>
  <div class="ele-file-list-item-tool" @click.stop="handleClick">
    <slot></slot>
  </div>
</template>

<script setup>
  defineOptions({ name: 'EleFileListTool' });

  const emit = defineEmits({
    click: (_e) => true
  });

  const handleClick = (e) => {
    emit('click', e);
  };
</script>
