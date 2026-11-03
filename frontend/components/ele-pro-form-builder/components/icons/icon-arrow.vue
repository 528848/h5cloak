<template>
  <div
    :class="[
      {
        'ele-icon-border-color-text-light': color !== 'primary'
      },
      { 'ele-icon-border-color-primary': color === 'primary' }
    ]"
    :style="{
      flexShrink: 0,
      borderStyle: 'solid',
      borderWidth: size === 'sm' ? '3px' : '4px',
      ...{
        right: {
          borderTopColor: 'transparent',
          borderRightColor: 'transparent',
          borderBottomColor: 'transparent'
        },
        left: {
          borderTopColor: 'transparent',
          borderLeftColor: 'transparent',
          borderBottomColor: 'transparent'
        },
        down: {
          borderLeftColor: 'transparent',
          borderRightColor: 'transparent',
          borderBottomColor: 'transparent'
        },
        up: {
          borderTopColor: 'transparent',
          borderLeftColor: 'transparent',
          borderRightColor: 'transparent'
        }
      }[direction || 'right']
    }"
  ></div>
</template>

<script setup>
  defineProps({
    size: String,
    direction: String,
    color: String
  });
</script>
