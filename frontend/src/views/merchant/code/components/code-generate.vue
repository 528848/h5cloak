<!-- 批量生成分销码 -->
<template>
  <ele-modal
    :form="true"
    :width="560"
    title="批量生成"
    :loading="loading"
    v-bind="modalProps"
  >
    <template v-if="!result">
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="96px"
        @submit.prevent=""
      >
        <el-form-item label="站点" prop="siteId">
          <el-select
            v-model="form.siteId"
            placeholder="请选择站点"
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
        <el-form-item label="类型">
          <el-radio-group v-model="form.codeType">
            <el-radio
              v-for="o in codeTypeOptions"
              :key="o.value"
              :value="o.value"
            >
              {{ o.label }}
            </el-radio>
          </el-radio-group>
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="前缀">
              <el-input
                v-model.trim="form.prefix"
                maxlength="16"
                placeholder="可选，如 TK"
                class="mono"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="随机位数" prop="length">
              <el-input-number
                v-model="form.length"
                :min="4"
                :max="32"
                controls-position="right"
                class="ele-fluid"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="数量" prop="count">
              <el-input-number
                v-model="form.count"
                :min="1"
                :max="10000"
                controls-position="right"
                class="ele-fluid"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="渠道标签">
              <el-select
                v-model="form.channel"
                filterable
                allow-create
                default-first-option
                clearable
                placeholder="可选"
                class="ele-fluid"
              >
                <el-option
                  v-for="c in channels"
                  :key="c"
                  :label="c"
                  :value="c"
                />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="失效时间">
          <el-date-picker
            v-model="form.expiresAt"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm:ss"
            placeholder="留空长期有效"
            class="ele-fluid"
          />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" maxlength="200" placeholder="可选" />
        </el-form-item>
      </el-form>
      <el-alert type="info" show-icon :closable="false">
        示例：{{ preview }}。生成的码使用去掉易混淆字符的大写字母和数字。
      </el-alert>
    </template>
    <el-result
      v-else
      icon="success"
      :title="`已生成 ${result.created} 个码`"
      :sub-title="`耗时 ${result.elapsedMs} ms`"
    >
      <template #extra>
        <div class="sample mono">{{ result.sample.join('  ') }}</div>
      </template>
    </el-result>
    <template #footer>
      <btn-items
        v-if="!result"
        :items="[
          { preset: 'cancel', onClick: () => closeModal() },
          { preset: 'save', title: '生成', onClick: () => handleSave() }
        ]"
      />
      <el-button v-else type="primary" @click="closeModal()">完成</el-button>
    </template>
  </ele-modal>
</template>

<script setup>
  import { ref, reactive, computed } from 'vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { generateCodes } from '@/api/merchant/code';
  import { codeTypeOptions } from '../constants';

  const props = defineProps({
    siteId: Number,
    sites: { type: Array, default: () => [] },
    channels: { type: Array, default: () => [] }
  });
  const emit = defineEmits(['done']);
  const { modalProps, closeModal } = useModal();
  const loading = ref(false);
  const formRef = ref(null);
  const result = ref(null);

  const [form] = useFormData({
    siteId: props.siteId ?? undefined,
    codeType: 'distributor',
    prefix: '',
    length: 8,
    count: 100,
    channel: '',
    expiresAt: '',
    remark: ''
  });

  const preview = computed(
    () => `${form.prefix || ''}${'X'.repeat(Math.min(form.length || 8, 32))}`
  );

  const rules = reactive({
    siteId: [{ required: true, message: '请选择站点', trigger: 'change' }],
    count: [{ required: true, message: '请输入数量', trigger: 'change' }],
    length: [{ required: true, message: '请输入随机位数', trigger: 'change' }]
  });

  const handleSave = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) return;
      loading.value = true;
      generateCodes({ ...form, expiresAt: form.expiresAt || '' })
        .then((res) => {
          loading.value = false;
          result.value = res.data;
          emit('done');
        })
        .catch((e) => {
          loading.value = false;
          EleMessage.error({ message: e.message, plain: true });
        });
    });
  };
</script>

<style scoped>
  .sample {
    word-break: break-all;
    color: var(--el-text-color-secondary);
  }
</style>
