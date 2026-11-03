<!-- 分销码搜索 -->
<template>
  <ele-card search-form class="page-section">
    <el-form label-width="72px" @keyup.enter="handleSearch" @submit.prevent="">
      <el-row :gutter="12">
        <el-col :lg="5" :md="8" :sm="12" :xs="24">
          <el-form-item label="站点">
            <el-select
              v-model="siteModel"
              placeholder="全部"
              clearable
              class="ele-fluid"
            >
              <el-option
                v-for="s in sites"
                :key="s.siteId"
                :label="s.name"
                :value="s.siteId"
              />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :lg="5" :md="8" :sm="12" :xs="24">
          <el-form-item label="关键字">
            <el-input
              v-model.trim="form.keyword"
              clearable
              placeholder="码值、名称、渠道"
            />
          </el-form-item>
        </el-col>
        <el-col :lg="4" :md="8" :sm="12" :xs="24">
          <el-form-item label="类型">
            <el-select
              v-model="form.codeType"
              clearable
              placeholder="全部"
              class="ele-fluid"
            >
              <el-option
                v-for="o in codeTypeOptions"
                :key="o.value"
                :label="o.label"
                :value="o.value"
              />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :lg="4" :md="8" :sm="12" :xs="24">
          <el-form-item label="状态">
            <el-select
              v-model="form.status"
              clearable
              placeholder="全部"
              class="ele-fluid"
            >
              <el-option
                v-for="o in codeStatusOptions.filter(
                  (o) => o.value !== 'pending'
                )"
                :key="o.value"
                :label="o.label"
                :value="o.value"
              />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :lg="6" :md="8" :sm="12" :xs="24">
          <el-form-item label-width="0">
            <btn-items
              :wrap="false"
              :items="[
                { preset: 'search', onClick: () => handleSearch() },
                { preset: 'reset', onClick: () => handleReset() }
              ]"
            />
          </el-form-item>
        </el-col>
      </el-row>
    </el-form>
  </ele-card>
</template>

<script setup>
  import { computed } from 'vue';
  import { useFormData } from '@/utils/use-form-data';
  import { codeTypeOptions, codeStatusOptions } from '../constants';

  const props = defineProps({
    sites: { type: Array, default: () => [] },
    siteId: { type: Number, default: null }
  });
  const emit = defineEmits(['search', 'update:siteId']);

  const siteModel = computed({
    get: () => props.siteId,
    set: (v) => emit('update:siteId', v || null)
  });

  const [form, resetFields] = useFormData({
    keyword: '',
    codeType: '',
    status: ''
  });

  const where = () => ({
    keyword: form.keyword,
    codeType: form.codeType,
    status: form.status,
    siteId: props.siteId || undefined
  });

  const handleSearch = () => emit('search', where());
  const handleReset = () => {
    resetFields();
    handleSearch();
  };

  const setStatus = (status) => {
    form.status = status;
    handleSearch();
  };

  defineExpose({ setStatus, where });
</script>
