<!-- 搜索表单 -->
<template>
  <el-form label-width="72px" @keyup.enter="handleSearch" @submit.prevent="">
    <el-row :gutter="16">
      <el-col :lg="6" :md="12" :sm="12" :xs="24">
        <el-form-item label="任务名称">
          <el-input
            clearable
            v-model.trim="form.jobName"
            placeholder="请输入"
            :disabled="!!data"
          />
        </el-form-item>
      </el-col>
      <el-col :lg="6" :md="12" :sm="12" :xs="24">
        <el-form-item label="任务组名">
          <dict-data
            code="sys_job_group"
            v-model="form.jobGroup"
            placeholder="请选择"
            :disabled="!!data"
          />
        </el-form-item>
      </el-col>
      <el-col :lg="8" :md="18" :sm="24" :xs="24">
        <el-form-item label="执行时间">
          <el-date-picker
            unlink-panels
            type="daterange"
            v-model="dateRange"
            range-separator="-"
            value-format="YYYY-MM-DD"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            class="ele-fluid"
          />
        </el-form-item>
      </el-col>
      <el-col :lg="4" :md="6" :sm="24" :xs="24">
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
</template>

<script setup>
  import { ref } from 'vue';
  import { useFormData } from '@/utils/use-form-data';

  const emit = defineEmits(['search']);

  const props = defineProps({
    /** 定时任务 */
    data: Object
  });

  /** 表单数据 */
  const [form, resetFields] = useFormData({
    jobName: props.data?.jobName || '',
    jobGroup: props.data?.jobGroup
  });

  /** 日期范围 */
  const dateRange = ref(['', '']);

  /** 搜索 */
  const handleSearch = () => {
    const [d1, d2] = dateRange.value ?? [];
    emit('search', {
      ...form,
      params: {
        beginTime: d1 ? `${d1} 00:00:00` : '',
        endTime: d2 ? `${d2} 23:59:59` : ''
      }
    });
  };

  /** 重置 */
  const handleReset = () => {
    resetFields();
    dateRange.value = ['', ''];
    handleSearch();
  };
</script>
