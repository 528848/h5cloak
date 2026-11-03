<!-- 商户搜索 -->
<template>
  <ele-card search-form class="page-section">
    <el-form label-width="72px" @keyup.enter="handleSearch" @submit.prevent="">
      <el-row :gutter="12">
        <el-col :lg="6" :md="8" :sm="12" :xs="24">
          <el-form-item label="关键字">
            <el-input
              v-model.trim="form.keyword"
              clearable
              placeholder="商户名称、联系人、邮箱"
            />
          </el-form-item>
        </el-col>
        <el-col :lg="6" :md="8" :sm="12" :xs="24">
          <el-form-item label="状态">
            <el-select
              v-model="form.status"
              clearable
              placeholder="全部"
              class="ele-fluid"
            >
              <el-option
                v-for="item in merchantStatusOptions.filter(
                  (o) => o.value !== 'expired'
                )"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :lg="4" :md="8" :sm="12" :xs="24">
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
  import { useFormData } from '@/utils/use-form-data';
  import { merchantStatusOptions } from '../constants';

  const emit = defineEmits(['search']);

  const [form, resetFields] = useFormData({ keyword: '', status: '' });

  const handleSearch = () => emit('search', { ...form });

  const handleReset = () => {
    resetFields();
    handleSearch();
  };

  /** 供外部（概览卡片）写回筛选值 */
  const setStatus = (status) => {
    form.status = status;
    handleSearch();
  };

  defineExpose({ setStatus });
</script>
