<template>
  <ele-page>
    <ele-card :body-style="{ padding: '24px 28px' }">
      <div class="guide-header">
        <el-button text type="primary" @click="router.push('/merchant/alert/settings')">
          <el-icon><ArrowLeft /></el-icon>
          返回通知设置
        </el-button>
        <h2 class="guide-title">配置 Telegram 机器人</h2>
        <p class="guide-desc">
          用你自己的 Telegram 机器人收告警、每日日报，还能直接用中文问数据。全程不用抄 Chat ID：填一次
          Token，点一下绑定即可。
        </p>
      </div>

      <div class="guide-section">
        <h3>1. 创建你自己的机器人</h3>
        <p>每个商户用自己的机器人，告警和日报都从你的机器人账号发出。</p>
        <ol class="guide-steps">
          <li>打开 Telegram，搜索 <code>@BotFather</code>，进入这个官方机器人。</li>
          <li>发送 <code>/newbot</code>。</li>
          <li>按提示填机器人的显示名称和用户名。用户名必须以 <code>bot</code> 结尾，例如 <code>yourshop_bot</code>。</li>
          <li>BotFather 返回一串 Token，形如 <code>123456789:AA...</code>。这串就是下一步要填的 Token，妥善保管，不要发到群里。</li>
        </ol>
      </div>

      <div class="guide-section">
        <h3>2. 填入 Token</h3>
        <p>回到「告警通知 → Telegram 配置」，把 Token 粘进「机器人 Token」输入框，点保存。</p>
        <ol class="guide-steps">
          <li>系统会连一次 Telegram 校验 Token。成功后显示你的机器人用户名（如 <code>@yourshop_bot</code>）和「已连接」。</li>
          <li>Token 加密保存，页面只回显后 4 位。以后可「更换 Token」或「移除机器人」。</li>
          <li>填错会提示「Token 无效或无法连接」，不会保存。重新从 BotFather 复制即可。</li>
        </ol>
        <el-alert
          type="info"
          :closable="false"
          show-icon
          class="guide-alert"
          title="不用再登录服务器、不用再手抄 Chat ID。Token 在后台填，绑定在下一步一键完成。"
        />
      </div>

      <div class="guide-section">
        <h3>3. 绑定接收会话（不用抄 Chat ID）</h3>
        <p>绑定后的会话才会收到告警和日报。可以绑给自己，也可以绑到一个群。</p>
        <h4>绑给自己（私聊）</h4>
        <ol class="guide-steps">
          <li>在配置页点「在 Telegram 中打开并绑定」。它会打开你机器人的对话。</li>
          <li>在对话里点「开始 / Start」。机器人提示绑定成功，配置页数秒内显示「已绑定」。</li>
        </ol>
        <h4>绑到群</h4>
        <ol class="guide-steps">
          <li>把你的机器人拉进目标群，允许它发言。</li>
          <li>复制配置页给出的一次性指令 <code>/bind 绑定码</code>，在群里发送。机器人回复绑定成功，该群就会收到告警和日报。</li>
        </ol>
        <el-alert
          type="info"
          :closable="false"
          show-icon
          class="guide-alert"
          title="绑定码 10 分钟有效、用一次就失效。最多绑定 10 个会话（自己 + 群），每个可单独停用或解绑。"
        />
      </div>

      <div class="guide-section">
        <h3>4. 开启渠道、设置日报</h3>
        <ol class="guide-steps">
          <li>在「启用的通知渠道」里勾选 Telegram，再保存，告警才会走 Telegram。</li>
          <li>打开「每日日报」，设一个时间（按你的时区，默认 09:00）。次日这个点会收到昨日概况。</li>
          <li>点「发送测试」，已绑定的会话应收到一条测试消息。收不到时，先确认机器人已连接、会话已绑定。</li>
        </ol>
      </div>

      <div class="guide-section">
        <h3>5. 直接问它数据</h3>
        <p>在绑定过的会话里，用中文直接问，机器人读你自己的数据后回复。例如：</p>
        <ul class="guide-list">
          <li><code>今天多少访问</code> —— 今日访问、真实、审核、首访/回访</li>
          <li><code>昨天转化多少</code> —— 昨日转化数与金额</li>
          <li><code>近 7 天怎么样</code> —— 7 日访问与转化概况</li>
          <li><code>有没有告警</code> —— 未恢复的告警列表</li>
          <li><code>日报</code> —— 立即补发一次昨日日报</li>
          <li><code>帮助</code> 或 <code>/help</code> —— 查看支持的问法</li>
        </ul>
        <p class="guide-note">
          多站点时可带站点名，例如「今天 站点A 多少访问」；不写则汇总全部站点。机器人只读统计、只回你自己的数据，不会改任何配置。
        </p>
      </div>

      <div class="guide-section">
        <h3>常见问题</h3>
        <ul class="guide-list">
          <li><strong>填了 Token 却显示未连接？</strong> Token 要从 BotFather 原样复制，不要带空格；服务器需能访问 Telegram。</li>
          <li><strong>点了绑定没反应？</strong> 要在机器人对话里点一次「开始」。绑定码过期就回配置页重新生成。</li>
          <li><strong>群里收不到？</strong> 确认机器人已入群、允许发言，并且在群里发过一次 <code>/bind 绑定码</code>。</li>
          <li><strong>还想用老的平台机器人？</strong> 没配自己的机器人时，仍可用平台 Bot + 手填 Chat ID 的老方式；配了自己的机器人后自动优先用你的。</li>
        </ul>
      </div>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { useRouter } from 'vue-router';
  import { ArrowLeft } from '@element-plus/icons-vue';

  defineOptions({ name: 'MerchantAlertTelegram' });

  const router = useRouter();
</script>

<style scoped>
  .guide-header {
    margin-bottom: 8px;
  }

  .guide-title {
    margin: 12px 0 8px;
    font-size: 22px;
    font-weight: 600;
  }

  .guide-desc {
    margin: 0 0 20px;
    color: var(--el-text-color-secondary);
    line-height: 1.7;
  }

  .guide-section {
    margin-top: 8px;
  }

  .guide-section h3 {
    margin: 22px 0 8px;
    font-size: 16px;
  }

  .guide-section h4 {
    margin: 14px 0 8px;
    font-size: 14px;
  }

  .guide-section p {
    margin: 0 0 8px;
    color: var(--el-text-color-regular);
    line-height: 1.7;
  }

  .guide-steps {
    margin: 0;
    padding-left: 20px;
    line-height: 1.9;
    color: var(--el-text-color-regular);
  }

  .guide-alert {
    margin-top: 16px;
  }

  .guide-list {
    margin: 8px 0 0;
    padding-left: 20px;
    line-height: 1.9;
    color: var(--el-text-color-regular);
  }

  .guide-note {
    margin: 12px 0 0;
    padding: 10px 14px;
    font-size: 13px;
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-lighter);
    border-radius: 6px;
    border-left: 3px solid var(--el-color-primary);
  }

  code {
    padding: 1px 6px;
    font-size: 13px;
    background: var(--el-fill-color);
    border-radius: 4px;
  }
</style>
