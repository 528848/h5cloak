<?php

if (!defined('ABSPATH')) {
    exit;
}

class Cloak_Attr_Admin
{
    private const PAGE = 'cloak-attribution';

    public function __construct()
    {
        add_action('admin_menu', [$this, 'menu']);
        add_action('admin_init', [$this, 'register']);
        add_action('admin_notices', [$this, 'notices']);
        add_action('wp_ajax_cloak_attr_test', [$this, 'ajax_test']);
        add_filter('plugin_action_links_' . plugin_basename(CLOAK_ATTR_FILE), function ($links) {
            array_unshift($links, '<a href="' . esc_url(admin_url('options-general.php?page=' . self::PAGE)) . '">' . esc_html__('设置', 'cloak-attribution') . '</a>');
            return $links;
        });
    }

    public function menu(): void
    {
        add_options_page('Cloak Attribution', 'Cloak 归因', 'manage_options', self::PAGE, [$this, 'render']);
    }

    public function register(): void
    {
        register_setting('cloak_attr_group', Cloak_Attr_Settings::OPTION_KEY, [
            'type' => 'array',
            'sanitize_callback' => static function ($input) {
                return Cloak_Attr_Settings::sanitize(is_array($input) ? $input : []);
            },
        ]);
    }

    public function notices(): void
    {
        if (!current_user_can('manage_options')) {
            return;
        }
        $migrated = get_option('cloak_attr_migrated_from_v1');
        if (is_array($migrated) && !empty($migrated['needs_secret'])) {
            $settings = Cloak_Attr_Settings::get();
            if ($settings['secret'] === '') {
                echo '<div class="notice notice-warning"><p>' . sprintf(
                    esc_html__('Cloak 归因：已从旧版插件迁移配置，但 v2 需要站点「服务端密钥 sk_...」，请到 %s 填写。', 'cloak-attribution'),
                    '<a href="' . esc_url(admin_url('options-general.php?page=' . self::PAGE)) . '">' . esc_html__('设置页', 'cloak-attribution') . '</a>'
                ) . '</p></div>';
            } else {
                delete_option('cloak_attr_migrated_from_v1');
            }
        }
    }

    public function ajax_test(): void
    {
        check_ajax_referer('cloak_attr_test');
        if (!current_user_can('manage_options')) {
            wp_send_json_error(['message' => 'forbidden'], 403);
        }
        $current = Cloak_Attr_Settings::get();
        $input = isset($_POST['settings']) && is_array($_POST['settings']) ? wp_unslash($_POST['settings']) : []; // phpcs:ignore
        $settings = Cloak_Attr_Settings::sanitize($input, $current);
        $result = Cloak_Attr_Client::test($settings);
        if ($result['ok']) {
            wp_send_json_success($result);
        }
        wp_send_json_error($result);
    }

    /** 渲染单侧（review/real）的输出模式单选组。 */
    private function mode_options(string $opt, string $side, array $s, array $pages): void
    {
        $m = (string) ($s[$side . '_mode'] ?? '');
        $name = $opt . '[' . $side . '_mode]';
        ?>
        <fieldset>
            <?php if ($side === 'real') : ?>
                <label><input type="radio" name="<?php echo esc_attr($name); ?>" value="current" <?php checked($m, 'current'); ?>> 继续显示当前网站（推荐，地址不变）</label><br>
            <?php else : ?>
                <label><input type="radio" name="<?php echo esc_attr($name); ?>" value="api_html" <?php checked($m, 'api_html'); ?>> 使用平台配置的审核模板（推荐，在商户后台维护，地址不变）</label><br>
            <?php endif; ?>
            <label><input type="radio" name="<?php echo esc_attr($name); ?>" value="wp_page" <?php checked($m, 'wp_page'); ?>> 渲染站内页面（地址不变）：</label>
            <select name="<?php echo esc_attr($opt . '[' . $side . '_page_id]'); ?>">
                <option value="0">— 选择页面 —</option>
                <?php foreach ($pages as $p) : ?>
                    <option value="<?php echo (int) $p->ID; ?>" <?php selected((int) ($s[$side . '_page_id'] ?? 0), (int) $p->ID); ?>><?php echo esc_html($p->post_title); ?></option>
                <?php endforeach; ?>
            </select><br>
            <label><input type="radio" name="<?php echo esc_attr($name); ?>" value="html_file" <?php checked($m, 'html_file'); ?>> 输出服务器 HTML 文件（地址不变）：</label>
            <input type="text" class="regular-text" name="<?php echo esc_attr($opt . '[' . $side . '_html_path]'); ?>" value="<?php echo esc_attr($s[$side . '_html_path'] ?? ''); ?>" placeholder="<?php echo esc_attr(ABSPATH . $side . '/index.html'); ?>"><br>
            <label><input type="radio" name="<?php echo esc_attr($name); ?>" value="proxy_url" <?php checked($m, 'proxy_url'); ?>> 服务端拉取外部地址内容（地址不变）：</label>
            <input type="url" class="regular-text" name="<?php echo esc_attr($opt . '[' . $side . '_proxy_url]'); ?>" value="<?php echo esc_attr($s[$side . '_proxy_url'] ?? ''); ?>" placeholder="https://content.example.com/"><br>
            <hr style="margin:10px 0;border:0;border-top:1px solid #e0e0e0">
            <p class="description" style="margin:0 0 6px">以下为兼容 V1 的跳转，<strong>浏览器地址会变成目标地址</strong>：</p>
            <label><input type="radio" name="<?php echo esc_attr($name); ?>" value="wp_page_redirect" <?php checked($m, 'wp_page_redirect'); ?>> 跳转到站内页面（302）：使用上方所选页面的固定链接</label><br>
            <label><input type="radio" name="<?php echo esc_attr($name); ?>" value="external_redirect" <?php checked($m, 'external_redirect'); ?>> 跳转到外部链接（302）：</label>
            <input type="url" class="regular-text" name="<?php echo esc_attr($opt . '[' . $side . '_external_url]'); ?>" value="<?php echo esc_attr($s[$side . '_external_url'] ?? ''); ?>" placeholder="https://go.example.com/"><br>
            <label><input type="radio" name="<?php echo esc_attr($name); ?>" value="html_redirect" <?php checked($m, 'html_redirect'); ?>> 跳转到 HTML 文件地址（302）：</label>
            <input type="url" class="regular-text" name="<?php echo esc_attr($opt . '[' . $side . '_html_url]'); ?>" value="<?php echo esc_attr($s[$side . '_html_url'] ?? ''); ?>" placeholder="https://your-site.com/wp-content/cloak/page.html">
        </fieldset>
        <?php
    }

    public function render(): void
    {
        if (!current_user_can('manage_options')) {
            return;
        }
        $s = Cloak_Attr_Settings::get();
        $opt = Cloak_Attr_Settings::OPTION_KEY;
        $secret_locked = defined('CLOAK_SECRET') && CLOAK_SECRET;
        $pages = get_pages(['post_status' => 'publish']);
        ?>
        <div class="wrap">
            <h1>Cloak 归因（v<?php echo esc_html(CLOAK_ATTR_VERSION); ?>）</h1>
            <p class="description">带有效归因参数（推广码、点击参数等）的访客看到真实页面，其余访客在同一地址看到审核内容。判定由斗篷网关完成，本插件只负责调用与输出。</p>
            <form method="post" action="options.php" id="cloak-attr-form">
                <?php settings_fields('cloak_attr_group'); ?>
                <h2 class="title">连接</h2>
                <table class="form-table" role="presentation">
                    <tr>
                        <th scope="row">启用</th>
                        <td><label><input type="checkbox" name="<?php echo $opt; ?>[enabled]" value="1" <?php checked($s['enabled']); ?>> 对前台访客生效</label></td>
                    </tr>
                    <tr>
                        <th scope="row"><label for="ca-api-base">网关地址</label></th>
                        <td><input id="ca-api-base" type="url" class="regular-text" name="<?php echo $opt; ?>[api_base]" value="<?php echo esc_attr($s['api_base']); ?>" placeholder="https://entry.example.com" <?php disabled(defined('CLOAK_API_BASE') && CLOAK_API_BASE); ?>>
                            <p class="description">斗篷网关的访问地址（不带路径）。可在 wp-config.php 中用 <code>CLOAK_API_BASE</code> 覆盖。</p></td>
                    </tr>
                    <tr>
                        <th scope="row"><label for="ca-secret">服务端密钥</label></th>
                        <td><input id="ca-secret" type="password" class="regular-text" name="<?php echo $opt; ?>[secret]" value="" autocomplete="new-password" placeholder="<?php echo $s['secret'] ? esc_attr(substr($s['secret'], 0, 12) . '…（已保存，留空保持不变）') : 'sk_...'; ?>" <?php disabled($secret_locked); ?>>
                            <p class="description">商户后台 → 站点 → 密钥 中的 <code>sk_</code> 开头密钥。建议在 wp-config.php 中定义 <code>CLOAK_SECRET</code> 而不是保存在数据库。</p></td>
                    </tr>
                    <tr>
                        <th scope="row"><label for="ca-timeout">超时（秒）</label></th>
                        <td><input id="ca-timeout" type="number" min="1" max="10" class="small-text" name="<?php echo $opt; ?>[timeout]" value="<?php echo esc_attr($s['timeout']); ?>"></td>
                    </tr>
                    <tr>
                        <th scope="row">网关不可用时</th>
                        <td>
                            <label><input type="radio" name="<?php echo $opt; ?>[fail_mode]" value="review" <?php checked($s['fail_mode'], 'review'); ?>> 显示审核内容（更安全）</label><br>
                            <label><input type="radio" name="<?php echo $opt; ?>[fail_mode]" value="real" <?php checked($s['fail_mode'], 'real'); ?>> 显示真实内容</label>
                        </td>
                    </tr>
                    <tr>
                        <th scope="row"><label for="ca-cache">判定缓存（秒）</label></th>
                        <td><input id="ca-cache" type="number" min="0" max="600" class="small-text" name="<?php echo $opt; ?>[cache_ttl]" value="<?php echo esc_attr($s['cache_ttl']); ?>">
                            <p class="description">仅缓存「已有访客令牌且无参数」的回访判定，带参数的访问始终实时判定。0 关闭。</p></td>
                    </tr>
                    <tr>
                        <th scope="row">连接测试</th>
                        <td><button type="button" class="button" id="ca-test">测试连接</button> <span id="ca-test-result" style="margin-left:8px"></span></td>
                    </tr>
                </table>

                <h2 class="title">审核内容（未归因访客看到）</h2>
                <table class="form-table" role="presentation">
                    <tr>
                        <th scope="row">输出方式</th>
                        <td>
                            <?php $this->mode_options($opt, 'review', $s, $pages); ?>
                            <p class="description">前 4 种在当前地址以 200 输出，不改 URL；后 3 种是兼容 V1 的 302 跳转，浏览器地址会变。</p>
                        </td>
                    </tr>
                    <tr>
                        <th scope="row">内置浏览器引导条</th>
                        <td><label><input type="checkbox" name="<?php echo $opt; ?>[inject_banner]" value="1" <?php checked($s['inject_banner']); ?>> 访客在微信/抖音等内置浏览器中看到真实页面时，页脚显示「在浏览器中打开」引导条（需站点开启交接）</label></td>
                    </tr>
                </table>

                <h2 class="title">真实内容（已归因访客看到）</h2>
                <table class="form-table" role="presentation">
                    <tr>
                        <th scope="row">输出方式</th>
                        <td>
                            <?php $this->mode_options($opt, 'real', $s, $pages); ?>
                            <p class="description">默认「继续显示当前网站」：带有效归因参数的访客直接看到他打开的这篇页面，无需另建真实页。仅在需要把真实访客导向另一个页面 / 外链 / 文件时才改为其它模式。</p>
                        </td>
                    </tr>
                    <tr>
                        <th scope="row"><label for="ca-ext-hosts">外链白名单</label></th>
                        <td><textarea id="ca-ext-hosts" class="large-text code" rows="2" name="<?php echo $opt; ?>[external_allowed_hosts]" placeholder="go.example.com&#10;promo.example.net"><?php echo esc_textarea($s['external_allowed_hosts']); ?></textarea>
                            <p class="description">仅当审核页或真实页选择「跳转到外部链接」时生效。每行一个主机，只允许 https。</p></td>
                    </tr>
                </table>

                <h2 class="title">范围</h2>
                <table class="form-table" role="presentation">
                    <tr>
                        <th scope="row">跳过</th>
                        <td><label><input type="checkbox" name="<?php echo $opt; ?>[skip_logged_in]" value="1" <?php checked($s['skip_logged_in']); ?>> 已登录用户不判定</label>
                            <p class="description">后台、登录页、REST、Cron、Feed 始终跳过。</p></td>
                    </tr>
                    <tr>
                        <th scope="row"><label for="ca-exclude">排除路径</label></th>
                        <td><textarea id="ca-exclude" class="large-text code" rows="5" name="<?php echo $opt; ?>[exclude_paths]"><?php echo esc_textarea($s['exclude_paths']); ?></textarea>
                            <p class="description">每行一条，支持 <code>/path/*</code> 前缀匹配。排除的路径直接显示真实内容。</p></td>
                    </tr>
                </table>

                <h2 class="title">WooCommerce</h2>
                <?php if (!Cloak_Attr_Woo::active()) : ?>
                    <p class="description">未检测到 WooCommerce，以下设置暂不生效。</p>
                <?php endif; ?>
                <table class="form-table" role="presentation">
                    <tr>
                        <th scope="row">商店页面</th>
                        <td>
                            <label><input type="checkbox" name="<?php echo $opt; ?>[woo_enabled]" value="1" <?php checked($s['woo_enabled']); ?>> 对 WooCommerce 页面启用范围规则</label>
                            <fieldset style="margin-top:8px">
                                <label><input type="radio" name="<?php echo $opt; ?>[woo_scope]" value="all" <?php checked($s['woo_scope'], 'all'); ?>> 全部商店页面都判定</label><br>
                                <label><input type="radio" name="<?php echo $opt; ?>[woo_scope]" value="products" <?php checked($s['woo_scope'], 'products'); ?>> 只判定下列商品 / 分类（含含有这些商品的购物车与结账页），其余商店页面直接显示真实内容</label>
                            </fieldset>
                        </td>
                    </tr>
                    <tr>
                        <th scope="row"><label for="ca-woo-ids">商品 ID</label></th>
                        <td><textarea id="ca-woo-ids" class="large-text code" rows="2" name="<?php echo $opt; ?>[woo_product_ids]" placeholder="12, 34, 56"><?php echo esc_textarea($s['woo_product_ids']); ?></textarea></td>
                    </tr>
                    <tr>
                        <th scope="row"><label for="ca-woo-cats">分类别名（slug）</label></th>
                        <td><textarea id="ca-woo-cats" class="large-text code" rows="2" name="<?php echo $opt; ?>[woo_category_slugs]" placeholder="promo, hot-sale"><?php echo esc_textarea($s['woo_category_slugs']); ?></textarea></td>
                    </tr>
                </table>
                <?php submit_button(); ?>
            </form>
        </div>
        <script>
        (function () {
            var btn = document.getElementById('ca-test');
            var out = document.getElementById('ca-test-result');
            if (!btn) return;
            btn.addEventListener('click', function () {
                btn.disabled = true;
                out.textContent = '测试中…';
                out.style.color = '';
                var form = document.getElementById('cloak-attr-form');
                var fd = new FormData(form);
                var body = new URLSearchParams();
                body.append('action', 'cloak_attr_test');
                body.append('_ajax_nonce', '<?php echo esc_js(wp_create_nonce('cloak_attr_test')); ?>');
                fd.forEach(function (v, k) {
                    var m = k.match(/^<?php echo $opt; ?>\[(.+)\]$/);
                    if (m) body.append('settings[' + m[1] + ']', v);
                });
                fetch(ajaxurl, { method: 'POST', credentials: 'same-origin', body: body })
                    .then(function (r) { return r.json(); })
                    .then(function (j) {
                        var d = j.data || {};
                        out.textContent = d.message || (j.success ? 'OK' : '失败');
                        out.style.color = j.success ? '#1a7f37' : '#b32d2e';
                    })
                    .catch(function (e) { out.textContent = '请求失败：' + e; out.style.color = '#b32d2e'; })
                    .finally(function () { btn.disabled = false; });
            });
        })();
        </script>
        <?php
    }
}
