<!-- 开通 / 编辑商户 -->
<template>
  <ele-modal
    :form="true"
    :width="680"
    :title="isEdit ? '编辑商户' : '开通商户'"
    :loading="loading"
    v-bind="modalProps"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="100px"
      @submit.prevent=""
    >
      <el-row :gutter="16">
        <el-col :sm="12" :xs="24">
          <el-form-item label="商户名称" prop="name">
            <el-input
              v-model.trim="form.name"
              clearable
              maxlength="50"
              show-word-limit
              placeholder="请输入商户名称"
            />
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="到期时间" prop="expireAt">
            <el-date-picker
              v-model="form.expireAt"
              type="datetime"
              value-format="YYYY-MM-DD HH:mm:ss"
              placeholder="不填表示长期"
              class="ele-fluid"
            />
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="联系人" prop="contactName">
            <el-input
              v-model.trim="form.contactName"
              clearable
              placeholder="请输入联系人"
            />
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="联系电话" prop="contactPhone">
            <el-input
              v-model.trim="form.contactPhone"
              clearable
              placeholder="请输入联系电话"
            />
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="联系邮箱" prop="contactEmail">
            <el-input
              v-model.trim="form.contactEmail"
              clearable
              placeholder="请输入联系邮箱"
            />
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="时区" prop="timezone">
            <el-select
              v-model="form.timezone"
              class="ele-fluid"
              filterable
              placeholder="请选择时区"
            >
              <el-option
                v-for="tz in timezones"
                :key="tz"
                :label="tz"
                :value="tz"
              />
            </el-select>
            <div class="muted-text" style="margin-top: 6px; font-size: 12px; line-height: 1.5">
              商户的业务时区，用来划分报表里的「今天」。目前只保存在商户资料中，报表尚未按此时区计算日期。
            </div>
          </el-form-item>
        </el-col>
      </el-row>

      <template v-if="!isEdit">
        <el-divider content-position="left">管理员账号</el-divider>
        <el-row :gutter="16">
          <el-col :sm="12" :xs="24">
            <el-form-item label="登录账号" prop="adminUsername">
              <el-input
                v-model.trim="form.adminUsername"
                clearable
                placeholder="3-32 位字母数字"
                autocomplete="off"
              />
            </el-form-item>
          </el-col>
          <el-col :sm="12" :xs="24">
            <el-form-item label="初始密码" prop="adminPassword">
              <el-input
                v-model="form.adminPassword"
                type="password"
                show-password
                placeholder="8-64 位，须包含字母和数字"
                autocomplete="new-password"
              />
            </el-form-item>
          </el-col>
        </el-row>
        <el-alert type="info" show-icon :closable="false">
          额度与功能开关按平台默认值开通（判定 100 万/月、站点 5、域名
          20、归因码 1000、子账号 10、日志 90
          天），开通后可在「额度与功能」中调整。
        </el-alert>
      </template>

      <el-form-item label="备注" style="margin-top: 16px">
        <el-input
          v-model="form.remark"
          type="textarea"
          :rows="2"
          maxlength="200"
          show-word-limit
          placeholder="内部备注，商户不可见"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <btn-items
        :items="[
          { preset: 'cancel', onClick: () => handleCancel() },
          { preset: 'save', onClick: () => handleSave() }
        ]"
      />
    </template>
  </ele-modal>
</template>

<script setup>
  import { ref, reactive, computed } from 'vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage, useModal, emailReg } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { addMerchant, updateMerchant } from '@/api/admin/merchant';

  const props = defineProps({ data: Object });
  const emit = defineEmits(['done']);
  const { modalProps, closeModal } = useModal();

  const isEdit = computed(() => !!props.data?.merchantId);
  const loading = ref(false);
  const formRef = ref(null);

  const timezones = [
    'Asia/Shanghai',
    'Asia/Hong_Kong',
    'Asia/Singapore',
    'Asia/Tokyo',
    'Asia/Dubai',
    'Europe/London',
    'Europe/Berlin',
    'America/New_York',
    'America/Los_Angeles',
    'UTC'
  ];

  const [form] = useFormData({
    name: props.data?.merchantName ?? '',
    contactName: props.data?.contactName ?? '',
    contactEmail: props.data?.contactEmail ?? '',
    contactPhone: props.data?.contactPhone ?? '',
    expireAt: props.data?.expireAt ?? '',
    timezone: props.data?.timezone ?? 'Asia/Shanghai',
    remark: props.data?.remark ?? '',
    adminUsername: '',
    adminPassword: ''
  });

  const rules = reactive({
    name: [{ required: true, message: '请输入商户名称', trigger: 'blur' }],
    contactEmail: [
      { pattern: emailReg, message: '邮箱格式不正确', trigger: 'blur' }
    ],
    adminUsername: [
      { required: true, message: '请输入管理员账号', trigger: 'blur' },
      {
        pattern: /^[A-Za-z0-9_.@-]{3,32}$/,
        message: '3-32 位，仅字母、数字、_ . - @',
        trigger: 'blur'
      }
    ],
    adminPassword: [
      { required: true, message: '请输入初始密码', trigger: 'blur' },
      {
        pattern: /^(?=.*[A-Za-z])(?=.*\d).{8,64}$/,
        message: '8-64 位，须包含字母和数字',
        trigger: 'blur'
      }
    ]
  });

  const handleCancel = () => closeModal();

  const handleSave = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) return;
      loading.value = true;
      const payload = { ...form, expireAt: form.expireAt || '' };
      const req = isEdit.value
        ? updateMerchant(props.data.merchantId, payload).then((msg) => ({
            msg
          }))
        : addMerchant(payload);
      req
        .then((res) => {
          loading.value = false;
          if (!isEdit.value && res.data) {
            ElMessageBox.alert(
              `商户「${res.data.merchantName}」已开通。\n管理员账号：${res.data.adminUsername}\n初始密码：创建时填写的密码，请通过安全渠道告知商户。`,
              '开通成功',
              { confirmButtonText: '知道了', type: 'success' }
            );
          } else {
            EleMessage.success({ message: res.msg, plain: true });
          }
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
