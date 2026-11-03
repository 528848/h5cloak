<template>
  <div
    :class="[
      { 'ele-icon-bg-fill-light': color !== 'primary' },
      { 'ele-icon-bg-primary': color === 'primary' }
    ]"
    :style="{
      flexShrink: 0,
      height: { xl: '18px', lg: '12px', md: '10px', sm: '6px', xs: '4px' }[
        size || 'md'
      ],
      borderRadius: size === 'sm' ? '3px' : size === 'xs' ? '2px' : '4px'
    }"
  >
    <slot></slot>
  </div>
</template>

<script setup>
  defineProps({
    size: String,
    color: String
  });
</script>
