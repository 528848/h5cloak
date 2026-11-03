<!-- 黑白名单搜索 -->
<template>
  <ele-card search-form :body-style="{ paddingBottom: '2px' }">
    <el-form
      label-width="72px"
      class="access-list-search-form"
      @keyup.enter="handleSearch"
      @submit.prevent=""
    >
      <el-row :gutter="12" align="middle">
        <el-col :xl="6" :lg="6" :md="12" :sm="12" :xs="24">
          <el-form-item label="名单类型">
            <el-select
              v-model="form.listType"
              clearable
              placeholder="全部"
              class="ele-fluid"
            >
              <el-option label="黑名单" value="black" />
              <el-option label="白名单" value="white" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :xl="6" :lg="6" :md="12" :sm="12" :xs="24">
          <el-form-item label="IP 地址">
            <el-input
              v-model.trim="form.targetValue"
              clearable
              placeholder="请输入"
            />
          </el-form-item>
        </el-col>
        <el-col
          :xl="12"
          :lg="12"
          :md="24"
          :sm="24"
          :xs="24"
          class="search-btn-col"
        >
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

  const [form, resetFields] = useFormData({
    listType: void 0,
    targetValue: ''
  });

  const handleSearch = () => {
    emit('search', { ...form });
  };

  const handleReset = () => {
    resetFields();
    handleSearch();
  };
</script>

<style scoped>
  .access-list-search-form :deep(.el-form-item) {
    margin-bottom: 12px;
  }

  .search-btn-col {
    display: flex;
    justify-content: flex-end;
  }

  .search-btn-col :deep(.el-form-item) {
    margin-right: 0;
  }

  @media screen and (max-width: 991.99px) {
    .search-btn-col {
      justify-content: flex-start;
    }
  }
</style>
