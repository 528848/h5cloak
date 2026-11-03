<!-- 域名搜索 -->
<template>
  <ele-card search-form class="page-section">
    <el-form label-width="72px" @keyup.enter="handleSearch" @submit.prevent="">
      <el-row :gutter="12">
        <el-col :lg="6" :md="8" :sm="12" :xs="24">
          <el-form-item label="域名">
            <el-input
              v-model.trim="form.keyword"
              clearable
              placeholder="域名关键字"
            />
          </el-form-item>
        </el-col>
        <el-col :lg="6" :md="8" :sm="12" :xs="24">
          <el-form-item label="所属站点">
            <el-select
              v-model="form.siteId"
              clearable
              placeholder="全部"
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
        <el-col :lg="6" :md="8" :sm="12" :xs="24">
          <el-form-item label="状态">
            <el-select
              v-model="form.status"
              clearable
              placeholder="全部"
              class="ele-fluid"
            >
              <el-option
                v-for="o in domainStatusOptions"
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
  import { ref, onMounted } from 'vue';
  import { useFormData } from '@/utils/use-form-data';
  import { listSiteOptions } from '@/api/merchant/site';
  import { domainStatusOptions } from '../../site/constants';

  const emit = defineEmits(['search']);
  const sites = ref([]);
  const [form, resetFields] = useFormData({
    keyword: '',
    siteId: '',
    status: ''
  });

  const payload = () => ({
    keyword: form.keyword,
    status: form.status,
    siteId: form.siteId || undefined
  });

  const handleSearch = () => emit('search', payload());
  const handleReset = () => {
    resetFields();
    handleSearch();
  };

  const setStatus = (status) => {
    form.status = status;
    handleSearch();
  };

  defineExpose({ setStatus });

  onMounted(() => {
    listSiteOptions()
      .then((rows) => (sites.value = rows || []))
      .catch(() => {});
  });
</script>
