<!-- 登记 / 编辑分销码 -->
<template>
  <ele-modal
    :form="true"
    :width="600"
    :title="isEdit ? '编辑分销码' : '登记分销码'"
    :loading="loading"
    v-bind="modalProps"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="96px"
      @submit.prevent=""
    >
      <el-row :gutter="12">
        <el-col v-if="!isEdit" :span="12">
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
        </el-col>
        <el-col :span="12">
          <el-form-item label="类型" prop="codeType">
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
        </el-col>
        <el-col :span="12">
          <el-form-item label="码值" prop="code">
            <el-input
              v-model.trim="form.code"
              :disabled="isEdit"
              placeholder="2-64 位字母数字 _ -"
              class="mono"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="名称">
            <el-input
              v-model.trim="form.name"
              maxlength="64"
              placeholder="分销商 / 推广员 / 活动名"
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
              placeholder="如 tiktok、kol、email"
              class="ele-fluid"
            >
              <el-option v-for="c in channels" :key="c" :label="c" :value="c" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="状态">
            <el-radio-group v-model="form.status">
              <el-radio value="enabled">启用</el-radio>
              <el-radio value="disabled">停用</el-radio>
            </el-radio-group>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="生效时间">
            <el-date-picker
              v-model="form.startsAt"
              type="datetime"
              value-format="YYYY-MM-DD HH:mm:ss"
              placeholder="留空立即生效"
              class="ele-fluid"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="失效时间">
            <el-date-picker
              v-model="form.expiresAt"
              type="datetime"
              value-format="YYYY-MM-DD HH:mm:ss"
              placeholder="留空长期有效"
              class="ele-fluid"
            />
          </el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="允许路径">
            <el-input
              v-model="form.allowedPathsText"
              type="textarea"
              :rows="2"
              placeholder="每行一个路径前缀，如 /landing。留空表示任意路径"
            />
          </el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="备注">
            <el-input
              v-model="form.remark"
              maxlength="200"
              placeholder="请输入备注"
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
  import { addCode, updateCode } from '@/api/merchant/code';
  import { codeTypeOptions } from '../constants';

  const props = defineProps({
    data: Object,
    siteId: Number,
    sites: { type: Array, default: () => [] },
    channels: { type: Array, default: () => [] }
  });
  const emit = defineEmits(['done']);
  const { modalProps, closeModal } = useModal();
  const isEdit = computed(() => !!props.data?.codeId);
  const loading = ref(false);
  const formRef = ref(null);

  const [form] = useFormData({
    siteId: props.data?.siteId ?? props.siteId ?? undefined,
    codeType: props.data?.codeType ?? 'distributor',
    code: props.data?.code ?? '',
    name: props.data?.name ?? '',
    channel: props.data?.channel ?? '',
    status: props.data?.status ?? 'enabled',
    startsAt: props.data?.startsAt || '',
    expiresAt: props.data?.expiresAt || '',
    allowedPathsText: (props.data?.allowedPaths || []).join('\n'),
    remark: props.data?.remark ?? ''
  });

  const rules = reactive({
    siteId: [{ required: true, message: '请选择站点', trigger: 'change' }],
    code: [
      { required: true, message: '请输入码值', trigger: 'blur' },
      {
        pattern: /^[A-Za-z0-9_-]{2,64}$/,
        message: '2-64 位字母、数字、_ -',
        trigger: 'blur'
      }
    ]
  });

  const handleSave = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) return;
      loading.value = true;
      const payload = {
        siteId: form.siteId,
        codeType: form.codeType,
        code: form.code,
        name: form.name,
        channel: form.channel,
        status: form.status,
        startsAt: form.startsAt || '',
        expiresAt: form.expiresAt || '',
        allowedPaths: String(form.allowedPathsText || '')
          .split(/\r?\n/)
          .map((s) => s.trim())
          .filter(Boolean),
        remark: form.remark
      };
      const req = isEdit.value
        ? updateCode(props.data.codeId, payload)
        : addCode(payload).then((r) => r.msg);
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
