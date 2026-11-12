<!-- 决策日志搜索 -->
<template>
  <ele-card search-form class="page-section">
    <el-form label-width="76px" @keyup.enter="handleSearch" @submit.prevent="">
      <el-row :gutter="12">
        <el-col v-if="showSite" :lg="6" :md="8" :sm="12" :xs="24">
          <el-form-item label="站点">
            <el-select
              v-model="siteModel"
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
        <el-col :lg="8" :md="12" :sm="24" :xs="24">
          <el-form-item label="时间">
            <el-date-picker
              v-model="range"
              type="datetimerange"
              value-format="YYYY-MM-DD HH:mm:ss"
              range-separator="至"
              start-placeholder="开始"
              end-placeholder="结束"
              class="ele-fluid"
              :shortcuts="shortcuts"
            />
          </el-form-item>
        </el-col>
        <el-col :lg="5" :md="8" :sm="12" :xs="24">
          <el-form-item label="返回内容">
            <el-select
              v-model="form.content"
              clearable
              placeholder="全部"
              class="ele-fluid"
            >
              <el-option
                v-for="o in contentOptions"
                :key="o.value"
                :label="o.label"
                :value="o.value"
              />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :lg="5" :md="8" :sm="12" :xs="24">
          <el-form-item label="触点">
            <el-select
              v-model="form.touchType"
              clearable
              placeholder="全部"
              class="ele-fluid"
            >
              <el-option
                v-for="o in touchOptions"
                :key="o.value"
                :label="o.label"
                :value="o.value"
              />
            </el-select>
          </el-form-item>
        </el-col>
        <template v-if="expanded">
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="路径">
              <el-input
                v-model.trim="form.path"
                clearable
                placeholder="前缀匹配，如 /lp"
                class="mono"
              />
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="归因码">
              <el-input
                v-model.trim="form.code"
                clearable
                placeholder="精确码值"
                class="mono"
              />
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="失败原因">
              <el-select
                v-model="form.reason"
                clearable
                placeholder="全部"
                class="ele-fluid"
              >
                <el-option
                  v-for="o in reasonOptions"
                  :key="o.value"
                  :label="o.label"
                  :value="o.value"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="内置浏览器">
              <el-select
                v-model="form.inApp"
                clearable
                placeholder="全部"
                class="ele-fluid"
              >
                <el-option
                  v-for="o in inAppOptions"
                  :key="o.value"
                  :label="o.label"
                  :value="o.value"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="渠道">
              <el-input
                v-model.trim="form.channel"
                clearable
                placeholder="如 distributor:ALICE"
                class="mono"
              />
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="国家">
              <el-input
                v-model.trim="form.country"
                clearable
                placeholder="两位代码，如 US"
                maxlength="2"
                class="mono"
              />
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="访客 ID">
              <el-input v-model.trim="form.visitorId" clearable class="mono" />
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="请求 ID">
              <el-input v-model.trim="form.requestId" clearable class="mono" />
            </el-form-item>
          </el-col>
        </template>
        <el-col :lg="6" :md="8" :sm="12" :xs="24">
          <el-form-item label-width="0">
            <el-space :size="8">
              <btn-items
                :wrap="false"
                :items="[
                  { preset: 'search', onClick: () => handleSearch() },
                  { preset: 'reset', onClick: () => handleReset() }
                ]"
              />
              <el-button link type="primary" @click="expanded = !expanded">
                {{ expanded ? '收起' : '更多条件' }}
              </el-button>
            </el-space>
          </el-form-item>
        </el-col>
      </el-row>
    </el-form>
  </ele-card>
</template>

<script setup>
  import { ref, computed } from 'vue';
  import { useFormData } from '@/utils/use-form-data';
  import {
    contentOptions,
    touchOptions,
    reasonOptions,
    inAppOptions
  } from '../constants';

  const props = defineProps({
    sites: { type: Array, default: () => [] },
    siteId: { type: Number, default: null },
    showSite: { type: Boolean, default: true }
  });
  const emit = defineEmits(['search', 'update:siteId']);

  const expanded = ref(false);
  const range = ref([]);
  const siteModel = computed({
    get: () => props.siteId,
    set: (v) => emit('update:siteId', v || null)
  });

  const [form, resetFields] = useFormData({
    content: '',
    touchType: '',
    path: '',
    code: '',
    reason: '',
    inApp: '',
    channel: '',
    country: '',
    visitorId: '',
    requestId: ''
  });

  const fmt = (d) => {
    const p = (n) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
  };
  const shortcuts = [
    {
      text: '最近 1 小时',
      value: () => [new Date(Date.now() - 3600e3), new Date()]
    },
    {
      text: '今天',
      value: () => [new Date(new Date().setHours(0, 0, 0, 0)), new Date()]
    },
    {
      text: '最近 7 天',
      value: () => [new Date(Date.now() - 7 * 86400e3), new Date()]
    }
  ];

  const where = () => ({
    ...form,
    siteId: props.siteId || undefined,
    from: range.value?.[0] || undefined,
    to: range.value?.[1] || undefined
  });

  const handleSearch = () => emit('search', where());
  const handleReset = () => {
    resetFields();
    range.value = [];
    handleSearch();
  };

  const setContent = (content) => {
    form.content = content;
    handleSearch();
  };

  defineExpose({ where, setContent, fmt });
</script>
