<!-- 模板库列表 -->
<template>
  <div class="ele-pro-form-builder-template-wrapper">
    <ElEmpty
      v-if="!templateData || !templateData.length"
      :imageSize="58"
      class="ele-pro-form-builder-form-empty"
    />
    <template v-else>
      <div
        v-for="item in templateData"
        :key="item.name"
        class="ele-pro-form-builder-template-item"
        @click="handleImportTemplate(item)"
      >
        <div class="ele-pro-form-builder-template-item-label">
          {{ item.name }}
        </div>
        <div class="ele-pro-form-builder-template-item-body">
          <div class="ele-pro-form-builder-template-item-cover">
            <component v-if="item.cover" :is="item.cover" />
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
  import { ElEmpty } from 'element-plus';
  import { deepCloneObject } from './build-core';
  import { itemsGenerateNewKey } from './build-util';

  defineOptions({ name: 'TemplateList' });

  defineProps({
    /** 模板数据 */
    templateData: Array
  });

  const emit = defineEmits({
    importData: (_data) => true
  });

  /** 导入模板 */
  const handleImportTemplate = (item) => {
    const result = deepCloneObject(item.config);
    itemsGenerateNewKey(result.items, [], false);
    emit('importData', result);
  };
</script>
