<template>
  <ele-page>
    <section-card class="page-section" title="接收邮箱" subtitle="每行一个，最多 10 个">
      <el-form label-width="120px">
        <el-form-item label="邮箱列表">
          <el-input
            v-model="form.emailsText"
            type="textarea"
            :rows="4"
            maxlength="1000"
            show-word-limit
            :disabled="!canEdit"
            placeholder="每行一个邮箱，例如 ops@example.com"
          />
        </el-form-item>
        <el-form-item label="平台 SMTP">
          <el-tag size="small" :type="form.emailConfigured ? 'success' : 'info'" effect="plain">
            {{ form.emailConfigured ? '已配置' : '未配置，邮件渠道会跳过' }}
          </el-tag>
          <el-button
            link
            type="primary"
            :disabled="!form.emailConfigured"
            @click="test('email')"
          >发送测试</el-button>
        </el-form-item>
      </el-form>
    </section-card>

    <section-card class="page-section" title="Telegram" subtitle="推荐用自己的机器人：告警、日报、还能用中文问数据">
      <template #extra>
        <el-button link type="primary" @click="router.push('/merchant/alert/telegram-setup')">配置我的机器人</el-button>
        <el-button link type="primary" @click="router.push('/merchant/alert/telegram')">查看教程</el-button>
      </template>
      <el-alert
        type="info"
        :closable="false"
        show-icon
        style="margin-bottom: 12px"
        title="已支持每个商户用自己的 Telegram 机器人。点右上「配置我的机器人」填 Token、一键绑定，并开启每日日报与中文问答。下面的 Chat ID 是未配置自有机器人时的平台兜底方式。"
      />
      <el-form label-width="120px">
        <el-form-item label="Chat ID">
          <el-input
            v-model.trim="form.telegramChatId"
            clearable
            :disabled="!canEdit"
            placeholder="请输入 Chat ID"
          />
        </el-form-item>
        <el-form-item label="平台 Bot">
          <el-tag size="small" :type="form.telegramConfigured ? 'success' : 'info'" effect="plain">
            {{ form.telegramConfigured ? '已配置' : '未配置，Telegram 渠道会跳过' }}
          </el-tag>
          <el-button
            link
            type="primary"
            :disabled="!form.telegramConfigured"
            @click="test('telegram')"
          >发送测试</el-button>
        </el-form-item>
      </el-form>
    </section-card>

    <section-card class="page-section" title="启用的通知渠道" subtitle="这里关闭的渠道对所有规则生效">
      <el-form label-width="120px">
        <el-form-item label="渠道">
          <div class="channel-field">
            <el-checkbox-group v-model="form.channels" :disabled="!canEdit">
              <el-checkbox v-for="item in channelOptions" :key="item.value" :value="item.value" :label="item.label" />
            </el-checkbox-group>
            <div class="hint">Webhook 的接收地址在「Webhook」页配置。</div>
          </div>
        </el-form-item>
      </el-form>
    </section-card>

    <section-card class="page-section" title="最近站内通知" subtitle="与右上角铃铛同步">
      <template #extra>
        <el-button link type="primary" @click="markAllRead">全部已读</el-button>
      </template>
      <div v-if="notifications.length" class="notif-list">
        <div v-for="item in notifications" :key="item.id" class="notif-item" :class="{ 'is-unread': !item.readAt }">
          <el-tag size="small" :type="severityType(item.severity)" effect="light">{{ kindLabel(item.kind) }}</el-tag>
          <div class="notif-body">
            <div>{{ item.title }}</div>
            <div class="hint">{{ item.content }}</div>
          </div>
          <span class="hint time">{{ fmtTime(item.createdAt) }}</span>
        </div>
      </div>
      <el-empty v-else :image-size="70" description="暂无通知" />
    </section-card>

    <ele-card class="table-section">
      <div class="form-footer">
        <el-button @click="test('inapp')">发一条站内测试通知</el-button>
        <el-button v-if="canEdit" @click="load">取消</el-button>
        <el-button v-permission="'alert:edit'" type="primary" :loading="saving" @click="save">保存</el-button>
      </div>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { onMounted, ref } from 'vue';
  import { useRouter } from 'vue-router';
  import { EleMessage } from 'ele-admin-plus';
  import { usePermission } from '@/utils/use-permission';
  import {
    getNotifySettings,
    markNotificationsRead,
    pageNotifications,
    saveNotifySettings,
    testNotifyChannel
  } from '@/api/merchant/ops';
  import { channelOptions, kindLabel, severityType } from '../constants';

  defineOptions({ name: 'MerchantAlertSettings' });

  const router = useRouter();
  const { hasPermission } = usePermission();
  const canEdit = hasPermission('alert:edit');
  const saving = ref(false);
  const notifications = ref([]);
  const form = ref({
    emailsText: '',
    telegramChatId: '',
    channels: [],
    emailConfigured: false,
    telegramConfigured: false
  });

  const apply = (data) => {
    form.value = {
      ...data,
      emailsText: (data?.emails || []).join('\n'),
      channels: [...(data?.channels || [])],
      telegramChatId: data?.telegramChatId || ''
    };
  };

  const load = () =>
    getNotifySettings()
      .then(apply)
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));

  const loadNotifications = () =>
    pageNotifications({ pageNum: 1, pageSize: 10 })
      .then((data) => (notifications.value = data?.rows || []))
      .catch(() => {});

  const save = () => {
    const emails = form.value.emailsText
      .split(/\r?\n|,/)
      .map((item) => item.trim())
      .filter(Boolean);
    if (emails.length > 10) {
      EleMessage.error({ message: '接收邮箱最多 10 个', plain: true });
      return;
    }
    saving.value = true;
    saveNotifySettings({
      emails,
      telegramChatId: form.value.telegramChatId,
      channels: form.value.channels
    })
      .then(() => {
        EleMessage.success({ message: '已保存', plain: true });
        return load();
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => {
        saving.value = false;
      });
  };

  const test = (channel) =>
    testNotifyChannel(channel)
      .then(() => {
        EleMessage.success({ message: '已发送', plain: true });
        loadNotifications();
        window.dispatchEvent(new CustomEvent('cloak:notifications-changed'));
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));

  const markAllRead = () =>
    markNotificationsRead([])
      .then(() => {
        loadNotifications();
        window.dispatchEvent(new CustomEvent('cloak:notifications-changed'));
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));

  const fmtTime = (value) => (value ? String(value).replace('T', ' ').slice(0, 19) : '');

  onMounted(() => {
    load();
    loadNotifications();
  });
</script>

<style scoped>
  .hint {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }

  .channel-field {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 6px;
  }

  .form-footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 12px;
  }

  .notif-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .notif-item {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 10px 12px;
    border-radius: 8px;
    font-size: 13px;
  }

  .notif-item.is-unread {
    background: var(--el-fill-color-light);
  }

  .notif-body {
    flex: 1;
    min-width: 0;
  }

  .time {
    flex: none;
  }
</style>
