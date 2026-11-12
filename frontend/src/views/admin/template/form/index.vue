<template>
  <ele-page hide-footer>
    <section-card class="page-section" title="基本信息" subtitle="无归因时，站点可以选用这里的模板作为审核内容">
      <el-form label-width="120px" :disabled="!canEdit">
        <el-row :gutter="16">
          <el-col :md="12" :xs="24">
            <el-form-item label="编码">
              <el-input v-model.trim="form.code" :disabled="!!form.templateId" placeholder="创建后不可修改" />
            </el-form-item>
          </el-col>
          <el-col :md="12" :xs="24">
            <el-form-item label="名称">
              <el-input v-model.trim="form.name" maxlength="64" placeholder="请输入模板名称" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="说明">
              <el-input v-model.trim="form.description" maxlength="200" placeholder="给商户看的简短说明" />
            </el-form-item>
          </el-col>
          <el-col :md="12" :xs="24">
            <el-form-item label="来源">
              <el-select v-model="form.kind" class="ele-fluid" @change="onKind">
                <el-option label="HTML" value="html" />
                <el-option label="网址" value="url" />
                <el-option label="静态项目" value="static" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :md="12" :xs="24">
            <el-form-item label="状态">
              <el-select v-model="form.status" class="ele-fluid">
                <el-option label="启用" value="enabled" />
                <el-option label="停用" value="disabled" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </section-card>

    <section-card v-if="form.kind === 'html'" class="page-section" title="HTML" subtitle="任意路径都返回这一份页面">
      <el-input
        v-model="form.html"
        type="textarea"
        :rows="16"
        placeholder="可使用 {{.siteName}} 和 {{.slogan}}"
      />
    </section-card>

    <section-card v-else-if="form.kind === 'url'" class="page-section" title="网址" subtitle="网关按访客的路径和参数回源，地址栏仍是商户域名">
      <el-form label-width="120px">
        <el-form-item label="根地址">
          <el-input v-model.trim="form.url" placeholder="https://review.example.com" />
        </el-form-item>
        <el-form-item label="回源 Host">
          <el-input v-model.trim="form.hostHeader" placeholder="不填则使用网址里的主机名" />
        </el-form-item>
        <el-form-item label="超时">
          <el-input-number v-model="form.timeoutMs" :min="1000" :max="30000" :step="1000" />
          <span class="hint">毫秒</span>
        </el-form-item>
      </el-form>
    </section-card>

    <section-card v-else class="page-section" title="静态项目" subtitle="上传 zip 后按路径提供文件，不执行服务端脚本">
      <el-form label-width="120px">
        <el-form-item label="入口文件">
          <el-input v-model.trim="form.entry" placeholder="index.html" />
        </el-form-item>
        <el-form-item label="单页回退">
          <div class="switch-line">
            <el-switch v-model="form.spaFallback" />
            <span class="hint">关闭时，不存在的路径返回 404。打开后改为返回入口文件。</span>
          </div>
        </el-form-item>
        <el-form-item label="压缩包">
          <el-upload :auto-upload="false" :limit="1" accept=".zip" :on-change="onFile" :on-remove="() => (file = null)">
            <el-button>选择 zip</el-button>
          </el-upload>
          <div class="hint">小于 50MB，包内需要有入口文件。保存基本信息后才会上传。</div>
        </el-form-item>
      </el-form>
      <el-table v-if="versions.length" :data="versions" border>
        <el-table-column prop="version" label="版本" width="80" />
        <el-table-column prop="entry" label="入口" min-width="140" />
        <el-table-column prop="fileCount" label="文件数" width="90" />
        <el-table-column label="当前" width="80">
          <template #default="{ row }">
            <el-tag v-if="row.current" type="success" effect="light" size="small">当前</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="90">
          <template #default="{ row }">
            <el-button v-if="!row.current" link type="primary" @click="rollback(row)">回滚</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section-card>

    <ele-card>
      <div class="form-footer">
        <el-button @click="router.push('/admin/template')">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">保存</el-button>
      </div>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { computed, reactive, ref } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage } from 'ele-admin-plus';
  import { usePermission } from '@/utils/use-permission';
  import {
    getTemplate,
    rollbackTemplate,
    saveTemplate,
    uploadTemplatePackage
  } from '@/api/admin/platform';

  defineOptions({ name: 'AdminTemplateForm' });

  const route = useRoute();
  const router = useRouter();
  const { hasPermission } = usePermission();
  const canEdit = computed(() => hasPermission('system:template:edit'));
  const saving = ref(false);
  const file = ref(null);
  const versions = ref([]);
  const loadedKind = ref('html');
  const form = reactive({
    templateId: 0,
    code: '',
    name: '',
    description: '',
    kind: 'html',
    status: 'enabled',
    html: '<!doctype html><title>{{.siteName}}</title><p>{{.slogan}}</p>',
    url: '',
    hostHeader: '',
    timeoutMs: 8000,
    entry: 'index.html',
    spaFallback: false
  });

  const onFile = (uploadFile) => {
    file.value = uploadFile.raw;
  };

  const onKind = (next) => {
    const prev = loadedKind.value;
    if (!prev || prev === next) {
      loadedKind.value = next;
      return;
    }
    ElMessageBox.confirm(
      '切换来源后，保存时会按新来源写入。未保存的另一类内容不会提交。是否继续？',
      '系统提示',
      { type: 'warning', draggable: true }
    )
      .then(() => {
        loadedKind.value = next;
      })
      .catch(() => {
        form.kind = prev;
      });
  };

  const load = async () => {
    const id = route.query.id;
    if (!id) return;
    const data = await getTemplate(id);
    Object.assign(form, data);
    form.templateId = data.templateId;
    loadedKind.value = data.kind || 'html';
    versions.value = data.versions || [];
  };

  const submit = async () => {
    saving.value = true;
    try {
      const body = await saveTemplate(form.templateId, { ...form });
      const id = body.data?.templateId || form.templateId;
      if (form.kind === 'static' && file.value) {
        await uploadTemplatePackage(id, file.value, form.entry, form.spaFallback);
      }
      EleMessage.success({ message: body.msg || '保存成功', plain: true });
      router.push('/admin/template');
    } catch (e) {
      EleMessage.error({ message: e.message, plain: true });
    } finally {
      saving.value = false;
    }
  };

  const rollback = (row) => {
    ElMessageBox.confirm(`回滚到版本 ${row.version}？选用该模板的站点会在数秒内切换。`, '系统提示', {
      type: 'warning',
      draggable: true
    })
      .then(() => rollbackTemplate(form.templateId, row.version))
      .then((msg) => {
        EleMessage.success({ message: msg, plain: true });
        return load();
      })
      .catch((e) => {
        if (e?.message) EleMessage.error({ message: e.message, plain: true });
      });
  };

  load().catch((e) => EleMessage.error({ message: e.message, plain: true }));
</script>

<style scoped>
  .hint {
    margin-left: 8px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }

  .switch-line {
    display: flex;
    align-items: center;
  }

  .form-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }
</style>
