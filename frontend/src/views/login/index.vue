<template>
  <div class="login-wrapper">
    <div class="login-main">
      <ele-card shadow="always" class="login-card">
        <div class="login-cover">
          <div class="login-brand">
            <img class="login-logo" src="@/assets/logo.png" alt="H5Cloak" />
            <h1 class="login-title">{{ PROJECT_NAME }}</h1>
          </div>
        </div>
        <div class="login-body">
          <ele-text type="heading" style="font-size: 24px; margin-bottom: 18px">
            {{ t('login.title') }}
          </ele-text>
          <el-form
            ref="formRef"
            size="large"
            :model="form"
            :rules="rules"
            @keyup.enter="handleSubmit"
            @submit.prevent=""
          >
            <el-form-item prop="username">
              <el-input
                clearable
                v-model="form.username"
                autocomplete="off"
                :placeholder="t('login.username')"
                :prefix-icon="UserOutlined"
              />
            </el-form-item>
            <el-form-item prop="password">
              <el-input
                show-password
                v-model="form.password"
                autocomplete="off"
                :placeholder="t('login.password')"
                :prefix-icon="LockOutlined"
              />
            </el-form-item>
            <el-form-item v-if="captchaEnabled" prop="code">
              <div class="login-captcha-group">
                <el-input
                  clearable
                  v-model="form.code"
                  :placeholder="t('login.code')"
                  :prefix-icon="ProtectOutlined"
                />
                <div class="login-captcha" @click="queryCaptcha">
                  <img v-if="captcha" :src="captcha" :style="captchaStyle" />
                </div>
              </div>
            </el-form-item>
            <el-form-item>
              <el-checkbox v-model="form.rememberMe">
                {{ t('login.remember') }}
              </el-checkbox>
            </el-form-item>
            <el-form-item>
              <el-button
                size="large"
                type="primary"
                :loading="loading"
                style="width: 100%"
                @click="handleSubmit"
              >
                {{ t('login.login') }}
              </el-button>
            </el-form-item>
          </el-form>
        </div>
      </ele-card>
    </div>
    <page-footer style="padding-top: 0" />
  </div>
</template>

<script setup>
  import { ref, reactive, computed } from 'vue';
  import { EleMessage } from 'ele-admin-plus';
  import {
    UserOutlined,
    LockOutlined,
    ProtectOutlined
  } from '@/components/icons';
  import PageFooter from '@/layout/components/page-footer.vue';
  import { useLogin } from '@/utils/use-login';
  import { getCaptcha } from '@/api/login';
  import { useI18n } from 'vue-i18n';
  import { useRoute } from 'vue-router';
  import { APP_NAME as PROJECT_NAME, APP_PORTAL } from '@portal';

  const route = useRoute();
  const { login, loginByTicket, checkLogin } = useLogin();
  const { t } = useI18n();

  /** 表单组件 */
  const formRef = ref(null);

  /** 登录状态 */
  const loading = ref(false);

  /** 表单数据 */
  const form = reactive({
    username: '',
    password: '',
    code: '',
    uuid: '',
    rememberMe: true
  });

  /** 表单验证规则 */
  const rules = computed(() => {
    return {
      username: [
        {
          required: true,
          message: t('login.username'),
          type: 'string',
          trigger: 'blur'
        }
      ],
      password: [
        {
          required: true,
          message: t('login.password'),
          type: 'string',
          trigger: 'blur'
        }
      ],
      code: [
        {
          required: true,
          message: t('login.code'),
          type: 'string',
          trigger: 'blur'
        }
      ]
    };
  });

  /** 图形验证码 */
  const captcha = ref('');

  /** 图形验证码开关 */
  const captchaEnabled = ref(false);

  /** 图形验证码样式 */
  const captchaStyle = ref({});

  /** 获取图形验证码 */
  const queryCaptcha = () => {
    getCaptcha()
      .then((result) => {
        captchaEnabled.value = result.captchaEnabled ?? true;
        if (captchaEnabled.value) {
          captcha.value = `data:image/svg+xml;base64,${result.img}`;
          captchaStyle.value = {};
          form.uuid = result.uuid;
          formRef.value?.clearValidate?.();
        }
      })
      .catch((e) => {
        EleMessage.error({ message: e.message, plain: true });
      });
  };

  /** 提交 */
  const handleSubmit = () => {
    formRef.value?.validate?.((valid) => {
      if (!valid) {
        return;
      }
      loading.value = true;
      login(form).catch((e) => {
        loading.value = false;
        EleMessage.error({ message: e.message, plain: true });
        queryCaptcha();
      });
    });
  };

  /** 模拟登录票据优先于已有会话；否则已登录直接进入首页 */
  const ticket = [route.query.ticket].flat()[0];
  if (APP_PORTAL === 'merchant' && ticket) {
    loading.value = true;
    loginByTicket(String(ticket)).catch((e) => {
      loading.value = false;
      EleMessage.error({ message: e.message, plain: true });
      queryCaptcha();
    });
  } else {
    checkLogin().catch(() => {
      queryCaptcha();
    });
  }
</script>

<style lang="scss" scoped>
  .login-wrapper {
    min-height: 100vh;
    min-height: 100dvh;
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    background-image: url('@/assets/login-bg.png');
    background-repeat: no-repeat;
    background-size: 100% 100%;

    .login-main {
      flex: auto;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      box-sizing: border-box;
      padding: 20px;
    }

    .login-card {
      width: 920px;
      max-width: 100%;
      overflow: hidden;

      :deep(.ele-card-body) {
        display: flex;
        padding: 0;
        height: 462px;
      }
    }
  }

  .login-cover {
    flex: 1;
    padding: 32px 8px;
    box-sizing: border-box;
    background-color: #1681fd;
    background-image: url('@/assets/login-img.png');
    background-repeat: no-repeat;
    background-position: bottom;
    background-size: contain;
    text-align: center;
  }

  .login-brand {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 16px;
  }

  .login-logo {
    width: 80px;
    height: 80px;
    object-fit: contain;
    flex-shrink: 0;
  }

  .login-body {
    width: 400px;
    flex-shrink: 0;
    padding: 32px 48px 0 48px;
    box-sizing: border-box;

    :deep(.el-checkbox) {
      height: auto;

      .el-checkbox__label {
        color: inherit;
      }
    }

    :deep(.el-input__prefix-inner > .el-icon),
    :deep(.el-select__prefix > .el-icon) {
      margin-right: 12px;
      transform: scale(1.16);
    }
  }

  /* 验证码 */
  .login-captcha-group {
    width: 100%;
    display: flex;
    align-items: center;

    :deep(.el-input) {
      flex: 1;
    }

    .login-captcha {
      flex-shrink: 0;
      width: 120px;
      height: 40px;
      margin-left: 8px;
      border-radius: var(--el-border-radius-base);
      border: 1px solid var(--el-border-color);
      transition: border 0.2s;
      box-sizing: border-box;
      background: #fff;
      overflow: hidden;
      cursor: pointer;

      img {
        width: 100%;
        height: 100%;
        object-fit: contain;
        display: block;
      }

      &:hover {
        border-color: var(--el-color-primary);
      }
    }
  }

  /* 标题 */
  .login-title {
    color: rgba(255, 255, 255, 0.98);
    font-size: 28px;
    margin: 0;
    font-weight: normal;
    letter-spacing: 1.2px;
    font-family:
      -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue',
      Arial, 'Noto Sans', sans-serif, 'Apple Color Emoji', 'Segoe UI Emoji',
      'Segoe UI Symbol', 'Noto Color Emoji';
  }

  /* 小屏幕适应 */
  @media screen and (max-width: 680px) {
    .login-wrapper {
      background: #fff;

      .login-main {
        padding: 0;
        display: block;
      }

      .login-card {
        width: 100%;
        background: none;
        box-shadow: none;
        border-radius: 0;

        :deep(.ele-card-body) {
          display: block;
          height: auto;
        }
      }
    }

    .login-cover {
      padding: 20px 12px 100px 12px;
      background-size: auto 100px;
    }

    .login-body {
      width: 100%;
    }
  }
</style>
