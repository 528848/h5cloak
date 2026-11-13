<!-- 文本 / CSV 导入归因码 -->
<template>
  <ele-modal
    :form="true"
    :width="640"
    title="导入归因码"
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
        <el-col :span="12">
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
      </el-row>
      <el-form-item label="默认渠道">
        <el-input
          v-model.trim="form.channel"
          placeholder="行内未填写渠道时使用"
        />
      </el-form-item>
      <el-form-item label="内容" prop="content">
        <el-input
          v-model="form.content"
          type="textarea"
          :rows="10"
          class="mono"
          placeholder="每行一个：码值,名称,渠道,失效时间
ALICE01,Alice,tiktok,2030-01-01
BOB02,Bob"
        />
      </el-form-item>
      <el-form-item label-width="0">
        <el-upload
          :auto-upload="false"
          :show-file-list="false"
          accept=".csv,.txt"
          :on-change="onFile"
        >
          <el-button plain>从 CSV / TXT 文件读取</el-button>
        </el-upload>
        <span class="field-hint"
          >单次最多 10000 行，已登记或格式错误的行会被跳过并列出</span
        >
      </el-form-item>
    </el-form>
    <el-alert
      v-if="result"
      :type="result.skipped ? 'warning' : 'success'"
      show-icon
      :closable="false"
      :title="`导入 ${result.created} 个，跳过 ${result.skipped} 个`"
    >
      <ul v-if="result.conflicts.length" class="conflicts">
        <li v-for="(c, i) in result.conflicts.slice(0, 50)" :key="i">
          <code-cell :text="c.code" />
          <span v-if="c.line">（第 {{ c.line }} 行）</span>
          ：{{ c.reason }}
        </li>
        <li v-if="result.conflicts.length > 50"
          >… 其余 {{ result.conflicts.length - 50 }} 条</li
        >
      </ul>
    </el-alert>
    <template #footer>
      <btn-items
        :items="[
          {
            preset: 'cancel',
            title: result ? '关闭' : '取消',
            onClick: () => closeModal()
          },
          { preset: 'save', title: '导入', onClick: () => handleSave() }
        ]"
      />
    </template>
  </ele-modal>
</template>

<script setup>
  import { ref, reactive } from 'vue';
  import CodeCell from '@/components/CodeCell.vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { importCodes } from '@/api/merchant/code';

  const props = defineProps({
    siteId: Number,
    sites: { type: Array, default: () => [] }
  });
  const emit = defineEmits(['done']);
  const { modalProps, closeModal } = useModal();
  const loading = ref(false);
  const formRef = ref(null);
  const result = ref(null);

  const [form] = useFormData({
    siteId: props.siteId ?? undefined,
    codeType: 'distributor',
    channel: '',
    content: ''
  });

  const rules = reactive({
    siteId: [{ required: true, message: '请选择站点', trigger: 'change' }],
    content: [{ required: true, message: '请粘贴或读取内容', trigger: 'blur' }]
  });

  const onFile = (file) => {
    const reader = new FileReader();
    reader.onload = () => {
      form.content = String(reader.result || '');
    };
    reader.readAsText(file.raw, 'utf-8');
  };

  const handleSave = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) return;
      loading.value = true;
      importCodes({ ...form })
        .then((res) => {
          loading.value = false;
          result.value = res.data;
          EleMessage.success({ message: res.msg, plain: true });
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
  .field-hint {
    margin-left: 12px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .conflicts {
    margin: 8px 0 0;
    padding-left: 16px;
    max-height: 160px;
    overflow: auto;
    font-size: 12px;
  }
</style>
