<!-- 海外投放地区选择，按大洲分组，数据与 V1 相同。 -->
<template>
  <el-cascader
    v-model="model"
    :options="countryRegions"
    :props="cascaderProps"
    placeholder="请选择投放地区"
    clearable
    filterable
    collapse-tags
    :max-collapse-tags="5"
    :show-all-levels="false"
    class="ele-fluid"
    popper-class="ele-popper-higher"
  />
</template>

<script setup>
  import { countryRegions } from '@/utils/country-regions';

  defineOptions({ name: 'CountryRegionsSelect' });

  const model = defineModel({ type: Array, default: () => [] });

  const cascaderProps = {
    multiple: true,
    emitPath: true,
    value: 'value',
    label: 'label',
    children: 'children'
  };
</script>
