<!-- 斗篷搜索 -->
<template>
  <ele-card search-form :body-style="{ paddingBottom: '2px' }">
    <el-form
      label-width="72px"
      class="cloak-search-form"
      @keyup.enter="handleSearch"
      @submit.prevent=""
    >
      <el-row :gutter="12" align="middle">
        <el-col :xl="5" :lg="5" :md="12" :sm="12" :xs="24">
          <el-form-item label="名称">
            <el-input v-model.trim="form.name" clearable placeholder="请输入" />
          </el-form-item>
        </el-col>
        <el-col :xl="5" :lg="5" :md="12" :sm="12" :xs="24">
          <el-form-item label="短链">
            <el-input
              v-model.trim="form.shortLinkUrl"
              clearable
              placeholder="请输入"
            />
          </el-form-item>
        </el-col>
        <el-col :xl="5" :lg="5" :md="12" :sm="12" :xs="24">
          <el-form-item label="真实链接">
            <el-input
              v-model.trim="form.blackUrl"
              clearable
              placeholder="请输入"
            />
          </el-form-item>
        </el-col>
        <el-col :xl="5" :lg="5" :md="12" :sm="12" :xs="24">
          <el-form-item label="模式">
            <el-select
              v-model="form.mode"
              clearable
              placeholder="全部"
              class="ele-fluid"
            >
              <el-option label="斗篷" value="cloak" />
              <el-option label="绿色" value="green" />
              <el-option label="全开" value="full_open" />
              <el-option label="审核" value="review" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col
          :xl="4"
          :lg="4"
          :md="12"
          :sm="12"
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
    name: '',
    shortLinkUrl: '',
    blackUrl: '',
    mode: void 0
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
  .cloak-search-form :deep(.el-form-item) {
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
