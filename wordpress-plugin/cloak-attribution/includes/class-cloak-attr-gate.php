<?php

if (!defined('ABSPATH')) {
    exit;
}

/** 前台判定：template_redirect 优先级 1。 */
class Cloak_Attr_Gate
{
    private ?array $decision = null;

    public function __construct()
    {
        add_action('template_redirect', [$this, 'handle'], 1);
        add_action('wp_footer', [$this, 'footer'], 99);
    }

    public function handle(): void
    {
        $settings = Cloak_Attr_Settings::get();
        if (empty($settings['enabled']) || !$this->should_run($settings)) {
            return;
        }
        $data = Cloak_Attr_Client::decide($settings);
        if ($data === null) {
            $data = ['content' => $settings['fail_mode'] === 'real' ? 'real' : 'review', 'failed' => true];
        }
        $this->decision = $data;

        if (!empty($data['cookie']) && !empty($data['visitorToken']) && !headers_sent()) {
            $c = $data['cookie'];
            setcookie($c['name'] ?? Cloak_Attr_Client::COOKIE_DEFAULT, (string) $data['visitorToken'], [
                'expires' => time() + (int) ($c['maxAge'] ?? 2592000),
                'path' => '/',
                'domain' => (string) ($c['domain'] ?? ''),
                'secure' => !empty($c['secure']) && is_ssl(),
                'httponly' => true,
                'samesite' => 'Lax',
            ]);
        }

        $content = (string) ($data['content'] ?? 'review');
        if ($content === 'real' || $content === 'bypass') {
            // 放行：按真实页模式输出（默认 current，即继续渲染访客当前打开的页面）
            $this->render('real', $settings, $data);
            return;
        }
        if ($content === 'block') {
            status_header(403);
            nocache_headers();
            echo '<!doctype html><html><head><meta charset="utf-8"><title>403</title></head><body style="font-family:sans-serif;text-align:center;padding:80px 20px"><h1>403</h1></body></html>';
            exit;
        }
        if ($content === 'not_found') {
            status_header(404);
            nocache_headers();
            echo '<!doctype html><html><head><meta charset="utf-8"><title>404</title></head><body style="font-family:sans-serif;text-align:center;padding:80px 20px"><h1>404</h1></body></html>';
            exit;
        }
        $this->render('review', $settings, $data);
    }

    /** 引导条：放行且在内置浏览器内时，由接口给出 handoffUrl。 */
    public function footer(): void
    {
        $d = $this->decision;
        if (!$d || empty($d['handoffUrl']) || empty($d['handoffBanner']['enabled'])) {
            return;
        }
        $settings = Cloak_Attr_Settings::get();
        if (empty($settings['inject_banner']) || !empty($_COOKIE['_ck_hb'])) {
            return;
        }
        $href = (string) $d['handoffUrl'];
        $os = (string) ($d['inAppBrowser']['os'] ?? '');
        if ($os === 'ios' && strpos($href, 'https://') === 0) {
            $href = 'x-safari-' . $href;
        } elseif ($os === 'android') {
            $p = wp_parse_url($href);
            if (!empty($p['host'])) {
                $href = 'intent://' . $p['host'] . ($p['path'] ?? '/') . (isset($p['query']) ? '?' . $p['query'] : '') . '#Intent;scheme=' . ($p['scheme'] ?? 'https') . ';action=android.intent.action.VIEW;end';
            }
        }
        $text = esc_html((string) ($d['handoffBanner']['text'] ?? '在浏览器中打开，继续访问'));
        echo '<div id="ck-handoff" style="position:fixed;left:0;right:0;bottom:0;z-index:2147483647;background:#111;color:#fff;font:14px/1.4 -apple-system,BlinkMacSystemFont,sans-serif;padding:12px 14px;display:flex;align-items:center;gap:12px;box-shadow:0 -2px 10px rgba(0,0,0,.25)">'
            . '<span style="flex:1">' . $text . '</span>'
            . '<a href="' . esc_attr($href) . '" style="background:#fff;color:#111;border-radius:6px;padding:8px 12px;text-decoration:none;font-weight:600;white-space:nowrap">打开</a>'
            . '<button type="button" onclick="document.getElementById(\'ck-handoff\').style.display=\'none\';document.cookie=\'_ck_hb=1;path=/;SameSite=Lax\'" style="background:none;border:0;color:#aaa;font-size:20px;line-height:1;padding:0 4px" aria-label="close">&times;</button>'
            . '</div>';
    }

    private function should_run(array $settings): bool
    {
        if (is_admin() || wp_doing_ajax() || wp_doing_cron() || wp_doing_autosave() || is_feed() || is_robots()) {
            return false;
        }
        if ((defined('REST_REQUEST') && REST_REQUEST) || (defined('WP_CLI') && WP_CLI) || (defined('DOING_CRON') && DOING_CRON)) {
            return false;
        }
        if (!empty($settings['skip_logged_in']) && is_user_logged_in()) {
            return false;
        }
        $path = (string) strtok((string) ($_SERVER['REQUEST_URI'] ?? '/'), '?');
        foreach (Cloak_Attr_Settings::exclude_patterns($settings) as $pattern) {
            if (Cloak_Attr_Settings::path_matches($path === '' ? '/' : $path, $pattern)) {
                return false;
            }
        }
        // 审核页/真实页指向的站内页面本身不再判定，避免站内渲染或跳转形成循环
        foreach (['review', 'real'] as $side) {
            $mode = (string) ($settings[$side . '_mode'] ?? '');
            $is_page_mode = $mode === Cloak_Attr_Settings::REVIEW_WP_PAGE || $mode === Cloak_Attr_Settings::MODE_WP_PAGE_REDIRECT;
            $page_id = (int) ($settings[$side . '_page_id'] ?? 0);
            if ($is_page_mode && $page_id > 0 && is_page($page_id)) {
                return false;
            }
        }
        return Cloak_Attr_Woo::should_gate($settings);
    }

    /**
     * 审核页与真实页共用的输出器。
     * $side = review | real。同域模式以 200 在当前地址输出；跳转模式返回 302；
     * current / 跳转命中目标页 时直接返回，交回 WordPress 继续渲染当前页面。
     */
    private function render(string $side, array $settings, array $data): void
    {
        $default = $side === 'real' ? Cloak_Attr_Settings::MODE_CURRENT : Cloak_Attr_Settings::REVIEW_API_HTML;
        $mode = (string) ($settings[$side . '_mode'] ?? $default);

        switch ($mode) {
            case Cloak_Attr_Settings::MODE_CURRENT:
                // 继续渲染访客当前打开的页面
                header('X-Cloak-Content: ' . $side);
                return;

            case Cloak_Attr_Settings::MODE_WP_PAGE_REDIRECT:
                $this->redirect_wp_page((int) ($settings[$side . '_page_id'] ?? 0), $side);
                header('X-Cloak-Content: ' . $side); // 已在目标页：继续渲染
                return;

            case Cloak_Attr_Settings::MODE_EXTERNAL_REDIRECT:
                $this->redirect_external((string) ($settings[$side . '_external_url'] ?? ''), $settings, $side);
                header('X-Cloak-Content: ' . $side);
                return;

            case Cloak_Attr_Settings::MODE_HTML_REDIRECT:
                $this->redirect_html((string) ($settings[$side . '_html_url'] ?? ''), $side);
                header('X-Cloak-Content: ' . $side);
                return;

            case Cloak_Attr_Settings::REVIEW_WP_PAGE:
                $page_id = (int) ($settings[$side . '_page_id'] ?? 0);
                if ($page_id > 0 && get_post_status($page_id) === 'publish') {
                    $this->same_url_headers($side);
                    $this->render_wp_page($page_id);
                    exit;
                }
                break;

            case Cloak_Attr_Settings::REVIEW_HTML_FILE:
                $path = (string) ($settings[$side . '_html_path'] ?? '');
                if ($path !== '' && is_readable($path)) {
                    $this->same_url_headers($side);
                    header('Content-Type: text/html; charset=utf-8');
                    readfile($path);
                    exit;
                }
                break;

            case Cloak_Attr_Settings::REVIEW_PROXY_URL:
                $html = $this->fetch_remote((string) ($settings[$side . '_proxy_url'] ?? ''));
                if ($html !== null) {
                    $this->same_url_headers($side);
                    header('Content-Type: text/html; charset=utf-8');
                    echo $html; // phpcs:ignore WordPress.Security.EscapeOutput
                    exit;
                }
                break;
        }

        // 走到这里说明上面的模式未能输出（配置缺失或模式为 api_html）
        if ($side === 'real') {
            // 真实页兜底：继续渲染访客当前页面，绝不回落到审核模板
            header('X-Cloak-Content: real');
            return;
        }

        // 审核页兜底：接口返回的平台模板 HTML
        $this->same_url_headers('review');
        header('Content-Type: text/html; charset=utf-8');
        $hint = $data['renderHint'] ?? [];
        if (($hint['type'] ?? '') === 'html' && !empty($hint['html'])) {
            echo $hint['html']; // phpcs:ignore WordPress.Security.EscapeOutput
            exit;
        }
        if (($hint['type'] ?? '') === 'url' && !empty($hint['url'])) {
            $html = $this->fetch_remote((string) $hint['url']);
            if ($html !== null) {
                echo $html; // phpcs:ignore WordPress.Security.EscapeOutput
                exit;
            }
        }
        echo '<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>' . esc_html(get_bloginfo('name')) . '</title></head><body style="font-family:sans-serif;max-width:720px;margin:48px auto;padding:0 20px;line-height:1.7"><h1>' . esc_html(get_bloginfo('name')) . '</h1><p>' . esc_html(get_bloginfo('description')) . '</p></body></html>';
        exit;
    }

    /** 同域 200 输出的公共响应头。审核页加 noindex，真实页不加。 */
    private function same_url_headers(string $side): void
    {
        status_header(200);
        nocache_headers();
        header('X-Cloak-Content: ' . $side);
        if ($side === 'review') {
            header('X-Robots-Tag: noindex, nofollow');
        }
    }

    /** 302 到站内页面固定链接；已在目标页则不跳，交回当前渲染。 */
    private function redirect_wp_page(int $page_id, string $side): void
    {
        if ($page_id <= 0 || is_page($page_id)) {
            return;
        }
        $permalink = get_permalink($page_id);
        if (!$permalink) {
            return;
        }
        header('X-Cloak-Content: ' . $side);
        wp_safe_redirect($permalink, 302);
        exit;
    }

    /** 302 到外部 https 地址，须通过白名单校验。 */
    private function redirect_external(string $url, array $settings, string $side): void
    {
        $url = trim($url);
        if ($url === '' || !$this->is_allowed_external_url($url, $settings) || $this->is_current_url($url)) {
            return;
        }
        header('X-Cloak-Content: ' . $side);
        wp_redirect(esc_url_raw($url), 302); // phpcs:ignore WordPress.Security.SafeRedirect
        exit;
    }

    /** 302 到 HTML 文件公开 URL。 */
    private function redirect_html(string $url, string $side): void
    {
        $url = trim($url);
        if ($url === '' || $this->is_current_url($url)) {
            return;
        }
        $path = (string) (wp_parse_url($url, PHP_URL_PATH) ?? '');
        if ($path !== '' && strpos((string) ($_SERVER['REQUEST_URI'] ?? ''), $path) !== false) {
            return;
        }
        header('X-Cloak-Content: ' . $side);
        wp_safe_redirect($url, 302);
        exit;
    }

    private function is_allowed_external_url(string $url, array $settings): bool
    {
        if (!wp_http_validate_url($url) || wp_parse_url($url, PHP_URL_SCHEME) !== 'https') {
            return false;
        }
        $host = strtolower((string) wp_parse_url($url, PHP_URL_HOST));
        $allowed = Cloak_Attr_Settings::allowed_external_hosts($settings);
        return $host !== '' && in_array($host, $allowed, true);
    }

    private function is_current_url(string $url): bool
    {
        $current = (is_ssl() ? 'https://' : 'http://') . (string) ($_SERVER['HTTP_HOST'] ?? '') . (string) ($_SERVER['REQUEST_URI'] ?? '');
        return untrailingslashit(strtok($current, '?')) === untrailingslashit(strtok(trim($url), '?'));
    }

    /** 渲染指定页面但保持当前 URL。 */
    private function render_wp_page(int $page_id): void
    {
        global $wp_query, $post;
        $post = get_post($page_id); // phpcs:ignore WordPress.WP.GlobalVariablesOverride
        $wp_query = new WP_Query(['page_id' => $page_id, 'post_type' => 'page']); // phpcs:ignore WordPress.WP.GlobalVariablesOverride
        $wp_query->is_404 = false;
        $wp_query->is_page = true;
        $wp_query->is_singular = true;
        setup_postdata($post);
        $template = get_page_template();
        if ($template && file_exists($template)) {
            include $template;
            return;
        }
        get_header();
        echo '<main>' . apply_filters('the_content', $post->post_content) . '</main>'; // phpcs:ignore WordPress.Security.EscapeOutput
        get_footer();
    }

    private function fetch_remote(string $url): ?string
    {
        if ($url === '') {
            return null;
        }
        $cache_key = 'cloak_attr_review_' . substr(hash('sha256', $url), 0, 32);
        $cached = get_transient($cache_key);
        if (is_string($cached) && $cached !== '') {
            return $cached;
        }
        $res = wp_remote_get($url, ['timeout' => 5, 'headers' => ['Accept' => 'text/html']]);
        if (is_wp_error($res) || (int) wp_remote_retrieve_response_code($res) !== 200) {
            return null;
        }
        $html = (string) wp_remote_retrieve_body($res);
        if ($html !== '') {
            set_transient($cache_key, $html, 60);
        }
        return $html;
    }
}
