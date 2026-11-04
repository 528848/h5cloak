<template>
  <ele-modal
    :form="true"
    :width="640"
    title="修改菜单"
    :loading="loading"
    v-bind="modalProps"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="88px"
      @submit.prevent=""
    >
      <el-row :gutter="16">
        <el-col :sm="12" :xs="24">
          <el-form-item label="菜单名称" prop="name">
            <el-input
              clearable
              v-model="form.name"
              placeholder="请输入菜单名称"
            />
          </el-form-item>
          <el-form-item label="显示排序" prop="sort">
            <el-input-number
              v-model="form.sort"
              :min="0"
              :max="9999"
              controls-position="right"
              class="ele-fluid"
            />
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="菜单图标">
            <icon-select
              clearable
              filterable="popper"
              :popper-width="460"
              :popper-height="388"
              placeholder="请选择图标"
              v-model="form.icon"
              :disabled="form.menuType === 'F'"
            />
          </el-form-item>
          <el-form-item label="显示状态">
            <el-radio-group v-model="form.visible">
              <el-radio :value="true" label="显示" />
              <el-radio :value="false" label="隐藏" />
            </el-radio-group>
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="权限字符">
        <el-input :model-value="form.perms" disabled />
      </el-form-item>
      <el-form-item v-if="form.path" label="路由地址">
        <el-input :model-value="form.path" disabled />
      </el-form-item>
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
  import { ref, reactive } from 'vue';
  import { EleMessage, useModal } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import IconSelect from '@/components/IconSelect/index.vue';
  import { saveMenu } from '@/api/admin/platform';

  const props = defineProps({ data: Object });
  const emit = defineEmits(['done']);
  const { modalProps, closeModal } = useModal();
  const loading = ref(false);
  const formRef = ref(null);
  const [form, _reset, assignFields] = useFormData({
    menuId: void 0,
    name: '',
    icon: '',
    sort: 0,
    visible: true,
    menuType: 'C',
    perms: '',
    path: ''
  });
  const rules = reactive({
    name: [{ required: true, message: '请输入菜单名称', trigger: 'blur' }]
  });

  const handleSave = () => {
    formRef.value?.validate?.(async (valid) => {
      if (!valid) {
        return;
      }
      loading.value = true;
      try {
        const msg = await saveMenu(form.menuId, {
          name: form.name,
          icon: form.icon || '',
          sort: form.sort,
          visible: form.visible
        });
        EleMessage.success({ message: msg, plain: true });
        closeModal();
        emit('done');
      } catch (e) {
        EleMessage.error({ message: e.message, plain: true });
      } finally {
        loading.value = false;
      }
    });
  };

  if (props.data) {
    assignFields(props.data);
  }
</script>
