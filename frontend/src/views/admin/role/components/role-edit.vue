<template>
  <ele-modal
    :form="true"
    :width="760"
    :title="isUpdate ? '修改角色' : '添加角色'"
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
          <el-form-item label="角色名称" prop="name">
            <el-input
              clearable
              v-model="form.name"
              placeholder="请输入角色名称"
            />
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="权限字符" prop="roleKey">
            <el-input
              clearable
              :disabled="isUpdate"
              v-model="form.roleKey"
              placeholder="小写字母开头，如 ops_viewer"
            />
          </el-form-item>
        </el-col>
        <el-col :sm="12" :xs="24">
          <el-form-item label="状态">
            <el-radio-group
              v-model="form.status"
              :disabled="form.roleKey === 'admin'"
            >
              <el-radio value="enabled" label="启用" />
              <el-radio value="disabled" label="停用" />
            </el-radio-group>
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="菜单权限">
        <div style="line-height: 1">
          <el-button size="small" class="ele-btn-icon" @click="expandAll"
            >展开全部</el-button
          >
          <el-button size="small" class="ele-btn-icon" @click="foldAll"
            >折叠全部</el-button
          >
          <el-button size="small" class="ele-btn-icon" @click="checkAll"
            >全选</el-button
          >
          <el-button size="small" class="ele-btn-icon" @click="checkOutAll"
            >取消全选</el-button
          >
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
        <div class="hint"
          >超级管理员始终可见全部菜单，这里的勾选对其不生效。</div
        >
      </el-form-item>
      <el-form-item label="备注">
        <el-input
          :rows="3"
          type="textarea"
          v-model="form.remark"
          placeholder="请输入备注"
        />
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
  import { ref, reactive, nextTick } from 'vue';
  import { EleMessage, useModal, toTree, eachTree } from 'ele-admin-plus';
  import { useFormData } from '@/utils/use-form-data';
  import { listMenus, saveRole } from '@/api/admin/platform';

  const props = defineProps({
    data: Object
  });
  const emit = defineEmits(['done']);
  const { modalProps, closeModal } = useModal();

  const isUpdate = ref(false);
  const loading = ref(false);
  const formRef = ref(null);
  const treeRef = ref(null);
  const menuData = ref([]);
  const checkedKeys = ref([]);

  const [form, _reset, assignFields] = useFormData({
    roleId: void 0,
    name: '',
    roleKey: '',
    status: 'enabled',
    remark: ''
  });

  const rules = reactive({
    name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
    roleKey: [{ required: true, message: '请输入权限字符', trigger: 'blur' }]
  });

  const leafIds = (ids, nodes) => {
    const parents = new Set();
    eachTree(nodes, (node) => {
      if (node.children?.length) {
        parents.add(node.menuId);
      }
    });
    return (ids || []).filter((id) => !parents.has(id));
  };

  const expandAll = () => {
    eachTree(menuData.value, (node) => {
      const item = treeRef.value?.getNode?.(node.menuId);
      if (item) {
        item.expanded = true;
      }
    });
  };
  const foldAll = () => {
    eachTree(menuData.value, (node) => {
      const item = treeRef.value?.getNode?.(node.menuId);
      if (item) {
        item.expanded = false;
      }
    });
  };
  const checkAll = () => treeRef.value?.setCheckedNodes?.(menuData.value);
  const checkOutAll = () => treeRef.value?.setCheckedKeys?.([]);

  const handleSave = () => {
    formRef.value?.validate?.(async (valid) => {
      if (!valid) {
        return;
      }
      const menuIds = [
        ...(treeRef.value?.getCheckedKeys?.() || []),
        ...(treeRef.value?.getHalfCheckedKeys?.() || [])
      ];
      loading.value = true;
      try {
        await saveRole(form.roleId || 0, { ...form, menuIds });
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

  listMenus()
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

  if (props.data) {
    isUpdate.value = true;
    assignFields(props.data);
  }
</script>

<style scoped>
  .role-menu-tree {
    width: 100%;
    height: 280px;
    margin-top: 8px;
    overflow: auto;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 4px;
    padding: 6px 0;
  }
  .hint {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.4;
    margin-top: 6px;
  }
</style>
