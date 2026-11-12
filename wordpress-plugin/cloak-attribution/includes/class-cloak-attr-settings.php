<?php

if (!defined('ABSPATH')) {
    exit;
}

class Cloak_Attr_Settings
{
    public const OPTION_KEY = 'cloak_attr_settings';
    public const V1_OPTION_KEY = 'cloak_api_settings';

    // —— 同域输出（200，地址不变）——
    public const REVIEW_API_HTML = 'api_html';     // 用判定接口返回的平台模板 HTML（仅审核）
    public const REVIEW_WP_PAGE = 'wp_page';       // 在当前地址渲染站内某个页面
    public const REVIEW_HTML_FILE = 'html_file';   // 在当前地址输出服务器上的 HTML 文件
    public const REVIEW_PROXY_URL = 'proxy_url';   // 服务端拉取外部地址内容后在当前地址输出（不改写域名）
    public const REVIEW_DEEP_PROXY = 'deep_proxy'; // 深度代理：回源审核站并把其域名改写成本站域名后在当前地址输出
    // —— 真实页专用 ——
    public const MODE_CURRENT = 'current';         // 继续渲染访客打开的当前页面（仅真实页）
    // —— 兼容 V1 的跳转（302，地址会变）——
    public const MODE_WP_PAGE_REDIRECT = 'wp_page_redirect'; // 302 到站内页面固定链接
    public const MODE_EXTERNAL_REDIRECT = 'external_redirect'; // 302 到外部 https 地址（白名单）
    public const MODE_HTML_REDIRECT = 'html_redirect'; // 302 到 HTML 文件公开 URL

    /** 审核页可选模式。 */
    public static function review_modes(): array
    {
        return [
            self::REVIEW_API_HTML,
            self::REVIEW_WP_PAGE,
            self::REVIEW_HTML_FILE,
            self::REVIEW_PROXY_URL,
            self::REVIEW_DEEP_PROXY,
            self::MODE_WP_PAGE_REDIRECT,
            self::MODE_EXTERNAL_REDIRECT,
            self::MODE_HTML_REDIRECT,
        ];
    }

    /** 真实页可选模式（无 api_html，多 current）。 */
    public static function real_modes(): array
    {
        return [
            self::MODE_CURRENT,
            self::REVIEW_WP_PAGE,
            self::REVIEW_HTML_FILE,
            self::REVIEW_PROXY_URL,
            self::REVIEW_DEEP_PROXY,
            self::MODE_WP_PAGE_REDIRECT,
            self::MODE_EXTERNAL_REDIRECT,
            self::MODE_HTML_REDIRECT,
        ];
    }

    public static function defaults(): array
    {
        return [
            'enabled' => false,
            'api_base' => '',            // 网关地址，如 https://entry.example.com
            'secret' => '',              // sk_... 完整密钥
            'timeout' => 3,
            'fail_mode' => 'review',     // review | real
            'cache_ttl' => 30,           // 同一访客 + 路径判定缓存秒数，0 关闭
            'skip_logged_in' => true,
            'exclude_paths' => "/wp-admin/*\n/wp-login.php\n/wp-json/*\n/wp-cron.php\n/xmlrpc.php",
            'review_mode' => self::REVIEW_API_HTML,
            'review_page_id' => 0,
            'review_html_path' => '',
            'review_proxy_url' => '',
            'review_deep_origin' => '',   // deep_proxy 审核站地址
            'review_deep_host' => '',     // deep_proxy 回源 Host（可选）
            'proxy_cache_ttl' => 86400,  // 深度代理缓存秒数，由设置页的天数换算，0 关闭
            'review_external_url' => '',  // external_redirect 目标
            'review_html_url' => '',      // html_redirect 目标
            'real_mode' => self::MODE_CURRENT,
            'real_page_id' => 0,
            'real_html_path' => '',
            'real_proxy_url' => '',
            'real_deep_origin' => '',
            'real_deep_host' => '',
            'real_external_url' => '',
            'real_html_url' => '',
            'external_allowed_hosts' => '', // external_redirect 允许的主机，每行一个
            'inject_banner' => true,
            'woo_enabled' => true,
            'woo_scope' => 'all',        // all | shop_only（商店/购物车/结账/商品） | products
            'woo_product_ids' => '',
            'woo_category_slugs' => '',
        ];
    }

    public static function get(): array
    {
        $saved = get_option(self::OPTION_KEY, []);
        if (!is_array($saved)) {
            $saved = [];
        }
        $settings = array_merge(self::defaults(), $saved);
        if (defined('CLOAK_SECRET') && CLOAK_SECRET) {
            $settings['secret'] = CLOAK_SECRET;
        }
        if (defined('CLOAK_API_BASE') && CLOAK_API_BASE) {
            $settings['api_base'] = CLOAK_API_BASE;
        }
        return $settings;
    }

    public static function sanitize(array $input, array $current = []): array
    {
        $current = $current ?: self::get();
        $secret = $current['secret'];
        if (isset($input['secret']) && trim((string) $input['secret']) !== '') {
            $secret = sanitize_text_field(trim((string) $input['secret']));
        }
        $review_mode = sanitize_key($input['review_mode'] ?? $current['review_mode']);
        if (!in_array($review_mode, self::review_modes(), true)) {
            $review_mode = self::REVIEW_API_HTML;
        }
        $real_mode = sanitize_key($input['real_mode'] ?? $current['real_mode']);
        if (!in_array($real_mode, self::real_modes(), true)) {
            $real_mode = self::MODE_CURRENT;
        }
        $fail_mode = sanitize_key($input['fail_mode'] ?? $current['fail_mode']);
        if (!in_array($fail_mode, ['review', 'real'], true)) {
            $fail_mode = 'review';
        }
        $woo_scope = sanitize_key($input['woo_scope'] ?? $current['woo_scope']);
        if (!in_array($woo_scope, ['all', 'shop_only', 'products'], true)) {
            $woo_scope = 'all';
        }
        return [
            'enabled' => !empty($input['enabled']),
            'api_base' => untrailingslashit(esc_url_raw(trim((string) ($input['api_base'] ?? $current['api_base'])))),
            'secret' => $secret,
            'timeout' => max(1, min(10, (int) ($input['timeout'] ?? $current['timeout']))),
            'fail_mode' => $fail_mode,
            'cache_ttl' => max(0, min(600, (int) ($input['cache_ttl'] ?? $current['cache_ttl']))),
            'skip_logged_in' => !empty($input['skip_logged_in']),
            'exclude_paths' => self::sanitize_lines($input['exclude_paths'] ?? $current['exclude_paths'], true),
            'review_mode' => $review_mode,
            'review_page_id' => absint($input['review_page_id'] ?? $current['review_page_id']),
            'review_html_path' => sanitize_text_field(str_replace('\\', '/', trim((string) ($input['review_html_path'] ?? $current['review_html_path'])))),
            'review_proxy_url' => self::sanitize_http_url((string) ($input['review_proxy_url'] ?? $current['review_proxy_url'])),
            'review_deep_origin' => self::sanitize_http_url((string) ($input['review_deep_origin'] ?? $current['review_deep_origin'])),
            'review_deep_host' => sanitize_text_field(trim((string) ($input['review_deep_host'] ?? $current['review_deep_host']))),
            'proxy_cache_ttl' => self::cache_ttl_from_input($input, $current),
            'review_external_url' => esc_url_raw(trim((string) ($input['review_external_url'] ?? $current['review_external_url']))),
            'review_html_url' => esc_url_raw(trim((string) ($input['review_html_url'] ?? $current['review_html_url']))),
            'real_mode' => $real_mode,
            'real_page_id' => absint($input['real_page_id'] ?? $current['real_page_id']),
            'real_html_path' => sanitize_text_field(str_replace('\\', '/', trim((string) ($input['real_html_path'] ?? $current['real_html_path'])))),
            'real_proxy_url' => self::sanitize_http_url((string) ($input['real_proxy_url'] ?? $current['real_proxy_url'])),
            'real_deep_origin' => self::sanitize_http_url((string) ($input['real_deep_origin'] ?? $current['real_deep_origin'])),
            'real_deep_host' => sanitize_text_field(trim((string) ($input['real_deep_host'] ?? $current['real_deep_host']))),
            'real_external_url' => esc_url_raw(trim((string) ($input['real_external_url'] ?? $current['real_external_url']))),
            'real_html_url' => esc_url_raw(trim((string) ($input['real_html_url'] ?? $current['real_html_url']))),
            'external_allowed_hosts' => self::sanitize_hosts($input['external_allowed_hosts'] ?? $current['external_allowed_hosts']),
            'inject_banner' => !empty($input['inject_banner']),
            'woo_enabled' => !empty($input['woo_enabled']),
            'woo_scope' => $woo_scope,
            'woo_product_ids' => self::sanitize_lines($input['woo_product_ids'] ?? $current['woo_product_ids'], false),
            'woo_category_slugs' => self::sanitize_lines($input['woo_category_slugs'] ?? $current['woo_category_slugs'], false),
        ];
    }

    /** 从 v1 插件配置迁移（只在 v2 尚无配置时执行）。 */
    public static function migrate_from_v1(): void
    {
        if (get_option(self::OPTION_KEY)) {
            return;
        }
        $v1 = get_option(self::V1_OPTION_KEY);
        if (!is_array($v1) || empty($v1)) {
            return;
        }
        $defaults = self::defaults();
        $api_base = '';
        if (!empty($v1['api_url'])) {
            $parts = wp_parse_url($v1['api_url']);
            if (!empty($parts['host'])) {
                $api_base = ($parts['scheme'] ?? 'https') . '://' . $parts['host'] . (isset($parts['port']) ? ':' . $parts['port'] : '');
            }
        }
        $exclude = (string) ($v1['exclude_paths'] ?? '');
        $merged = array_unique(array_filter(array_map('trim', array_merge(
            preg_split('/\r\n|\r|\n/', $defaults['exclude_paths']),
            preg_split('/\r\n|\r|\n/', $exclude)
        ))));
        // 审核页与真实页逐一映射，保留 V1 的 302 行为
        $sides = array_merge(
            self::map_v1_side($v1, 'review'),
            self::map_v1_side($v1, 'real')
        );
        $settings = array_merge($defaults, $sides, [
            'enabled' => !empty($v1['enabled']),
            'api_base' => $api_base,
            // v1 的 ck_ Key 不能直接用，须换成站点 sk_ 密钥；若已是 sk_ 则沿用
            'secret' => (strpos((string) ($v1['api_token'] ?? ''), 'sk_') === 0) ? (string) $v1['api_token'] : '',
            'fail_mode' => in_array($v1['fail_mode'] ?? '', ['review', 'real'], true) ? $v1['fail_mode'] : 'review',
            'exclude_paths' => implode("\n", $merged),
            'external_allowed_hosts' => self::sanitize_hosts((string) ($v1['external_allowed_hosts'] ?? '')),
        ]);
        update_option(self::OPTION_KEY, $settings);
        update_option('cloak_attr_migrated_from_v1', [
            'at' => current_time('mysql'),
            'needs_secret' => $settings['secret'] === '',
        ]);
    }

    /** 把 V1 单侧（review/real）配置映射为 V2 字段，保留 302。 */
    private static function map_v1_side(array $v1, string $side): array
    {
        $mode = (string) ($v1[$side . '_mode'] ?? '');
        switch ($mode) {
            case 'wp_page': // V1 为 302 跳转
                return [
                    $side . '_mode' => self::MODE_WP_PAGE_REDIRECT,
                    $side . '_page_id' => (int) ($v1[$side . '_page_id'] ?? 0),
                ];
            case 'external_url':
                return [
                    $side . '_mode' => self::MODE_EXTERNAL_REDIRECT,
                    $side . '_external_url' => (string) ($v1[$side . '_external_url'] ?? ''),
                ];
            case 'html_redirect':
                return [
                    $side . '_mode' => self::MODE_HTML_REDIRECT,
                    $side . '_html_url' => (string) ($v1[$side . '_html_url'] ?? ''),
                ];
            case 'html_output': // V1 唯一的同域 200 输出
                return [
                    $side . '_mode' => self::REVIEW_HTML_FILE,
                    $side . '_html_path' => (string) ($v1[$side . '_html_path'] ?? ''),
                ];
            default:
                return [$side . '_mode' => $side === 'real' ? self::MODE_CURRENT : self::REVIEW_API_HTML];
        }
    }

    /** 设置页按天填写，这里换成秒。0 天表示关闭，最多 30 天。 */
    public static function cache_ttl_from_input(array $input, array $current): int
    {
        if (!array_key_exists('proxy_cache_days', $input)) {
            return max(0, (int) ($current['proxy_cache_ttl'] ?? 86400));
        }
        $days = max(0, min(30, (int) $input['proxy_cache_days']));
        return $days * 86400;
    }

    /** 把已保存的秒数还原成设置页上的天数。不足 1 天但未关闭时显示为 1 天。 */
    public static function cache_days(array $settings): int
    {
        $seconds = (int) ($settings['proxy_cache_ttl'] ?? 0);
        if ($seconds <= 0) {
            return 0;
        }
        return (int) max(1, (int) round($seconds / 86400));
    }

    /**
     * 保存 http(s) 地址。浏览器的 url 输入框会拦下不带协议的内容，导致整张表单保存失败；
     * 这里接受 toptime.cc 这种写法并补上 https。不借用 esc_url_raw，避免被其它插件的链接过滤清掉。
     */
    public static function sanitize_http_url(string $value): string
    {
        $value = trim($value);
        if ($value === '') {
            return '';
        }
        if (!preg_match('#^[a-z][a-z0-9+.-]*://#i', $value)) {
            $value = 'https://' . ltrim($value, '/');
        }
        $parts = wp_parse_url($value);
        $host = strtolower((string) ($parts['host'] ?? ''));
        if ($host === '' || !preg_match('/^[a-z0-9.-]+$/', $host)) {
            return '';
        }
        $scheme = strtolower((string) ($parts['scheme'] ?? ''));
        if ($scheme !== 'http' && $scheme !== 'https') {
            return '';
        }
        $port = isset($parts['port']) ? ':' . (int) $parts['port'] : '';
        $path = (string) ($parts['path'] ?? '');
        $query = isset($parts['query']) ? '?' . $parts['query'] : '';
        return $scheme . '://' . $host . $port . $path . $query;
    }

    /** 外链白名单：每行一个主机或 URL，统一归一化为小写主机。 */
    public static function sanitize_hosts($value): string
    {
        $out = [];
        foreach (preg_split('/\r\n|\r|\n|,/', (string) $value) as $line) {
            $line = strtolower(trim($line));
            if ($line === '') {
                continue;
            }
            if (strpos($line, '://') !== false) {
                $line = (string) wp_parse_url($line, PHP_URL_HOST);
            }
            if ($line !== '') {
                $out[] = $line;
            }
        }
        return implode("\n", array_unique($out));
    }

    public static function allowed_external_hosts(array $settings): array
    {
        return array_values(array_filter(array_map('trim', preg_split('/\r\n|\r|\n/', (string) ($settings['external_allowed_hosts'] ?? '')))));
    }

    public static function exclude_patterns(array $settings): array
    {
        return array_values(array_filter(array_map('trim', preg_split('/\r\n|\r|\n/', (string) $settings['exclude_paths']))));
    }

    public static function path_matches(string $path, string $pattern): bool
    {
        $pattern = trim($pattern);
        if ($pattern === '') {
            return false;
        }
        if ($pattern[0] !== '/') {
            $pattern = '/' . $pattern;
        }
        if (substr($pattern, -2) === '/*') {
            $prefix = rtrim(substr($pattern, 0, -1), '/');
            return $prefix === '' || $path === $prefix || strpos($path, $prefix . '/') === 0;
        }
        if (substr($pattern, -1) === '*') {
            return strpos($path, rtrim($pattern, '*')) === 0;
        }
        return untrailingslashit($path) === untrailingslashit($pattern);
    }

    private static function sanitize_lines($value, bool $as_path): string
    {
        $out = [];
        foreach (preg_split('/\r\n|\r|\n|,/', (string) $value) as $line) {
            $line = trim($line);
            if ($line === '') {
                continue;
            }
            if ($as_path && $line[0] !== '/') {
                $line = '/' . $line;
            }
            $out[] = sanitize_text_field($line);
        }
        return implode("\n", array_unique($out));
    }
}
