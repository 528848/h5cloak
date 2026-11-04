<template>
  <ele-card search-form>
    <el-form label-width="72px" @keyup.enter="handleSearch" @submit.prevent="">
      <el-row :gutter="16">
        <el-col :lg="6" :md="8" :sm="12" :xs="24">
          <el-form-item label="菜单名称">
            <el-input clearable v-model.trim="form.name" placeholder="请输入" />
          </el-form-item>
        </el-col>
        <el-col :lg="6" :md="8" :sm="12" :xs="24">
          <el-form-item label="显示状态">
            <el-select
              v-model="form.visible"
              clearable
              placeholder="请选择"
              class="ele-fluid"
            >
              <el-option label="显示" value="1" />
              <el-option label="隐藏" value="0" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :lg="6" :md="8" :sm="12" :xs="24">
          <el-form-item label-width="0px">
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

  const emit = defineEmits(['search']);
  const [form, resetFields] = useFormData({ name: '', visible: '' });
  const handleSearch = () => emit('search', { ...form });
  const handleReset = () => {
    resetFields();
    handleSearch();
  };
</script>
