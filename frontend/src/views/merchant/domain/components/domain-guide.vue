<!-- 域名解析指引 -->
<template>
  <ele-modal :width="560" title="域名解析指引" v-bind="modalProps">
    <el-alert
      v-if="data.domainType === 'own'"
      type="info"
      show-icon
      :closable="false"
      title="自有域名只确认归属，不要求把解析指到网关。"
      style="margin-bottom: 12px"
    />
    <el-alert
      v-else
      type="warning"
      show-icon
      :closable="false"
      title="请在域名服务商添加下面的记录。校验通过前，托管入口不会生效。"
      style="margin-bottom: 12px"
    />
    <el-descriptions :column="1" border class="detail-table">
      <el-descriptions-item label="域名">
        <span class="mono">{{ data.domain }}</span>
      </el-descriptions-item>
      <el-descriptions-item label="记录类型">{{
        data.verifyType
      }}</el-descriptions-item>
      <el-descriptions-item label="主机记录">
        <span class="mono">{{ data.verifyRecord }}</span>
        <el-button link type="primary" @click="copy(data.verifyRecord)"
          >复制</el-button
        >
      </el-descriptions-item>
      <el-descriptions-item label="记录值">
        <span class="mono">{{ data.verifyValue }}</span>
        <el-button link type="primary" @click="copy(data.verifyValue)"
          >复制</el-button
        >
      </el-descriptions-item>
      <el-descriptions-item label="状态">
        <el-tag
          :type="getOptionType(domainStatusOptions, data.status)"
          effect="light"
          size="small"
        >
          {{ getOptionLabel(domainStatusOptions, data.status) }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="校验结果">
        {{ data.verifyMessage || '尚未校验' }}
      </el-descriptions-item>
    </el-descriptions>
    <template #footer>
      <el-button type="primary" @click="closeModal()">知道了</el-button>
    </template>
  </ele-modal>
</template>

<script setup>
  import { EleMessage, useModal } from 'ele-admin-plus';
  import {
    domainStatusOptions,
    getOptionLabel,
    getOptionType
  } from '../../site/constants';

  defineProps({
    data: { type: Object, required: true }
  });

  const { modalProps, closeModal } = useModal();

  const copy = async (text) => {
    try {
      await navigator.clipboard.writeText(text || '');
      EleMessage.success({ message: '已复制', plain: true });
    } catch (e) {
      EleMessage.error({ message: '复制失败', plain: true });
    }
  };
</script>
