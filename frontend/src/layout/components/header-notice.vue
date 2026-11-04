<!-- 顶栏消息通知：告警 / 导出 / 系统通知，读 notifications 表 -->
<template>
  <ele-popover
    :width="360"
    trigger="click"
    transition="el-zoom-in-top"
    :content-style="{ padding: 0 }"
    :body-style="{ overflow: 'hidden' }"
    :popper-style="{ maxWidth: 'calc(100% - 32px)' }"
    :popper-options="{
      strategy: 'fixed',
      modifiers: [{ name: 'offset', options: { offset: [0, 5] } }]
    }"
    @show="load"
  >
    <template #reference>
      <div style="display: flex; align-items: center; height: 100%">
        <el-badge
          :value="unread"
          :max="99"
          :hidden="!unread"
          style="line-height: 1; padding: 4px 0"
        >
          <el-icon style="transform: scale(1.17)">
            <BellOutlined />
          </el-icon>
        </el-badge>
      </div>
    </template>
    <div class="notice-head">
      <span>通知</span>
      <span v-if="unread" class="notice-unread">{{ unread }} 条未读</span>
    </div>
    <div class="list-wrapper">
      <div
        v-for="item in list"
        :key="item.id"
        class="list-item"
        :class="{ 'is-unread': !item.readAt }"
        @click="open(item)"
      >
        <div class="list-item-icon" :class="`is-${item.severity || 'info'}`">
          <el-icon>
            <WarningFilled v-if="item.kind === 'alert'" />
            <CircleCheckFilled v-else-if="item.kind === 'alert_resolved'" />
            <Download v-else-if="item.kind === 'export'" />
            <BellFilled v-else />
          </el-icon>
        </div>
        <div class="list-item-body">
          <ele-ellipsis>{{ item.title }}</ele-ellipsis>
          <ele-ellipsis type="placeholder" class="list-item-text">
            {{ item.content }}
          </ele-ellipsis>
          <ele-ellipsis type="placeholder" class="list-item-text">
            {{ fmtTime(item.createdAt) }}
          </ele-ellipsis>
        </div>
      </div>
      <el-empty v-if="!list.length" description="暂无通知" :image-size="68" />
    </div>
    <div class="bottom-tools">
      <div class="bottom-tool" @click="readAll">全部已读</div>
      <el-divider direction="vertical" style="margin: 0; width: 0" />
      <div class="bottom-tool" @click="more">查看更多</div>
    </div>
  </ele-popover>
</template>

<script setup>
  import { ref, onMounted, onBeforeUnmount } from 'vue';
  import { useRouter } from 'vue-router';
  import {
    WarningFilled,
    CircleCheckFilled,
    Download,
    BellFilled
  } from '@element-plus/icons-vue';
  import { BellOutlined } from '@/components/icons';
  import { APP_PORTAL } from '@portal';
  import {
    pageNotifications,
    getUnreadCount,
    markNotificationsRead
  } from '@/api/merchant/ops';

  const router = useRouter();
  const list = ref([]);
  const unread = ref(0);
  let timer = null;

  const refreshUnread = () =>
    getUnreadCount()
      .then((d) => (unread.value = d.unread || 0))
      .catch(() => {});
  const load = () =>
    pageNotifications({ pageNum: 1, pageSize: 10 })
      .then((r) => {
        list.value = r.rows || [];
        refreshUnread();
      })
      .catch(() => {});
  const open = (item) => {
    if (!item.readAt) {
      markNotificationsRead([item.id]).then(() => {
        item.readAt = new Date().toISOString();
        refreshUnread();
      });
    }
    if (item.link) {
      try {
        const u = new URL(item.link, window.location.origin);
        router.push(u.pathname + u.search);
      } catch {
        /* ignore */
      }
    }
  };
  const readAll = () =>
    markNotificationsRead([]).then(() => {
      list.value.forEach(
        (i) => (i.readAt = i.readAt || new Date().toISOString())
      );
      refreshUnread();
    });
  const more = () =>
    router.push(
      APP_PORTAL === 'merchant' ? '/merchant/alert' : '/admin/dashboard'
    );
  const fmtTime = (v) => (v ? String(v).replace('T', ' ').slice(0, 19) : '');
  const onChanged = () => {
    load();
  };

  onMounted(() => {
    refreshUnread();
    timer = setInterval(refreshUnread, 60000);
    window.addEventListener('cloak:notifications-changed', onChanged);
  });
  onBeforeUnmount(() => {
    clearInterval(timer);
    window.removeEventListener('cloak:notifications-changed', onChanged);
  });
</script>

<style lang="scss" scoped>
  .notice-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 18px;
    font-weight: 600;
    border-bottom: 1px solid hsla(0, 0%, 60%, 0.2);

    .notice-unread {
      font-weight: 400;
      font-size: 12px;
      color: var(--el-color-danger);
    }
  }

  .list-wrapper {
    padding: 4px 0;
    box-sizing: border-box;
    max-height: calc(100vh - 220px);
    min-height: 60px;
    overflow: auto;
  }

  .list-item {
    display: flex;
    padding: 12px 18px;
    box-sizing: border-box;
    transition: background-color 0.2s;
    cursor: pointer;

    &.is-unread {
      background-color: var(--el-color-primary-light-9);
    }

    .list-item-body {
      flex: 1;
      overflow: hidden;
      padding-left: 12px;

      .list-item-text {
        margin-top: 4px;
      }
    }

    .list-item-icon {
      width: 32px;
      height: 32px;
      color: #fff;
      font-size: 16px;
      border-radius: 50%;
      flex-shrink: 0;
      display: flex;
      align-items: center;
      justify-content: center;
      background-color: var(--el-color-info);

      &.is-warning {
        background-color: var(--el-color-warning);
      }

      &.is-critical {
        background-color: var(--el-color-danger);
      }

      &.is-info {
        background-color: var(--el-color-primary);
      }
    }

    &:hover {
      background-color: hsla(0, 0%, 60%, 0.08);
    }

    & + .list-item {
      border-top: 1px solid hsla(0, 0%, 60%, 0.12);
    }
  }

  .bottom-tools {
    display: flex;
    align-items: center;
    border-top: 1px solid hsla(0, 0%, 60%, 0.2);

    .bottom-tool {
      flex: 1;
      line-height: 44px;
      text-align: center;
      transition: background-color 0.3s;
      cursor: pointer;

      &:hover {
        background: hsla(0, 0%, 60%, 0.08);
      }
    }
  }
</style>
