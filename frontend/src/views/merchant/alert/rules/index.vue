<template>
  <ele-page>
    <ele-card class="page-section" :body-style="{ padding: '16px 18px' }">
      <el-alert
        type="info"
        show-icon
        :closable="false"
        title="每类告警一条默认规则，修改后只对本商户生效，可随时恢复默认。流量类按滚动窗口评估；源站、域名、证书在问题消失后自动关闭。"
      />
    </ele-card>
    <ele-card search-form class="page-section">
      <el-form label-width="72px" @keyup.enter="reload" @submit.prevent="">
        <el-row :gutter="12">
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label="规则名称">
              <el-input v-model.trim="keyword" clearable placeholder="请输入规则名称" />
            </el-form-item>
          </el-col>
          <el-col :lg="6" :md="8" :sm="12" :xs="24">
            <el-form-item label-width="0">
              <btn-items
                :wrap="false"
                :items="[
                  { preset: 'search', onClick: () => reload() },
                  { preset: 'reset', onClick: () => reset() }
                ]"
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </ele-card>
    <ele-card class="table-section" :body-style="{ paddingTop: '8px' }">
      <ele-pro-table
        ref="tableRef"
        row-key="type"
        :columns="columns"
        :datasource="datasource"
        :pagination="false"
        :show-overflow-tooltip="true"
        cache-key="merchantAlertRuleTable"
      >
        <template #enabled="{ row }">
          <el-switch
            size="small"
            :model-value="row.enabled"
            :disabled="!canEdit"
            :before-change="() => toggle(row)"
          />
        </template>
        <template #name="{ row }">
          <span>{{ row.name }}</span>
          <el-tag v-if="row.custom" size="small" type="primary" effect="plain" class="name-tag">已自定义</el-tag>
        </template>
        <template #severity="{ row }">
          <el-tag :type="severityType(row.severity)" size="small" effect="light">
            {{ severityLabel(row.severity) }}
          </el-tag>
        </template>
        <template #action="{ row }">
          <el-space :size="6" spacer="|">
            <el-button v-permission="'alert:edit'" link type="primary" @click="open(row)">编辑</el-button>
            <el-button v-if="row.custom" v-permission="'alert:edit'" link type="warning" @click="resetRule(row)">
              恢复默认
            </el-button>
          </el-space>
        </template>
      </ele-pro-table>
    </ele-card>

    <ele-modal form :width="560" :title="`编辑规则：${editing?.name || ''}`" :loading="saving" v-model="editVisible">
      <el-form v-if="editing" label-width="110px" @submit.prevent="">
        <el-form-item label="级别">
          <el-radio-group v-model="editing.severity">
            <el-radio v-for="item in severityOptions" :key="item.value" :value="item.value">
              {{ item.label }}
            </el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="fieldsOf(editing.type).includes('pct')" label="百分比阈值">
          <el-input-number v-model="editing.threshold.pct" :min="1" :max="100" />
          <span class="hint">%</span>
        </el-form-item>
        <el-form-item v-if="fieldsOf(editing.type).includes('count')" label="次数阈值">
          <el-input-number v-model="editing.threshold.count" :min="1" :max="1000000" />
          <span class="hint">窗口内达到该次数才告警</span>
        </el-form-item>
        <el-form-item v-if="fieldsOf(editing.type).includes('days')" label="提前天数">
          <el-input-number v-model="editing.threshold.days" :min="1" :max="90" />
          <span class="hint">天。到期前提醒</span>
        </el-form-item>
        <el-form-item v-if="fieldsOf(editing.type).includes('minVisits')" label="最少样本">
          <el-input-number v-model="editing.threshold.minVisits" :min="1" :max="100000" />
          <span class="hint">窗口内访问少于该值不评估</span>
        </el-form-item>
        <el-form-item label="评估窗口">
          <el-input-number v-model="editing.windowMinutes" :min="5" :max="1440" />
          <span class="hint">分钟</span>
        </el-form-item>
        <el-form-item label="冷却时间">
          <el-input-number v-model="editing.cooldownMinutes" :min="0" :max="10080" />
          <span class="hint">分钟内同一目标不重复告警</span>
        </el-form-item>
        <el-form-item label="通知渠道">
          <el-checkbox-group v-model="editing.channels">
            <el-checkbox v-for="item in channelOptions" :key="item.value" :value="item.value" :label="item.label" />
          </el-checkbox-group>
          <div class="hint block">还要在「通知设置」里打开对应渠道，这里的勾选才会发出去。</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <btn-items
          :items="[
            { preset: 'cancel', onClick: () => (editVisible = false) },
            { preset: 'save', props: { loading: saving }, onClick: () => save(editing, true) }
          ]"
        />
      </template>
    </ele-modal>
  </ele-page>
</template>

<script setup>
  import { ref } from 'vue';
  import { ElMessageBox } from 'element-plus';
  import { EleMessage } from 'ele-admin-plus';
  import { usePermission } from '@/utils/use-permission';
  import { deleteAlertRule, getAlertRules, saveAlertRule } from '@/api/merchant/ops';
  import {
    channelOptions,
    channelText,
    severityLabel,
    severityOptions,
    severityType,
    thresholdText
  } from '../constants';

  defineOptions({ name: 'MerchantAlertRules' });

  const { hasPermission } = usePermission();
  const canEdit = hasPermission('alert:edit');
  const tableRef = ref(null);
  const keyword = ref('');
  const rules = ref([]);
  const defaults = ref([]);
  const editing = ref(null);
  const editVisible = ref(false);
  const saving = ref(false);

  const columns = [
    {
      prop: 'enabled',
      label: '启用',
      width: 80,
      align: 'center',
      slot: 'enabled',
      formatter: (row) => (row.enabled ? '启用' : '停用')
    },
    {
      prop: 'name',
      label: '规则名称',
      minWidth: 180,
      slot: 'name',
      formatter: (row) => row.name
    },
    {
      prop: 'severity',
      label: '级别',
      width: 90,
      align: 'center',
      slot: 'severity',
      formatter: (row) => severityLabel(row.severity)
    },
    {
      prop: 'desc',
      label: '说明',
      minWidth: 220,
      formatter: (row) => defaults.value.find((item) => item.type === row.type)?.desc || '-'
    },
    {
      prop: 'threshold',
      label: '阈值',
      minWidth: 160,
      formatter: (row) => thresholdText(row)
    },
    {
      prop: 'windowMinutes',
      label: '窗口',
      width: 100,
      align: 'center',
      formatter: (row) => `${row.windowMinutes} 分钟`
    },
    {
      prop: 'cooldownMinutes',
      label: '冷却',
      width: 100,
      align: 'center',
      formatter: (row) => `${row.cooldownMinutes} 分钟`
    },
    {
      prop: 'channels',
      label: '通知渠道',
      minWidth: 180,
      formatter: (row) => channelText(row.channels)
    },
    {
      columnKey: 'action',
      label: '操作',
      width: 150,
      align: 'center',
      slot: 'action',
      fixed: 'right',
      hideInPrint: true,
      hideInExport: true
    }
  ];

  const fieldsOf = (type) => defaults.value.find((item) => item.type === type)?.fields || [];

  const load = () =>
    getAlertRules()
      .then((data) => {
        rules.value = (data?.rules || []).map((item) => ({
          ...item,
          threshold: { ...(item.threshold || {}) },
          channels: [...(item.channels || [])]
        }));
        defaults.value = data?.defaults || [];
      })
      .catch((e) => {
        EleMessage.error({ message: e.message, plain: true });
      });

  const datasource = async () => {
    if (!rules.value.length) await load();
    const text = keyword.value;
    return text ? rules.value.filter((item) => item.name.includes(text)) : rules.value;
  };

  const reload = async () => {
    await load();
    tableRef.value?.reload?.();
  };

  const reset = () => {
    keyword.value = '';
    reload();
  };

  const payload = (row) => ({
    siteId: row.siteId || 0,
    type: row.type,
    enabled: row.enabled,
    severity: row.severity,
    threshold: row.threshold,
    windowMinutes: row.windowMinutes,
    cooldownMinutes: row.cooldownMinutes,
    channels: row.channels
  });

  const toggle = (row) => {
    const next = !row.enabled;
    return saveAlertRule({ ...payload(row), enabled: next })
      .then(() => {
        row.enabled = next;
        EleMessage.success({ message: next ? '已启用' : '已停用', plain: true });
        return true;
      })
      .catch((e) => {
        EleMessage.error({ message: e.message, plain: true });
        return false;
      });
  };

  const open = (row) => {
    editing.value = JSON.parse(JSON.stringify(row));
    editVisible.value = true;
  };

  const save = (row, closeModal = false) => {
    saving.value = true;
    saveAlertRule(payload(row))
      .then(() => {
        EleMessage.success({ message: '已保存', plain: true });
        if (closeModal) editVisible.value = false;
        reload();
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => {
        saving.value = false;
      });
  };

  const resetRule = (row) => {
    ElMessageBox.confirm(`确认把「${row.name}」恢复为平台默认规则？`, '恢复默认', {
      type: 'warning',
      draggable: true
    })
      .then(() => deleteAlertRule(row.id))
      .then(() => {
        EleMessage.success({ message: '已恢复默认', plain: true });
        reload();
      })
      .catch((e) => {
        if (e?.message) EleMessage.error({ message: e.message, plain: true });
      });
  };
</script>

<style scoped>
  .name-tag {
    margin-left: 8px;
  }

  .hint {
    margin-left: 8px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }

  .hint.block {
    display: block;
    margin: 6px 0 0;
  }
</style>
