<template>
  <ele-page>
    <section-card
      class="page-section"
      title="机器人"
      subtitle="填入你自己用 BotFather 创建的 Bot Token"
    >
      <template #extra>
        <el-button link type="primary" @click="router.push('/merchant/alert/telegram')">
          查看教程
        </el-button>
      </template>
      <el-form label-width="120px">
        <el-form-item label="机器人 Token">
          <div class="field-col">
            <el-input
              v-model.trim="token"
              clearable
              :disabled="!canEdit"
              placeholder="形如 123456789:AA..."
              style="max-width: 460px"
            />
            <div v-if="cfg.hasBot" class="hint">
              当前：<b>@{{ cfg.botUsername }}</b>
              <el-tag
                size="small"
                :type="cfg.status === 'active' ? 'success' : 'info'"
                effect="plain"
                style="margin-left: 6px"
              >{{ cfg.status === 'active' ? '已连接' : '已停用' }}</el-tag>
              <span v-if="cfg.tokenSuffix"> · Token 尾号 {{ cfg.tokenSuffix }}</span>
            </div>
          </div>
        </el-form-item>
        <el-form-item>
          <el-button
            v-permission="'alert:edit'"
            type="primary"
            :loading="savingBot"
            @click="saveBot"
          >{{ cfg.hasBot ? '更换 Token' : '保存并连接' }}</el-button>
          <el-button
            v-if="cfg.hasBot"
            v-permission="'alert:edit'"
            @click="removeBot"
          >移除机器人</el-button>
        </el-form-item>
      </el-form>
    </section-card>

    <section-card
      class="page-section"
      title="接收会话"
      subtitle="绑定后的私聊或群才会收到告警和日报"
    >
      <el-form label-width="120px">
        <el-form-item label="添加绑定">
          <div class="field-col">
            <el-button
              v-permission="'alert:edit'"
              type="primary"
              plain
              :disabled="!cfg.hasBot"
              :loading="binding"
              @click="genBind"
            >生成绑定方式</el-button>
            <div v-if="bind.code" class="bind-box">
              <div class="bind-line">
                绑给自己：<el-link type="primary" :href="bind.deepLink" target="_blank">在 Telegram 中打开并绑定</el-link>
              </div>
              <div class="bind-line">
                绑到群：把机器人拉进群后发送
                <code>{{ bind.groupCommand }}</code>
                <el-button link type="primary" @click="copy(bind.groupCommand)">复制</el-button>
              </div>
              <div class="hint">绑定码 {{ Math.round(bind.expiresInSec / 60) }} 分钟内有效，用一次即失效。最多 10 个会话。绑定成功后下方列表会自动出现。</div>
            </div>
          </div>
        </el-form-item>
      </el-form>
      <el-table :data="cfg.chats" border size="small" style="width: 100%; max-width: 680px">
        <el-table-column label="会话" min-width="160">
          <template #default="{ row }">
            {{ row.title || row.chatId }}
          </template>
        </el-table-column>
        <el-table-column label="类型" width="110">
          <template #default="{ row }">{{ chatTypeLabel(row.chatType) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 'active' ? 'success' : 'info'" effect="plain">
              {{ row.status === 'active' ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <el-button link type="primary" :disabled="!canEdit" @click="toggleChat(row)">
              {{ row.status === 'active' ? '停用' : '启用' }}
            </el-button>
            <el-button link type="danger" :disabled="!canEdit" @click="unbindChat(row)">解绑</el-button>
          </template>
        </el-table-column>
        <template #empty>还没有绑定会话</template>
      </el-table>
    </section-card>

    <section-card class="page-section" title="每日日报" subtitle="每天定时把昨日概况发到绑定会话">
      <el-form label-width="120px">
        <el-form-item label="开启日报">
          <el-switch v-model="daily.enabled" :disabled="!canEdit || !cfg.hasBot" />
        </el-form-item>
        <el-form-item label="发送时间">
          <el-time-select
            v-model="daily.timeText"
            :disabled="!canEdit || !daily.enabled"
            start="00:00"
            step="01:00"
            end="23:00"
            placeholder="每天几点"
            style="width: 160px"
          />
          <span class="hint" style="margin-left: 8px">按商户时区整点发送</span>
        </el-form-item>
        <el-form-item>
          <el-button v-permission="'alert:edit'" type="primary" :loading="savingDaily" @click="saveDaily">
            保存日报设置
          </el-button>
          <el-button :disabled="!cfg.hasBot" @click="test">发送测试</el-button>
        </el-form-item>
      </el-form>
    </section-card>

    <section-card class="page-section" title="怎么用" subtitle="在绑定会话里直接用中文问">
      <ul class="usage">
        <li><code>今天多少访问</code>、<code>昨天转化多少</code>、<code>近 7 天怎么样</code></li>
        <li><code>有没有告警</code>、<code>日报</code>、<code>帮助</code></li>
        <li>多站点可带站点名，如「今天 站点A 多少访问」。机器人只读统计、只回你自己的数据。</li>
      </ul>
    </section-card>
  </ele-page>
</template>

<script setup>
  import { onMounted, onBeforeUnmount, ref } from 'vue';
  import { useRouter } from 'vue-router';
  import { EleMessage } from 'ele-admin-plus';
  import { usePermission } from '@/utils/use-permission';
  import {
    getTelegramConfig,
    saveTelegramBot,
    deleteTelegramBot,
    newTelegramBindCode,
    toggleTelegramChat,
    deleteTelegramChat,
    saveTelegramDaily,
    testTelegram
  } from '@/api/merchant/ops';

  defineOptions({ name: 'MerchantAlertTelegramSetup' });

  const router = useRouter();
  const { hasPermission } = usePermission();
  const canEdit = hasPermission('alert:edit');

  const token = ref('');
  const cfg = ref({ hasBot: false, botUsername: '', status: '', tokenSuffix: '', chats: [] });
  const bind = ref({ code: '', deepLink: '', groupCommand: '', expiresInSec: 600 });
  const daily = ref({ enabled: false, timeText: '09:00' });
  const savingBot = ref(false);
  const savingDaily = ref(false);
  const binding = ref(false);
  let pollTimer = null;

  const applyCfg = (d) => {
    cfg.value = {
      hasBot: !!d?.hasBot,
      botUsername: d?.botUsername || '',
      status: d?.status || '',
      tokenSuffix: d?.tokenSuffix || '',
      chats: d?.chats || []
    };
    daily.value = {
      enabled: !!d?.dailyReport,
      timeText: String(d?.dailyHour ?? 9).padStart(2, '0') + ':00'
    };
  };

  const load = () =>
    getTelegramConfig()
      .then(applyCfg)
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));

  const saveBot = () => {
    if (!token.value) {
      EleMessage.error({ message: '请先粘贴 Bot Token', plain: true });
      return;
    }
    savingBot.value = true;
    saveTelegramBot(token.value)
      .then((d) => {
        EleMessage.success({ message: '已连接 @' + (d?.botUsername || ''), plain: true });
        token.value = '';
        return load();
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => (savingBot.value = false));
  };

  const removeBot = () => {
    deleteTelegramBot()
      .then(() => {
        EleMessage.success({ message: '已移除', plain: true });
        return load();
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));
  };

  const genBind = () => {
    binding.value = true;
    newTelegramBindCode()
      .then((d) => {
        bind.value = d || bind.value;
        startPoll();
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => (binding.value = false));
  };

  const startPoll = () => {
    stopPoll();
    let n = 0;
    const before = cfg.value.chats.length;
    pollTimer = setInterval(() => {
      n += 1;
      getTelegramConfig()
        .then((d) => {
          applyCfg(d);
          if ((d?.chats || []).length > before) {
            EleMessage.success({ message: '绑定成功', plain: true });
            bind.value = { code: '', deepLink: '', groupCommand: '', expiresInSec: 600 };
            stopPoll();
          }
        })
        .catch(() => {});
      if (n >= 60) stopPoll();
    }, 3000);
  };

  const stopPoll = () => {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
  };

  const toggleChat = (row) =>
    toggleTelegramChat(row.id)
      .then(load)
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));

  const unbindChat = (row) =>
    deleteTelegramChat(row.id)
      .then(() => {
        EleMessage.success({ message: '已解绑', plain: true });
        return load();
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));

  const saveDaily = () => {
    const hour = parseInt(String(daily.value.timeText).slice(0, 2), 10) || 0;
    savingDaily.value = true;
    saveTelegramDaily({ enabled: daily.value.enabled, hour, siteId: 0 })
      .then(() => {
        EleMessage.success({ message: '已保存', plain: true });
        return load();
      })
      .catch((e) => EleMessage.error({ message: e.message, plain: true }))
      .finally(() => (savingDaily.value = false));
  };

  const test = () =>
    testTelegram()
      .then(() => EleMessage.success({ message: '已发送，去 Telegram 查看', plain: true }))
      .catch((e) => EleMessage.error({ message: e.message, plain: true }));

  const copy = (text) => {
    navigator.clipboard?.writeText(text);
    EleMessage.success({ message: '已复制', plain: true });
  };

  const chatTypeLabel = (t) =>
    ({ private: '私聊', group: '群', supergroup: '群', channel: '频道' }[t] || t || '');

  onMounted(load);
  onBeforeUnmount(stopPoll);
</script>

<style scoped>
  .field-col {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .hint {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.6;
  }

  .bind-box {
    padding: 10px 14px;
    background: var(--el-fill-color-lighter);
    border-radius: 8px;
    border-left: 3px solid var(--el-color-primary);
  }

  .bind-line {
    line-height: 1.9;
  }

  .usage {
    margin: 0;
    padding-left: 20px;
    line-height: 2;
    color: var(--el-text-color-regular);
  }

  code {
    padding: 1px 6px;
    font-size: 13px;
    background: var(--el-fill-color);
    border-radius: 4px;
  }
</style>
