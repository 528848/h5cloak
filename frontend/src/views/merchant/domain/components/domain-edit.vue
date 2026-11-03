<!-- 添加域名 -->
<template>
  <ele-modal
    :form="true"
    :width="560"
    title="添加域名"
    :loading="loading"
    v-bind="modalProps"
  >
    <el-alert
      type="info"
      show-icon
      :closable="false"
      style="margin-bottom: 16px"
      :title="`托管域名请把 CNAME 或 TXT 指到网关入口 ${entryHost || '（加载中）'}。自有域名不要求解析到网关。`"
    />
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="96px"
      @submit.prevent=""
    >
      <el-form-item v-if="!siteId" label="所属站点" prop="siteId">
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
      <el-form-item label="域名" prop="domain">
        <el-input
          v-model.trim="form.domain"
          placeholder="example.com，不要带协议和路径"
        />
      </el-form-item>
      <el-form-item label="域名类型" prop="domainType">
        <el-radio-group v-model="form.domainType">
          <el-radio
            v-for="o in domainTypeOptions"
            :key="o.value"
            :value="o.value"
          >
            {{ o.label }}
          </el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item v-if="form.domainType === 'hosted'" label="校验方式">
        <el-radio-group v-model="form.verifyType">
          <el-radio value="CNAME">CNAME</el-radio>
          <el-radio value="TXT">TXT</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="备注">
        <el-input
          v-model="form.remark"
          type="textarea"
          :rows="2"
          maxlength="200"
          placeholder="请输入备注"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <btn-items
        :items="[
          { preset: 'cancel', onClick: () => closeModal() },
          { preset: 'save', title: '添加', onClick: () => handleSave() }
        ]"
      />
    </template>
  </ele-modal>
</template>

<script setup>
  import { ref, reactive, onMounted } from 'vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { listSiteOptions, getSiteGuide } from '@/api/merchant/site';
  import { addMerchantDomain } from '@/api/merchant/domain';
  import { domainTypeOptions } from '../../site/constants';

  const props = defineProps({ siteId: Number });
  const emit = defineEmits(['done']);
  const { modalProps, closeModal } = useModal();
  const loading = ref(false);
  const formRef = ref(null);
  const sites = ref([]);
  const entryHost = ref('');

  const [form] = useFormData({
    siteId: props.siteId || undefined,
    domain: '',
    domainType: 'hosted',
    verifyType: 'CNAME',
    remark: ''
  });

  const rules = reactive({
    siteId: [{ required: true, message: '请选择站点', trigger: 'change' }],
    domain: [
      { required: true, message: '请输入域名', trigger: 'blur' },
      {
        pattern: /^(?!https?:)[a-z0-9.-]+\.[a-z]{2,}$/i,
        message: '请输入域名，不要带协议和路径',
        trigger: 'blur'
      }
    ],
    domainType: [
      { required: true, message: '请选择域名类型', trigger: 'change' }
    ]
  });

  const handleSave = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) return;
      loading.value = true;
      const target = props.siteId || form.siteId;
      addMerchantDomain(target, {
        domain: form.domain,
        domainType: form.domainType,
        verifyType: form.domainType === 'own' ? 'CNAME' : form.verifyType,
        remark: form.remark
      })
        .then((res) => {
          loading.value = false;
          EleMessage.success({ message: res.msg, plain: true });
          emit('done', res.data);
          closeModal();
        })
        .catch((e) => {
          loading.value = false;
          EleMessage.error({ message: e.message, plain: true });
        });
    });
  };

  onMounted(() => {
    getSiteGuide()
      .then((d) => (entryHost.value = d.entryHost))
      .catch(() => {});
    if (!props.siteId) {
      listSiteOptions()
        .then((rows) => (sites.value = rows || []))
        .catch(() => {});
    }
  });
</script>
