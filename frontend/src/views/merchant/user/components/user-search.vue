<template>
  <ele-card search-form class="page-section">
    <el-form label-width="72px" @keyup.enter="handleSearch" @submit.prevent="">
      <el-row :gutter="12">
        <el-col :lg="6" :md="8" :sm="12" :xs="24">
          <el-form-item label="关键字">
            <el-input
              v-model.trim="form.keyword"
              clearable
              placeholder="账号或昵称"
            />
          </el-form-item>
        </el-col>
        <el-col :lg="6" :md="8" :sm="12" :xs="24">
          <el-form-item label="角色">
            <el-select
              v-model="form.roleKey"
              clearable
              placeholder="全部"
              class="ele-fluid"
            >
              <el-option
                v-for="o in roles"
                :key="o.roleKey"
                :label="o.name"
                :value="o.roleKey"
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
                v-for="o in userStatusOptions"
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
  import { ref } from 'vue';
  import { EleMessage } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { listMerchantRoles } from '@/api/merchant/user';
  import { roleOptions, userStatusOptions } from '../constants';

  const roles = ref([]);
  listMerchantRoles()
    .then((rows) => {
      roles.value = rows || [];
    })
    .catch((e) => {
      roles.value = roleOptions.map((o) => ({ roleKey: o.value, name: o.label }));
      EleMessage.error({ message: e.message, plain: true });
    });

  const emit = defineEmits(['search']);
  const [form, resetFields] = useFormData({
    keyword: '',
    roleKey: '',
    status: ''
  });
  const handleSearch = () => emit('search', { ...form });
  const handleReset = () => {
    resetFields();
    handleSearch();
  };
</script>
