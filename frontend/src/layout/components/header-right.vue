<template>
  <!-- 全屏切换 -->
  <layout-tool class="hidden-sm-and-down" @click="toggleFullscreen">
    <el-icon style="transform: scale(1.18)">
      <CompressOutlined v-if="isFullscreen" style="stroke-width: 4" />
      <ExpandOutlined v-else style="stroke-width: 4" />
    </el-icon>
  </layout-tool>
  <layout-tool v-if="isMerchant" @click="goDownload">
    <el-badge :value="activeExports" :hidden="!activeExports" :max="99">
      <el-icon style="transform: scale(1.05)">
        <Download />
      </el-icon>
    </el-badge>
  </layout-tool>
  <!-- 用户信息 -->
  <layout-tool>
    <header-user />
  </layout-tool>
  <!-- 夜间模式 -->
  <layout-tool ref="darkSwitchRef" class="ele-dark-switch hidden-sm-and-down">
    <el-switch
      :active-action-icon="MoonOutlined"
      :inactive-action-icon="SunOutlined"
      :model-value="darkMode"
      @update:modelValue="updateDarkMode"
    />
  </layout-tool>
  <!-- 主题设置 -->
  <layout-tool @click="openSetting">
    <el-icon>
      <MoreOutlined />
    </el-icon>
  </layout-tool>
</template>

<script setup>
  import { onMounted, onUnmounted, ref, watch } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import { storeToRefs } from 'pinia';
  import { Download } from '@element-plus/icons-vue';
  import { APP_PORTAL } from '@portal';
  import { getExportStats } from '@/api/merchant/ops';
  import {
    LayoutTool,
    requestFullscreen,
    exitFullscreen,
    useModal,
    EleMessage
  } from 'ele-admin-plus';
  import {
    ExpandOutlined,
    CompressOutlined,
    MoreOutlined,
    MoonOutlined,
    SunOutlined
  } from '@/components/icons';
  import { doWithTransition } from '@/utils/common';
  import { useThemeStore } from '@/store/modules/theme';
  import HeaderUser from './header-user.vue';

  const props = defineProps({
    /** 是否全屏状态 */
    isFullscreen: Boolean
  });

  const emit = defineEmits(['update:isFullscreen']);

  const { openModal } = useModal();

  const themeStore = useThemeStore();
  const { darkMode } = storeToRefs(themeStore);

  const { t } = useI18n();
  const route = useRoute();
  const router = useRouter();
  const isMerchant = APP_PORTAL === 'merchant';
  const activeExports = ref(0);
  let exportTimer;

  const loadExportBadge = () => {
    if (!isMerchant) return;
    getExportStats()
      .then((d) => {
        activeExports.value = Number(d?.active || 0);
      })
      .catch(() => {});
  };
  const restartExportPoll = () => {
    if (!isMerchant) return;
    if (exportTimer) clearInterval(exportTimer);
    const ms = route.path === '/merchant/download' ? 5000 : 30000;
    exportTimer = setInterval(loadExportBadge, ms);
  };
  const goDownload = () => router.push('/merchant/download');

  onMounted(() => {
    loadExportBadge();
    restartExportPoll();
  });
  watch(() => route.path, restartExportPoll);
  onUnmounted(() => {
    if (exportTimer) clearInterval(exportTimer);
  });

  /** 全屏切换 */
  const toggleFullscreen = () => {
    if (props.isFullscreen) {
      exitFullscreen();
      emit('update:isFullscreen', false);
      return;
    }
    try {
      requestFullscreen();
      emit('update:isFullscreen', true);
    } catch (e) {
      console.error(e);
      EleMessage.error({ message: e.message, plain: true });
    }
  };

  /** 打开主题设置抽屉 */
  const openSetting = () => {
    openModal({
      modalId: 'theme-setting-drawer',
      type: 'drawer',
      asyncComponent: () => import('./setting-drawer.vue'),
      props: {
        size: 268,
        title: t('layout.setting.title'),
        zIndex: 199999,
        bodyStyle: { padding: 0, height: '100%' },
        modalClass: 'ele-setting-drawer'
      },
      keepAlive: true
    });
  };

  /** 暗黑主题切换开关 */
  const darkSwitchRef = ref(null);

  /** 切换暗黑模式 */
  const updateDarkMode = (isDark) => {
    doWithTransition(
      () => themeStore.setValue('darkMode', isDark),
      darkSwitchRef.value?.$el?.querySelector?.('.el-switch__action'),
      !isDark
    );
  };
</script>
