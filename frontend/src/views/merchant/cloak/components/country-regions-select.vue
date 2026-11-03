<!-- 海外投放地区选择 -->
<template>
  <el-cascader
    v-model="model"
    :options="options"
    :props="cascaderProps"
    :placeholder="placeholder"
    :clearable="clearable"
    :filterable="filterable"
    :collapse-tags="true"
    :max-collapse-tags="5"
    :show-all-levels="false"
    class="ele-fluid"
    popper-class="ele-popper-higher"
  />
</template>

<script setup>
  import { onMounted, ref } from 'vue';
  import { EleMessage } from 'ele-admin-plus';
  import request from '@/utils/request';

  defineOptions({ name: 'CountryRegionsSelect' });

  defineProps({
    placeholder: {
      type: String,
      default: '请选择投放地区'
    },
    clearable: {
      type: Boolean,
      default: true
    },
    filterable: {
      type: Boolean,
      default: true
    }
  });

  const model = defineModel({ type: Array, default: () => [] });

  const options = ref([]);

  const cascaderProps = {
    multiple: true,
    emitPath: true,
    value: 'value',
    label: 'label',
    children: 'children'
  };

  onMounted(() => {
    const baseUrl = import.meta.env.BASE_URL || '/';
    request
      .get(`${baseUrl}json/country-regions-data.json`, { baseURL: '' })
      .then((res) => {
        options.value = res.data ?? [];
      })
      .catch((e) => {
        EleMessage.error({ message: e.message, plain: true });
      });
  });
</script>
