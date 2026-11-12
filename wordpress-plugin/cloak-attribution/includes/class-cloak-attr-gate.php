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
        try {
            $this->dispatch();
        } catch (Throwable $e) {
            error_log('cloak-attribution: ' . $e->getMessage());
            if (is_array($this->decision)) {
                try {
                    $this->render('review', Cloak_Attr_Settings::get(), $this->decision);
                    return;
                } catch (Throwable $again) {
                    error_log('cloak-attribution render: ' . $again->getMessage());
                    $e = $again;
                }
            }
            if (!headers_sent()) {
                status_header(200);
                nocache_headers();
                header('Content-Type: text/html; charset=utf-8');
                header('X-Cloak-Content: review');
            }
            echo '<!doctype html><html><head><meta charset="utf-8"><title>' . esc_html(get_bloginfo('name')) . '</title></head><body><h1>' . esc_html(get_bloginfo('name')) . '</h1><!-- cloak: ' . esc_html($e->getMessage()) . ' --></body></html>';
            exit;
        }
    }

    private function dispatch(): void
    {
        $settings = Cloak_Attr_Settings::get();
        if (empty($settings['enabled'])) {
            return;
        }
        // 审核页把 CSS/JS/图片改写到了本站域名。这些后缀会被判定成 bypass，
        // 若交给本站 WordPress，主题文件不存在就会 404，页面只剩裸链接。
        // 刚看过审核页的访客，静态资源改从审核站回源。
        if ($this->proxy_review_asset($settings)) {
            return;
        }
        if (!$this->should_run($settings)) {
            return;
        }
        // 被判定的请求按访客区分，绝不能进整页缓存。告诉 W3TC / WP Super Cache /
        // LiteSpeed / WP Rocket 等不要缓存本次响应，否则第一个访客的内容会发给所有人。
        $this->prevent_page_cache();
        $data = Cloak_Attr_Client::decide($settings);
        if ($data === null) {
            $data = ['content' => $settings['fail_mode'] === 'real' ? 'real' : 'review', 'failed' => true];
        }
        $this->decision = $data;

        if (!empty($data['cookie']) && !empty($data['visitorToken'])) {
            $this->write_cookie($data);
        }

        $content = (string) ($data['content'] ?? 'review');
        if ($content === 'real' || $content === 'bypass') {
            if (!$this->is_asset_request()) {
                $this->mark_review_assets(false);
            }
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
        if (($settings['review_mode'] ?? '') === Cloak_Attr_Settings::REVIEW_DEEP_PROXY) {
            $this->mark_review_assets(true);
        }
        // 审核源是整站时，当前地址（含 CSS/JS/图片）按路径回源。
        // 只拉首页再套到所有地址上，样式会变成一段 HTML，浏览器直接丢掉。
        if ($this->proxy_matched_review($settings, $data)) {
            return;
        }
        $this->render('review', $settings, $data);
    }

    /**
     * 审核内容来自整站源站时，按访客当前路径回源，并沿用源站的 Content-Type。
     * 深度代理模式，或判定结果给出的就是站点根地址时，页面和静态资源都走这里。
     */
    private function proxy_matched_review(array $settings, array $data): bool
    {
        if (!$this->should_proxy_current_path($settings, $data)) {
            return false;
        }
        $origin = $this->review_origin($settings, $data);
        if ($origin === '' || ($this->is_asset_request() && $this->local_asset_exists())) {
            return false;
        }
        $this->prevent_page_cache();
        if (Cloak_Attr_Deep_Proxy::serve($origin, (string) ($settings['review_deep_host'] ?? ''), Cloak_Attr_Deep_Proxy::cache_from_settings($settings), $this->browser_sdk($data))) {
            exit;
        }
        if ($this->is_asset_request() && !headers_sent()) {
            status_header(404);
            exit;
        }
        return false;
    }

    private function should_proxy_current_path(array $settings, array $data): bool
    {
        $mode = (string) ($settings['review_mode'] ?? '');
        if ($mode === Cloak_Attr_Settings::REVIEW_DEEP_PROXY) {
            return trim((string) ($settings['review_deep_origin'] ?? '')) !== '';
        }
        // 只有「使用平台配置的审核模板」才跟随后台源站。
        // 没选中的「拉取外部地址」（例如曾经填过的 baidu.com）不能参与，否则会把访客跳走。
        if ($mode !== Cloak_Attr_Settings::REVIEW_API_HTML) {
            return false;
        }
        $hint = $this->hint_url($data);
        if ($hint === '') {
            return false;
        }
        if ($this->is_site_root($hint)) {
            return true;
        }
        return $this->is_asset_request();
    }

    /** 当前输出方式对应的审核站根地址。未选中的输入框一律不用。 */
    private function review_origin(array $settings, array $data): string
    {
        $mode = (string) ($settings['review_mode'] ?? '');
        if ($mode === Cloak_Attr_Settings::REVIEW_DEEP_PROXY) {
            return $this->origin_base((string) ($settings['review_deep_origin'] ?? ''));
        }
        if ($mode === Cloak_Attr_Settings::REVIEW_PROXY_URL) {
            return $this->origin_base((string) ($settings['review_proxy_url'] ?? ''));
        }
        if ($mode === Cloak_Attr_Settings::REVIEW_API_HTML) {
            return $this->origin_base($this->hint_url($data));
        }
        return '';
    }

    private function hint_url(array $data): string
    {
        $hint = is_array($data['renderHint'] ?? null) ? $data['renderHint'] : [];
        if (($hint['type'] ?? '') === 'url') {
            return trim((string) ($hint['url'] ?? ''));
        }
        return '';
    }

    private function origin_base(string $url): string
    {
        $parts = wp_parse_url($url);
        if (empty($parts['host'])) {
            return '';
        }
        $scheme = ($parts['scheme'] ?? 'https') === 'http' ? 'http' : 'https';
        $port = isset($parts['port']) ? ':' . $parts['port'] : '';
        return $scheme . '://' . $parts['host'] . $port;
    }

    private function is_site_root(string $url): bool
    {
        $path = (string) (wp_parse_url($url, PHP_URL_PATH) ?? '');
        return $path === '' || $path === '/';
    }

    /** 审核态下的样式、脚本、图片、字体，回源审核站，不走本站文件。 */
    private function proxy_review_asset(array $settings): bool
    {
        if (($settings['review_mode'] ?? '') !== Cloak_Attr_Settings::REVIEW_DEEP_PROXY) {
            return false;
        }
        $origin = (string) ($settings['review_deep_origin'] ?? '');
        if ($origin === '' || !$this->is_asset_request() || $this->local_asset_exists()) {
            return false;
        }
        $this->prevent_page_cache();
        $served = Cloak_Attr_Deep_Proxy::serve($origin, (string) ($settings['review_deep_host'] ?? ''), Cloak_Attr_Deep_Proxy::cache_from_settings($settings));
        if (!$served && !headers_sent()) {
            status_header(404);
        }
        exit;
    }

    /** 本站磁盘上已经有的文件交给服务器，缺的才回源审核站。 */
    private function local_asset_exists(): bool
    {
        $path = (string) strtok((string) ($_SERVER['REQUEST_URI'] ?? '/'), '?');
        if ($path === '' || strpos($path, '..') !== false) {
            return false;
        }
        return is_file(ABSPATH . ltrim($path, '/'));
    }

    private function is_asset_request(): bool
    {
        $path = strtolower((string) strtok((string) ($_SERVER['REQUEST_URI'] ?? '/'), '?'));
        foreach (['.css', '.js', '.mjs', '.png', '.jpg', '.jpeg', '.gif', '.webp', '.svg', '.ico', '.woff', '.woff2', '.ttf', '.eot', '.otf', '.map'] as $suffix) {
            if (substr($path, -strlen($suffix)) === $suffix) {
                return true;
            }
        }
        return false;
    }

    /** 标记这个访客当前在看审核页，后续静态资源才知道要回源审核站。 */
    private function mark_review_assets(bool $on): void
    {
        if (headers_sent()) {
            return;
        }
        setcookie('_ck_rv', $on ? '1' : '', [
            'expires' => $on ? time() + 86400 : time() - 3600,
            'path' => '/',
            'secure' => is_ssl(),
            'httponly' => true,
            'samesite' => 'Lax',
        ]);
    }

    /** 阻止整页缓存插件缓存被判定的响应（它们按常量 / 响应头跳过）。 */
    private function prevent_page_cache(): void
    {
        foreach (['DONOTCACHEPAGE', 'DONOTCACHEOBJECT', 'DONOTCACHEDB'] as $c) {
            if (!defined($c)) {
                define($c, true);
            }
        }
        // LiteSpeed Cache：主动发一条不缓存指令
        if (!headers_sent()) {
            header('X-LiteSpeed-Cache-Control: no-cache');
        }
        if (function_exists('do_action')) {
            do_action('litespeed_control_set_nocache', 'cloak attribution per-visitor');
        }
    }

    /** 发送响应头前先确认还没开始输出，避免主题提前输出时报 warning。 */
    private function send_header(string $header): void
    {
        if (!headers_sent()) {
            header($header);
        }
    }

    /** 引导条：放行且在内置浏览器内时，由接口给出 handoffUrl。出错不影响页面。 */
    public function footer(): void
    {
        try {
            $this->render_footer();
        } catch (Throwable $e) {
            error_log('cloak-attribution footer: ' . $e->getMessage());
        }
    }

    private function render_footer(): void
    {
        if (!is_array($this->decision)) {
            return;
        }
        echo $this->browser_sdk($this->decision); // phpcs:ignore WordPress.Security.EscapeOutput
    }

    /**
     * 只把本次要打开的地址交给客户站点上的 JS SDK。弹框和跳转由 sdk.js 绘制。
     * 已放行时用接力地址，审核页用当前地址。
     */
    private function browser_sdk(array $data): string
    {
        $settings = Cloak_Attr_Settings::get();
        if (empty($settings['inject_banner']) || !empty($_COOKIE['_ck_hb'])) {
            return '';
        }
        if (isset($data['handoffBanner']) && empty($data['handoffBanner']['enabled'])) {
            return '';
        }
        $in = is_array($data['inAppBrowser'] ?? null) ? $data['inAppBrowser'] : [];
        if (empty($in['detected'])) {
            return '';
        }
        $base = untrailingslashit((string) ($settings['api_base'] ?? ''));
        if ($base === '') {
            return '';
        }
        $target = trim((string) ($data['handoffUrl'] ?? ''));
        if ($target === '') {
            $host = (string) ($_SERVER['HTTP_HOST'] ?? '');
            if ($host === '') {
                return '';
            }
            $target = (is_ssl() ? 'https://' : 'http://') . $host . (string) ($_SERVER['REQUEST_URI'] ?? '/');
        }
        $payload = wp_json_encode([
            'url' => $target,
            'os' => (string) ($in['os'] ?? ''),
            'text' => (string) ($data['handoffBanner']['text'] ?? '在系统浏览器中打开'),
            'enabled' => true,
        ]);
        if (!is_string($payload)) {
            return '';
        }
        $src = esc_url($base . '/_cloak/sdk.js');
        return '<script>window.__CLOAK_HANDOFF__=' . $payload . ';</script>'
            . '<script src="' . $src . '" data-banner="preset" async></script>';
    }

    private function should_run(array $settings): bool
    {
        if (is_admin() || wp_doing_ajax() || wp_doing_cron() || (function_exists('wp_doing_autosave') && wp_doing_autosave()) || is_feed() || is_robots()) {
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
                $this->send_header('X-Cloak-Content: ' . $side);
                return;

            case Cloak_Attr_Settings::MODE_WP_PAGE_REDIRECT:
                $this->redirect_wp_page((int) ($settings[$side . '_page_id'] ?? 0), $side);
                $this->send_header('X-Cloak-Content: ' . $side); // 已在目标页：继续渲染
                return;

            case Cloak_Attr_Settings::MODE_EXTERNAL_REDIRECT:
                $this->redirect_external((string) ($settings[$side . '_external_url'] ?? ''), $settings, $side);
                $this->send_header('X-Cloak-Content: ' . $side);
                return;

            case Cloak_Attr_Settings::MODE_HTML_REDIRECT:
                $this->redirect_html((string) ($settings[$side . '_html_url'] ?? ''), $side);
                $this->send_header('X-Cloak-Content: ' . $side);
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

            case Cloak_Attr_Settings::REVIEW_DEEP_PROXY:
                // 深度代理：回源审核站并把其域名改写成本站域名，在当前地址输出。
                $origin = (string) ($settings[$side . '_deep_origin'] ?? '');
                $host = (string) ($settings[$side . '_deep_host'] ?? '');
                if ($origin !== '' && Cloak_Attr_Deep_Proxy::serve($origin, $host, Cloak_Attr_Deep_Proxy::cache_from_settings($settings), $this->browser_sdk($data))) {
                    exit;
                }
                break;
        }

        // 走到这里说明上面的模式未能输出（配置缺失或模式为 api_html）
        if ($side === 'real') {
            // 真实页兜底：继续渲染访客当前页面，绝不回落到审核模板
            $this->send_header('X-Cloak-Content: real');
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
        if (headers_sent()) {
            return;
        }
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
        if (!$permalink || headers_sent()) {
            return;
        }
        $this->send_header('X-Cloak-Content: ' . $side);
        wp_safe_redirect($permalink, 302);
        exit;
    }

    /** 302 到外部 https 地址，须通过白名单校验。 */
    private function redirect_external(string $url, array $settings, string $side): void
    {
        $url = trim($url);
        if ($url === '' || headers_sent() || !$this->is_allowed_external_url($url, $settings) || $this->is_current_url($url)) {
            return;
        }
        $this->send_header('X-Cloak-Content: ' . $side);
        wp_redirect(esc_url_raw($url), 302); // phpcs:ignore WordPress.Security.SafeRedirect
        exit;
    }

    /** 302 到 HTML 文件公开 URL。 */
    private function redirect_html(string $url, string $side): void
    {
        $url = trim($url);
        if ($url === '' || headers_sent() || $this->is_current_url($url)) {
            return;
        }
        $path = (string) (wp_parse_url($url, PHP_URL_PATH) ?? '');
        if ($path !== '' && strpos((string) ($_SERVER['REQUEST_URI'] ?? ''), $path) !== false) {
            return;
        }
        $this->send_header('X-Cloak-Content: ' . $side);
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
            return $this->rewrite_fetched($cached, $url, '');
        }
        $res = wp_remote_get($url, [
            'timeout' => 5,
            'limit_response_size' => 1048576,
            'headers' => ['Accept' => 'text/html', 'Accept-Encoding' => 'identity'],
        ]);
        if (is_wp_error($res) || (int) wp_remote_retrieve_response_code($res) !== 200) {
            return null;
        }
        $html = \Cloak\Sdk\Rewriter::inflate(
            (string) wp_remote_retrieve_body($res),
            Cloak_Attr_Deep_Proxy::header_value($res, 'content-encoding')
        );
        // 只缓存较小的页面。整站首页（如搜索引擎）塞进 wp_options 会撑爆数据包，PHP 8 会直接致命错误。
        if ($html !== '' && strlen($html) <= 200000) {
            set_transient($cache_key, $html, 60);
        }
        return $this->rewrite_fetched($html, $url, Cloak_Attr_Deep_Proxy::header_value($res, 'content-type'));
    }

    /** 拉取到的审核 HTML 在输出前改写域名，缓存里的旧内容也走这一步。 */
    private function rewrite_fetched(string $html, string $url, string $contentType): string
    {
        $host = (string) (wp_parse_url($url, PHP_URL_HOST) ?? '');
        $pub = (string) ($_SERVER['HTTP_HOST'] ?? '');
        if ($host === '' || $pub === '') {
            return $html;
        }
        return Cloak_Attr_Deep_Proxy::rewrite_document(
            $html,
            [\Cloak\Sdk\Rewriter::stripPort($host)],
            $pub,
            is_ssl() ? 'https' : 'http',
            $contentType
        );
    }

    /** 空域名不要传给 setcookie，PHP 8 会抛 ValueError，整站变成关键错误。 */
    private function write_cookie(array $data): void
    {
        if (headers_sent() || !is_array($data['cookie'] ?? null)) {
            return;
        }
        $c = $data['cookie'];
        $name = (string) ($c['name'] ?? Cloak_Attr_Client::COOKIE_DEFAULT);
        if ($name === '' || preg_match('/[=,; \\t\\r\\n]/', $name)) {
            return;
        }
        $opts = [
            'expires' => time() + max(60, (int) ($c['maxAge'] ?? 2592000)),
            'path' => '/',
            'secure' => !empty($c['secure']) && is_ssl(),
            'httponly' => true,
            'samesite' => 'Lax',
        ];
        $domain = trim((string) ($c['domain'] ?? ''));
        if ($domain !== '') {
            $opts['domain'] = $domain;
        }
        try {
            setcookie($name, (string) $data['visitorToken'], $opts);
        } catch (Throwable $e) {
            error_log('cloak-attribution setcookie: ' . $e->getMessage());
        }
    }
}
