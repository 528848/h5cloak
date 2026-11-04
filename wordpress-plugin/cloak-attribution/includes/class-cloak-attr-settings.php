<?php

if (!defined('ABSPATH')) {
    exit;
}

class Cloak_Attr_Settings
{
    public const OPTION_KEY = 'cloak_attr_settings';
    public const V1_OPTION_KEY = 'cloak_api_settings';

    public const REVIEW_API_HTML = 'api_html';     // 用判定接口返回的平台模板 HTML
    public const REVIEW_WP_PAGE = 'wp_page';       // 渲染站内某个页面（地址不变）
    public const REVIEW_HTML_FILE = 'html_file';   // 输出服务器上的 HTML 文件
    public const REVIEW_PROXY_URL = 'proxy_url';   // 服务端拉取审核源站内容后输出

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
        if (!in_array($review_mode, [self::REVIEW_API_HTML, self::REVIEW_WP_PAGE, self::REVIEW_HTML_FILE, self::REVIEW_PROXY_URL], true)) {
            $review_mode = self::REVIEW_API_HTML;
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
            'review_proxy_url' => esc_url_raw(trim((string) ($input['review_proxy_url'] ?? $current['review_proxy_url']))),
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
        $review_mode = self::REVIEW_API_HTML;
        $page_id = 0;
        $html_path = '';
        $proxy_url = '';
        switch ($v1['review_mode'] ?? '') {
            case 'wp_page':
                $review_mode = self::REVIEW_WP_PAGE;
                $page_id = (int) ($v1['review_page_id'] ?? 0);
                break;
            case 'html_output':
            case 'html_redirect':
                if (!empty($v1['review_html_path'])) {
                    $review_mode = self::REVIEW_HTML_FILE;
                    $html_path = (string) $v1['review_html_path'];
                }
                break;
            case 'external_url':
                if (!empty($v1['review_external_url'])) {
                    $review_mode = self::REVIEW_PROXY_URL;
                    $proxy_url = (string) $v1['review_external_url'];
                }
                break;
        }
        $exclude = (string) ($v1['exclude_paths'] ?? '');
        $merged = array_unique(array_filter(array_map('trim', array_merge(
            preg_split('/\r\n|\r|\n/', $defaults['exclude_paths']),
            preg_split('/\r\n|\r|\n/', $exclude)
        ))));
        $settings = array_merge($defaults, [
            'enabled' => !empty($v1['enabled']),
            'api_base' => $api_base,
            // v1 的 ck_ Key 不能直接用，须换成站点 sk_ 密钥；若已是 sk_ 则沿用
            'secret' => (strpos((string) ($v1['api_token'] ?? ''), 'sk_') === 0) ? (string) $v1['api_token'] : '',
            'fail_mode' => in_array($v1['fail_mode'] ?? '', ['review', 'real'], true) ? $v1['fail_mode'] : 'review',
            'exclude_paths' => implode("\n", $merged),
            'review_mode' => $review_mode,
            'review_page_id' => $page_id,
            'review_html_path' => $html_path,
            'review_proxy_url' => $proxy_url,
        ]);
        update_option(self::OPTION_KEY, $settings);
        update_option('cloak_attr_migrated_from_v1', [
            'at' => current_time('mysql'),
            'needs_secret' => $settings['secret'] === '',
        ]);
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
