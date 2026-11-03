<!-- 全局页脚 -->
<template>
  <ele-text type="placeholder" class="ele-footer">
    <div>Copyright © {{ year }} {{ appName }} All Rights Reserved.</div>
  </ele-text>
</template>

<script setup>
  import { APP_NAME } from '@portal';

  const year = new Date().getFullYear();
  const appName = APP_NAME;
</script>
