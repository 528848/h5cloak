<template>
  <ele-modal form :width="760" :title="isUpdate ? '修改角色' : '添加角色'" :loading="loading" v-bind="modalProps">
    <el-form ref="formRef" :model="form" :rules="rules" label-width="100px" @submit.prevent="">
      <el-row :gutter="16">
        <el-col :sm="12" :xs="24">
          <el-form-item label="角色名称" prop="name">
            <el-input v-model.trim="form.name" clearable maxlength="64" placeholder="请输入角色名称" />
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="权限字符" prop="roleKey">
            <el-input v-model.trim="form.roleKey" clearable :disabled="isUpdate" placeholder="如 finance_viewer" />
            <div class="hint">小写字母开头。保存后自动加上本商户前缀，不会和其他商户或内置角色冲突。</div>
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="状态">
            <el-radio-group v-model="form.status">
              <el-radio value="enabled" label="启用" />
              <el-radio value="disabled" label="停用" />
            </el-radio-group>
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="菜单权限">
        <div>
          <el-button size="small" @click="expandAll">展开全部</el-button>
          <el-button size="small" @click="foldAll">折叠全部</el-button>
          <el-button size="small" @click="checkAll">全选</el-button>
          <el-button size="small" @click="checkOutAll">取消全选</el-button>
        </div>
        <div class="role-menu-tree">
          <el-tree
            ref="treeRef"
            show-checkbox
            node-key="menuId"
            :data="menuData"
            :props="{ label: 'name', children: 'children' }"
            :default-checked-keys="checkedKeys"
          />
        </div>
        <div class="hint">只能分配商户端已有菜单。内置角色的权限由平台维护。</div>
      </el-form-item>
      <el-form-item label="备注">
        <el-input v-model="form.remark" type="textarea" :rows="2" maxlength="200" show-word-limit placeholder="请输入备注" />
      </el-form-item>
    </el-form>
    <template #footer>
      <btn-items :items="[{ preset: 'cancel', onClick: () => closeModal() }, { preset: 'save', onClick: () => handleSave() }]" />
    </template>
  </ele-modal>
</template>

<script setup>
  import { ref, reactive, nextTick } from 'vue';
  import { EleMessage, useModal, toTree, eachTree } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { listMerchantMenus, saveMerchantRole } from '@/api/merchant/user';

  const props = defineProps({ data: Object });
  const emit = defineEmits(['done']);
  const { modalProps, closeModal } = useModal();
  const isUpdate = ref(false);
  const loading = ref(false);
  const formRef = ref(null);
  const treeRef = ref(null);
  const menuData = ref([]);
  const checkedKeys = ref([]);
  const [form, , assignFields] = useFormData({
    roleId: void 0,
    name: '',
    roleKey: '',
    status: 'enabled',
    remark: ''
  });
  const rules = reactive({
    name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
    roleKey: [
      { required: true, message: '请输入权限字符', trigger: 'blur' },
      {
        pattern: /^[a-z][a-z0-9_]{1,20}$/,
        message: '小写字母开头，2 到 21 位，只含字母、数字和下划线',
        trigger: 'blur'
      }
    ]
  });

  const leafIds = (ids, nodes) => {
    const parents = new Set();
    eachTree(nodes, (node) => {
      if (node.children?.length) parents.add(node.menuId);
    });
    return (ids || []).filter((id) => !parents.has(id));
  };
  const expandAll = () => eachTree(menuData.value, (node) => { const n = treeRef.value?.getNode?.(node.menuId); if (n) n.expanded = true; });
  const foldAll = () => eachTree(menuData.value, (node) => { const n = treeRef.value?.getNode?.(node.menuId); if (n) n.expanded = false; });
  const checkAll = () => treeRef.value?.setCheckedNodes?.(menuData.value);
  const checkOutAll = () => treeRef.value?.setCheckedKeys?.([]);

  const handleSave = () => {
    formRef.value?.validate?.(async (valid) => {
      if (!valid) return;
      const menuIds = [...(treeRef.value?.getCheckedKeys?.() || []), ...(treeRef.value?.getHalfCheckedKeys?.() || [])];
      loading.value = true;
      try {
        const key = isUpdate.value ? form.roleKey.replace(/^m\d+_/, '') : form.roleKey;
        await saveMerchantRole(form.roleId || 0, { ...form, roleKey: key, menuIds });
        EleMessage.success({ message: '保存成功', plain: true });
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
    isUpdate.value = true;
    assignFields({ ...props.data, roleKey: String(props.data.roleKey || '').replace(/^m\d+_/, '') });
  }
  listMerchantMenus()
    .then(async (rows) => {
      menuData.value = toTree({
        data: (rows || []).map((m) => ({ ...m, parentId: m.parentId || 0 })),
        idField: 'menuId',
        parentIdField: 'parentId'
      });
      await nextTick();
      if (props.data?.menuIds) {
        checkedKeys.value = leafIds(props.data.menuIds, menuData.value);
        treeRef.value?.setCheckedKeys?.(checkedKeys.value);
      }
    })
    .catch((e) => EleMessage.error({ message: e.message, plain: true }));
</script>

<style scoped>
  .role-menu-tree {
    width: 100%;
    max-height: 320px;
    margin-top: 8px;
    padding: 8px;
    overflow: auto;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }
  .hint {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }
</style>
