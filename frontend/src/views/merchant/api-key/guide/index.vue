<template>
  <ele-page>
    <ele-card :body-style="{ padding: '24px 28px' }">
      <div class="guide-header">
        <el-button text type="primary" @click="goBack">
          <el-icon><ArrowLeft /></el-icon>
          返回列表
        </el-button>
        <h2 class="guide-title">API 斗篷使用教程</h2>
        <p class="guide-desc">
          通过 API Key
          调用判定接口，由系统根据规则返回访客应访问审核页还是真实页。
        </p>
      </div>

      <el-alert
        title="斗篷仅适用于海外市场，中国及港澳台不适用"
        type="warning"
        :closable="false"
        show-icon
        class="guide-alert"
      />

      <div class="guide-section">
        <h3>1. 创建 API Key</h3>
        <p
          >在 API Key
          列表点击「添加」，填写名称与规则后确认。创建成功后会弹出完整
          Token，请立即复制保存，后续仅展示前缀。</p
        >
      </div>

      <div class="guide-section">
        <h3>2. 调用判定接口</h3>
        <p
          >第三方站点在服务端发起请求，在请求头携带 Api Token（也支持 Bearer
          Token，前缀为 <code>ck_</code>）。</p
        >
        <div class="code-block">
          <div class="code-label">请求地址</div>
          <pre>POST {{ apiBaseUrl }}/open/decision</pre>
        </div>
        <div class="code-block">
          <div class="code-label">请求头</div>
          <pre>X-API-Key: ck_xxxxxxxxxxxxxxxx</pre>
        </div>
        <div class="code-block">
          <div class="code-label">请求体（JSON）</div>
          <pre>
{
  "ip": "访客 IP",
  "userAgent": "访客 User-Agent",
  "referer": "来源地址",
  "acceptLanguage": "en-US,en;q=0.9"
}</pre
          >
        </div>
        <div class="code-block">
          <div class="code-label">cURL 示例</div>
          <pre>
curl -X POST "{{ apiBaseUrl }}/open/decision" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: ck_你的Token" \
  -d '{"ip":"1.2.3.4","userAgent":"Mozilla/5.0","referer":"https://example.com"}'</pre
          >
        </div>
      </div>

      <div class="guide-section">
        <h3>3. 解析返回结果</h3>
        <p>接口返回 <code>page</code> 字段，用于决定展示哪套页面：</p>
        <el-table
          :data="resultRows"
          border
          style="width: 100%; max-width: 640px"
        >
          <el-table-column prop="page" label="page" width="120" />
          <el-table-column prop="meaning" label="含义" />
        </el-table>
        <div class="code-block" style="margin-top: 16px">
          <div class="code-label">响应示例</div>
          <pre>
{
  "code": 200,
  "msg": "判定成功",
  "data": {
    "action": "white",
    "page": "review",
    "hitRule": "rule:cloak:review",
    "country": "US",
    "latencyMs": 12
  }
}</pre
          >
        </div>
        <ul class="guide-list">
          <li><code>page=review</code>：展示审核页（白页）</li>
          <li><code>page=real</code>：展示真实页（黑页）</li>
          <li>黑名单 IP 在所有模式下均返回审核页</li>
        </ul>
      </div>

      <div class="guide-section">
        <h3>4. 常见问题</h3>
        <ul class="guide-list">
          <li>Token 重置或删除后，旧 Token 立即失效。</li>
          <li>Key 停用后接口返回 401，请先在列表中启用。</li>
          <li>可在「API 调用日志」中查看每次调用的结果与耗时。</li>
        </ul>
      </div>

      <div class="guide-section guide-section--wordpress">
        <h3>5. WordPress 独立站接入</h3>
        <p>
          WordPress 站点必须在<strong>服务端（PHP）</strong>调用判定接口，Api
          Token 只能写在服务器配置中，切勿写入前端 JS 或页面源码。
        </p>

        <h4>5.1 前置准备</h4>
        <ul class="guide-list">
          <li
            >在斗篷后台创建 API Key，保存完整 Token（形如
            <code>ck_xxx</code>）。</li
          >
          <li
            >准备<strong>审核页（白页）</strong>与<strong>真实页（黑页）</strong>，审核页支持以下三种形式（见
            5.3～5.5）：</li
          >
          <li class="guide-sub-list">
            <ul>
              <li
                ><strong>方案一：</strong>WordPress 站内页面（后台新建页面）</li
              >
              <li
                ><strong>方案二：</strong>外部链接（跳转到其他域名或落地页）</li
              >
              <li
                ><strong>方案三：</strong>服务器 HTML 文件（静态文件或 PHP
                直接输出）</li
              >
            </ul>
          </li>
          <li
            >确认 WordPress 服务器能访问斗篷 API 地址（如
            <code>{{ apiBaseUrl }}/open/decision</code>）。</li
          >
          <li
            >若站点套了 Cloudflare 等 CDN，需正确传递访客真实
            IP（见下方代码）。</li
          >
        </ul>

        <h4>5.2 接入流程</h4>
        <ol class="guide-steps">
          <li>访客访问 WordPress 任意前台页面。</li>
          <li
            >在 <code>template_redirect</code> 钩子中，PHP 向斗篷 API 发起 POST
            请求。</li
          >
          <li>请求体传入访客 IP、User-Agent、Referer 等信息。</li>
          <li
            >根据返回的 <code>page</code> 字段决定展示内容：
            <code>review</code> → 审核页，<code>real</code> → 真实页。</li
          >
          <li>按所选方案跳转外部链接、输出 HTML 文件或进入站内页面。</li>
        </ol>

        <p class="guide-note"
          >以下三种方案共用同一段「判定接口」代码，仅审核页展示方式不同，请按需选择其一。</p
        >
        <div class="code-block">
          <div class="code-label">共用代码：判定接口（三个方案均需复制）</div>
          <pre>{{ cloakSharedCode }}</pre>
        </div>

        <h4>5.3 方案一：审核页为 WordPress 站内页面</h4>
        <p>
          在 WordPress 后台分别新建「审核页」和「真实页」，记下页面 ID。 判定为
          <code>review</code> 时跳转到审核页，为
          <code>real</code> 时跳转到真实页。
        </p>
        <ul class="guide-list">
          <li>适合：两套内容都在 WordPress 内维护。</li>
          <li>审核页、真实页均使用 <code>wp_safe_redirect</code> 站内跳转。</li>
        </ul>
        <div class="code-block">
          <div class="code-label">方案一：双站内页面跳转</div>
          <pre>{{ wordpressWpPageCode }}</pre>
        </div>

        <h4>5.4 方案二：审核页为外部链接</h4>
        <p>
          审核页不在本 WordPress 站点内，而是跳转到<strong>外部 URL</strong
          >（如独立落地页、Shopify 页面、其他服务器页面）。 真实页仍可使用
          WordPress 站内页面。
        </p>
        <ul class="guide-list">
          <li>适合：白页托管在第三方平台，或单独域名上的合规页面。</li>
          <li
            >外部链接须使用 <code>wp_redirect</code>（<code
              >wp_safe_redirect</code
            >
            仅限同域）。</li
          >
          <li>建议将允许的域名写入白名单，防止配置错误被利用。</li>
        </ul>
        <div class="code-block">
          <div class="code-label">方案二：审核页外部链接 + 真实页站内页面</div>
          <pre>{{ wordpressExternalCode }}</pre>
        </div>

        <h4>5.5 方案三：审核页为服务器 HTML 文件</h4>
        <p>
          将审核页做成独立 <code>.html</code> 文件放在服务器上，有两种常见用法：
        </p>
        <ul class="guide-list">
          <li
            ><strong>5.5.1 静态文件跳转：</strong>把 HTML 放到网站可访问目录（如
            <code>/wp-content/cloak/review.html</code>），判定后 302
            跳转到该文件 URL。</li
          >
          <li
            ><strong>5.5.2 PHP 直接输出：</strong>不跳转 URL，由 PHP 读取 HTML
            文件内容并输出，访客地址栏仍为 WordPress 域名。</li
          >
        </ul>

        <h5>5.5.1 静态 HTML 文件 + URL 跳转</h5>
        <p
          >先将 <code>review.html</code> 上传至
          <code>wp-content/cloak/review.html</code>，浏览器可访问
          <code>https://你的域名/wp-content/cloak/review.html</code>。</p
        >
        <div class="code-block">
          <div class="code-label">方案三-A：跳转到服务器 HTML 文件</div>
          <pre>{{ wordpressHtmlRedirectCode }}</pre>
        </div>

        <h5>5.5.2 PHP 读取 HTML 并直接输出</h5>
        <p
          >HTML 文件可放在 Web 目录外（更安全），仅 PHP
          有读权限。判定为审核页时直接输出文件内容并终止请求。</p
        >
        <div class="code-block">
          <div class="code-label">方案三-B：PHP 输出 HTML 文件（URL 不变）</div>
          <pre>{{ wordpressHtmlOutputCode }}</pre>
        </div>

        <h4>5.6 同域双模板（可选）</h4>
        <p>
          若真实页也不想跳转 URL，可在主题中准备
          <code>page-real.php</code> 模板； 审核页若用 HTML 文件，可在模板中
          <code>include</code> 该文件。
          注意：务必关闭全页缓存，或按访客维度区分缓存，否则所有用户可能看到同一套页面。
        </p>

        <h4>5.7 注意事项</h4>
        <ul class="guide-list">
          <li
            ><strong>外部链接：</strong>仅允许
            <code>https</code>，并在代码中配置域名白名单。</li
          >
          <li
            ><strong>HTML 文件：</strong>确保文件路径在服务器上可读；若放 Web
            目录外更安全，配合方案三-B 使用。</li
          >
          <li
            ><strong>Token 安全：</strong>仅存放在
            <code>wp-config.php</code> 或服务器环境变量，不要提交到 Git。</li
          >
          <li
            ><strong>页面缓存：</strong>WP Rocket、LiteSpeed、Nginx
            全页缓存会导致斗篷失效，需排除前台或使用「按 Cookie / 按
            IP」缓存策略。</li
          >
          <li
            ><strong>管理后台：</strong>代码中已跳过
            <code>wp-admin</code>，后台登录不受影响。</li
          >
          <li
            ><strong>爬虫预览：</strong>Facebook / Google
            等爬虫会命中规则中的蜘蛛白名单，便于广告审核。</li
          >
          <li
            ><strong>接口失败：</strong>建议默认展示审核页（白页），避免 API
            异常时暴露真实内容。</li
          >
          <li
            ><strong>调试：</strong>在斗篷后台「API
            调用日志」查看每次请求是否到达及判定结果。</li
          >
        </ul>

        <h4>5.8 测试步骤</h4>
        <ol class="guide-steps">
          <li>用浏览器访问 WordPress 首页，观察是否跳转到预期页面。</li>
          <li
            >在斗篷后台查看 API 调用日志，确认有记录且
            <code>page</code> 符合预期。</li
          >
          <li>切换手机 UA 或代理 IP 测试不同规则（投放地区、屏蔽 PC 等）。</li>
          <li>清空站点缓存后再次验证，排除缓存干扰。</li>
        </ol>

        <h4>5.9 官方 WordPress 插件（推荐）</h4>
        <p>
          无需手写代码，可直接安装官方插件
          <strong>Cloak API</strong>，在后台可视化配置教程中的几种接入方式。
        </p>
        <ul class="guide-list">
          <li
            ><strong>插件目录：</strong
            ><code>wordpress-plugin/cloak-api/</code></li
          >
          <li
            ><strong>安装：</strong>将 <code>cloak-api</code> 文件夹上传到
            WordPress 的 <code>wp-content/plugins/</code>，后台启用插件。</li
          >
          <li><strong>配置入口：</strong>WordPress 后台 → 设置 → API 斗篷</li>
          <li
            ><strong>支持模式：</strong>审核页与真实页均支持站内页面 / 外部链接
            / HTML 跳转 / HTML 直接输出</li
          >
          <li
            ><strong>排除路径：</strong>可配置
            <code>/blog/*</code> 等规则，匹配路径不执行斗篷判定</li
          >
          <li
            ><strong>安全：</strong>可在 <code>wp-config.php</code> 中定义
            <code>CLOAK_API_TOKEN</code> 和 <code>CLOAK_API_URL</code></li
          >
        </ul>
        <div class="code-block">
          <div class="code-label">wp-config.php 可选配置</div>
          <pre>
define('CLOAK_API_URL', '{{ apiBaseUrl }}/open/decision');
define('CLOAK_API_TOKEN', 'ck_你的Token');</pre
          >
        </div>
      </div>
    </ele-card>
  </ele-page>
</template>

<script setup>
  import { useRouter } from 'vue-router';
  import { ArrowLeft } from '@element-plus/icons-vue';

  defineOptions({ name: 'MerchantApiKeyGuide' });

  const router = useRouter();
  const apiBaseUrl = import.meta.env.VITE_API_URL || '/api';

  const cloakSharedCode = `<?php
define('CLOAK_API_URL', '${apiBaseUrl}/open/decision');
define('CLOAK_API_TOKEN', 'ck_你的Token');

function cloak_get_client_ip() {
    $keys = ['HTTP_CF_CONNECTING_IP', 'HTTP_X_FORWARDED_FOR', 'REMOTE_ADDR'];
    foreach ($keys as $key) {
        if (empty($_SERVER[$key])) continue;
        $ip = trim(explode(',', $_SERVER[$key])[0]);
        if (filter_var($ip, FILTER_VALIDATE_IP)) return $ip;
    }
    return '';
}

function cloak_decide_page() {
    $body = wp_json_encode([
        'ip'             => cloak_get_client_ip(),
        'userAgent'      => $_SERVER['HTTP_USER_AGENT'] ?? '',
        'referer'        => $_SERVER['HTTP_REFERER'] ?? '',
        'acceptLanguage' => $_SERVER['HTTP_ACCEPT_LANGUAGE'] ?? '',
    ]);

    $response = wp_remote_post(CLOAK_API_URL, [
        'timeout' => 3,
        'headers' => [
            'Content-Type' => 'application/json',
            'X-API-Key'    => CLOAK_API_TOKEN,
        ],
        'body' => $body,
    ]);

    if (is_wp_error($response)) return 'review';

    $data = json_decode(wp_remote_retrieve_body($response), true);
    return $data['data']['page'] ?? 'review';
}`;

  const wordpressWpPageCode = `<?php
// 方案一：审核页、真实页均为 WordPress 站内页面
define('CLOAK_REVIEW_PAGE_ID', 123);  // 审核页页面 ID
define('CLOAK_REAL_PAGE_ID', 456);    // 真实页页面 ID

add_action('template_redirect', function () {
    if (is_admin() || wp_doing_ajax() || wp_doing_cron()) return;

    $page = cloak_decide_page();
    $target_id = ($page === 'real') ? CLOAK_REAL_PAGE_ID : CLOAK_REVIEW_PAGE_ID;

    if (is_page($target_id)) return;

    wp_safe_redirect(get_permalink($target_id), 302);
    exit;
});`;

  const wordpressExternalCode = `<?php
// 方案二：审核页为外部链接，真实页为站内页面
define('CLOAK_REVIEW_URL', 'https://safe.example.com/landing'); // 外部审核页
define('CLOAK_REAL_PAGE_ID', 456);

/** 外部链接域名白名单，防止误配置 */
function cloak_is_allowed_external_url($url) {
    $host = wp_parse_url($url, PHP_URL_HOST);
    $allowed = ['safe.example.com', 'landing.example.com'];
    return $host && in_array($host, $allowed, true);
}

add_action('template_redirect', function () {
    if (is_admin() || wp_doing_ajax() || wp_doing_cron()) return;

    $page = cloak_decide_page();

    if ($page === 'review') {
        if (!cloak_is_allowed_external_url(CLOAK_REVIEW_URL)) return;
        wp_redirect(esc_url_raw(CLOAK_REVIEW_URL), 302);
        exit;
    }

    if (is_page(CLOAK_REAL_PAGE_ID)) return;

    wp_safe_redirect(get_permalink(CLOAK_REAL_PAGE_ID), 302);
    exit;
});`;

  const wordpressHtmlRedirectCode = `<?php
// 方案三-A：跳转到服务器上的静态 HTML 文件
// 文件路径示例：/wp-content/cloak/review.html
define('CLOAK_REVIEW_HTML_URL', home_url('/wp-content/cloak/review.html'));
define('CLOAK_REAL_PAGE_ID', 456);

add_action('template_redirect', function () {
    if (is_admin() || wp_doing_ajax() || wp_doing_cron()) return;

    $page = cloak_decide_page();

    if ($page === 'review') {
        // 避免已在 HTML 页时重复跳转
        $request_uri = $_SERVER['REQUEST_URI'] ?? '';
        if (str_contains($request_uri, '/wp-content/cloak/review.html')) return;

        wp_safe_redirect(CLOAK_REVIEW_HTML_URL, 302);
        exit;
    }

    if (is_page(CLOAK_REAL_PAGE_ID)) return;

    wp_safe_redirect(get_permalink(CLOAK_REAL_PAGE_ID), 302);
    exit;
});`;

  const wordpressHtmlOutputCode = `<?php
// 方案三-B：PHP 读取 HTML 文件并直接输出（地址栏 URL 不变）
// 建议将 HTML 放在 Web 根目录外，仅 PHP 可读
define('CLOAK_REVIEW_HTML_PATH', ABSPATH . '../cloak-pages/review.html');
define('CLOAK_REAL_PAGE_ID', 456);

function cloak_output_review_html() {
    if (!is_readable(CLOAK_REVIEW_HTML_PATH)) return false;

    status_header(200);
    header('Content-Type: text/html; charset=utf-8');
    header('X-Robots-Tag: noindex, nofollow');
    readfile(CLOAK_REVIEW_HTML_PATH);
    return true;
}

add_action('template_redirect', function () {
    if (is_admin() || wp_doing_ajax() || wp_doing_cron()) return;

    $page = cloak_decide_page();

    if ($page === 'review') {
        if (cloak_output_review_html()) exit;
        return; // 文件不存在时保持原页面，避免白屏
    }

    if (is_page(CLOAK_REAL_PAGE_ID)) return;

    wp_safe_redirect(get_permalink(CLOAK_REAL_PAGE_ID), 302);
    exit;
});`;

  const resultRows = [
    { page: 'review', meaning: '访问审核页（白页）' },
    { page: 'real', meaning: '访问真实页（黑页）' }
  ];

  const goBack = () => {
    router.push('/merchant/api-key');
  };
</script>

<style scoped>
  .guide-header {
    margin-bottom: 20px;
  }

  .guide-title {
    margin: 12px 0 8px;
    font-size: 22px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .guide-desc {
    margin: 0;
    color: var(--el-text-color-secondary);
    line-height: 1.6;
  }

  .guide-alert {
    margin-bottom: 24px;
  }

  .guide-section {
    margin-bottom: 28px;
  }

  .guide-section h3 {
    margin: 0 0 10px;
    font-size: 16px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .guide-section h4 {
    margin: 20px 0 8px;
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .guide-section h5 {
    margin: 16px 0 8px;
    font-size: 13px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .guide-note {
    margin: 16px 0 8px;
    padding: 10px 14px;
    font-size: 13px;
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-lighter);
    border-radius: 6px;
    border-left: 3px solid var(--el-color-primary);
  }

  .guide-sub-list {
    list-style: none;
    margin-top: 4px;
  }

  .guide-sub-list ul {
    margin: 4px 0 0;
    padding-left: 18px;
  }

  .guide-section--wordpress strong {
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .guide-steps {
    margin: 8px 0 0;
    padding-left: 20px;
    color: var(--el-text-color-regular);
    line-height: 1.9;
  }

  .guide-section p {
    margin: 0 0 12px;
    color: var(--el-text-color-regular);
    line-height: 1.7;
  }

  .guide-list {
    margin: 12px 0 0;
    padding-left: 20px;
    color: var(--el-text-color-regular);
    line-height: 1.8;
  }

  .code-block {
    margin-top: 12px;
    background: var(--el-fill-color-light);
    border-radius: 8px;
    overflow: hidden;
  }

  .code-label {
    padding: 8px 14px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .code-block pre {
    margin: 0;
    padding: 14px;
    font-size: 13px;
    line-height: 1.6;
    overflow-x: auto;
    white-space: pre-wrap;
    word-break: break-all;
    color: var(--el-text-color-primary);
  }

  code {
    padding: 2px 6px;
    font-size: 13px;
    background: var(--el-fill-color);
    border-radius: 4px;
  }
</style>
