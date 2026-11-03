<!-- 参数编辑：商户端（站点参数 / 覆盖内置）与总后台（内置字典）共用 -->
<template>
  <ele-modal
    :form="true"
    :width="620"
    :title="title"
    :loading="loading"
    v-bind="modalProps"
  >
    <el-alert
      v-if="mode === 'merchant' && data?.source === 'builtin'"
      type="info"
      show-icon
      :closable="false"
      style="margin-bottom: 16px"
      title="这是平台内置参数。保存后会生成本站点的覆盖配置，删除覆盖可恢复内置设置。"
    />
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="110px"
      @submit.prevent=""
    >
      <el-row :gutter="12">
        <el-col :span="12">
          <el-form-item label="参数名" prop="name">
            <el-input
              v-model.trim="form.name"
              :disabled="nameLocked"
              placeholder="如 fbclid、ref"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="类型" prop="paramType">
            <el-select v-model="form.paramType" class="ele-fluid">
              <el-option
                v-for="o in paramTypeOptions"
                :key="o.value"
                :label="o.label"
                :value="o.value"
              />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="所属平台">
            <el-input
              v-model.trim="form.platform"
              placeholder="facebook、google、tiktok"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="排序">
            <el-input-number
              v-model="form.sort"
              :min="0"
              :max="9999"
              controls-position="right"
              class="ele-fluid"
            />
          </el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="别名">
            <el-input
              v-model="form.aliasesText"
              placeholder="多个用逗号分隔，如 r, aff, affiliate"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="最短长度" prop="minLen">
            <el-input-number
              v-model="form.minLen"
              :min="1"
              :max="2048"
              controls-position="right"
              class="ele-fluid"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="最长长度" prop="maxLen">
            <el-input-number
              v-model="form.maxLen"
              :min="1"
              :max="2048"
              controls-position="right"
              class="ele-fluid"
            />
          </el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="格式正则">
            <el-input
              v-model.trim="form.pattern"
              placeholder="可选，如 ^[A-Za-z0-9_-]+$"
              class="mono"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="单独足以放行">
            <el-switch v-model="form.sufficient" />
            <span class="field-hint">关闭时需配合其它参数</span>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="状态">
            <el-radio-group v-model="form.status">
              <el-radio
                v-for="o in paramStatusOptions"
                :key="o.value"
                :value="o.value"
              >
                {{ o.label }}
              </el-radio>
            </el-radio-group>
          </el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="说明">
            <el-input
              v-model="form.remark"
              maxlength="200"
              placeholder="给运营看的说明"
            />
          </el-form-item>
        </el-col>
      </el-row>
    </el-form>
    <template #footer>
      <btn-items
        :items="[
          { preset: 'cancel', onClick: () => closeModal() },
          { preset: 'save', onClick: () => handleSave() }
        ]"
      />
    </template>
  </ele-modal>
</template>

<script setup>
  import { ref, reactive, computed } from 'vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { saveAttributionParam } from '@/api/merchant/attribution';
  import { addBuiltinParam, updateBuiltinParam } from '@/api/admin/param';
  import {
    paramTypeOptions,
    paramStatusOptions,
    emptyParamForm,
    paramToPayload
  } from '../constants';

  const props = defineProps({
    /** merchant | admin */
    mode: { type: String, default: 'merchant' },
    siteId: Number,
    data: Object
  });
  const emit = defineEmits(['done']);
  const { modalProps, closeModal } = useModal();
  const loading = ref(false);
  const formRef = ref(null);

  const isEdit = computed(() => !!props.data);
  const nameLocked = computed(() => isEdit.value);
  const title = computed(() => {
    if (props.mode === 'admin')
      return isEdit.value ? '编辑内置参数' : '新增内置参数';
    if (!isEdit.value) return '新增参数';
    return props.data.source === 'builtin' ? '覆盖内置参数' : '编辑参数';
  });

  const [form] = useFormData({
    ...emptyParamForm(),
    ...(props.data
      ? {
          name: props.data.name,
          paramType: props.data.paramType,
          platform: props.data.platform || '',
          aliasesText: (props.data.aliases || []).join(', '),
          minLen: props.data.minLen,
          maxLen: props.data.maxLen,
          pattern: props.data.pattern || '',
          sufficient: props.data.sufficient,
          status: props.data.status,
          sort: props.data.sort,
          remark: props.data.remark || ''
        }
      : {})
  });

  const rules = reactive({
    name: [
      { required: true, message: '请输入参数名', trigger: 'blur' },
      {
        pattern: /^[A-Za-z0-9_.-]{1,64}$/,
        message: '1-64 位字母、数字、_ . -',
        trigger: 'blur'
      }
    ],
    paramType: [{ required: true, message: '请选择类型', trigger: 'change' }],
    maxLen: [
      {
        validator: (_, v, cb) =>
          Number(v) >= Number(form.minLen)
            ? cb()
            : cb(new Error('最长须不小于最短')),
        trigger: 'change'
      }
    ]
  });

  const handleSave = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) return;
      loading.value = true;
      const payload = paramToPayload(form);
      let req;
      if (props.mode === 'admin') {
        req = isEdit.value
          ? updateBuiltinParam(props.data.paramId, payload)
          : addBuiltinParam(payload).then((r) => r.msg);
      } else {
        req = saveAttributionParam({ siteId: props.siteId, ...payload }).then(
          (r) => r.msg
        );
      }
      req
        .then((msg) => {
          loading.value = false;
          EleMessage.success({ message: msg, plain: true });
          emit('done');
          closeModal();
        })
        .catch((e) => {
          loading.value = false;
          EleMessage.error({ message: e.message, plain: true });
        });
    });
  };
</script>

<style scoped>
  .field-hint {
    margin-left: 8px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
</style>
