<?php

if (!defined('ABSPATH')) {
    exit;
}

class Cloak_Settings
{
    public const OPTION_KEY = 'cloak_api_settings';

    public const MODE_WP_PAGE = 'wp_page';
    public const MODE_EXTERNAL = 'external_url';
    public const MODE_HTML_REDIRECT = 'html_redirect';
    public const MODE_HTML_OUTPUT = 'html_output';

    /** @deprecated Use MODE_WP_PAGE */
    public const REVIEW_MODE_WP_PAGE = self::MODE_WP_PAGE;

    /** @deprecated Use MODE_EXTERNAL */
    public const REVIEW_MODE_EXTERNAL = self::MODE_EXTERNAL;

    /** @deprecated Use MODE_HTML_REDIRECT */
    public const REVIEW_MODE_HTML_REDIRECT = self::MODE_HTML_REDIRECT;

    /** @deprecated Use MODE_HTML_OUTPUT */
    public const REVIEW_MODE_HTML_OUTPUT = self::MODE_HTML_OUTPUT;

    public static function defaults(): array
    {
        return [
            'enabled' => false,
            'api_url' => '',
            'api_token' => '',
            'review_mode' => self::MODE_WP_PAGE,
            'review_page_id' => 0,
            'review_external_url' => '',
            'review_html_url' => '',
            'review_html_path' => '',
            'real_mode' => self::MODE_WP_PAGE,
            'real_page_id' => 0,
            'real_external_url' => '',
            'real_html_url' => '',
            'real_html_path' => '',
            'external_allowed_hosts' => '',
            'exclude_paths' => '',
            'fail_mode' => 'review',
            'scope' => 'entire_site',
        ];
    }

    public static function get(): array
    {
        $saved = get_option(self::OPTION_KEY, []);
        if (!is_array($saved)) {
            $saved = [];
        }

        $settings = array_merge(self::defaults(), $saved);

        if (defined('CLOAK_API_TOKEN') && CLOAK_API_TOKEN) {
            $settings['api_token'] = CLOAK_API_TOKEN;
        }

        if (defined('CLOAK_API_URL') && CLOAK_API_URL) {
            $settings['api_url'] = CLOAK_API_URL;
        }

        return $settings;
    }

    public static function update(array $input): array
    {
        $current = self::get();
        $clean = self::sanitize($input, $current);
        update_option(self::OPTION_KEY, $clean);

        return self::get();
    }

    public static function sanitize(array $input, array $current = []): array
    {
        $current = $current ?: self::get();
        $modes = self::mode_values();

        $review_mode = sanitize_key($input['review_mode'] ?? $current['review_mode']);
        if (!in_array($review_mode, $modes, true)) {
            $review_mode = self::MODE_WP_PAGE;
        }

        $real_mode = sanitize_key($input['real_mode'] ?? $current['real_mode']);
        if (!in_array($real_mode, $modes, true)) {
            $real_mode = self::MODE_WP_PAGE;
        }

        $fail_mode = sanitize_key($input['fail_mode'] ?? $current['fail_mode']);
        if (!in_array($fail_mode, ['review', 'real'], true)) {
            $fail_mode = 'review';
        }

        $scope = sanitize_key($input['scope'] ?? $current['scope']);
        if (!in_array($scope, ['entire_site', 'homepage_only'], true)) {
            $scope = 'entire_site';
        }

        $api_token = $current['api_token'];
        if (isset($input['api_token']) && $input['api_token'] !== '') {
            $api_token = sanitize_text_field($input['api_token']);
        }

        return [
            'enabled' => !empty($input['enabled']),
            'api_url' => esc_url_raw(trim($input['api_url'] ?? $current['api_url'])),
            'api_token' => $api_token,
            'review_mode' => $review_mode,
            'review_page_id' => absint($input['review_page_id'] ?? $current['review_page_id']),
            'review_external_url' => esc_url_raw(trim($input['review_external_url'] ?? $current['review_external_url'])),
            'review_html_url' => esc_url_raw(trim($input['review_html_url'] ?? $current['review_html_url'])),
            'review_html_path' => self::sanitize_path($input['review_html_path'] ?? $current['review_html_path']),
            'real_mode' => $real_mode,
            'real_page_id' => absint($input['real_page_id'] ?? $current['real_page_id']),
            'real_external_url' => esc_url_raw(trim($input['real_external_url'] ?? $current['real_external_url'])),
            'real_html_url' => esc_url_raw(trim($input['real_html_url'] ?? $current['real_html_url'])),
            'real_html_path' => self::sanitize_path($input['real_html_path'] ?? $current['real_html_path']),
            'external_allowed_hosts' => self::sanitize_hosts($input['external_allowed_hosts'] ?? $current['external_allowed_hosts']),
            'exclude_paths' => self::sanitize_exclude_paths($input['exclude_paths'] ?? $current['exclude_paths']),
            'fail_mode' => $fail_mode,
            'scope' => $scope,
        ];
    }

    public static function mode_values(): array
    {
        return [
            self::MODE_WP_PAGE,
            self::MODE_EXTERNAL,
            self::MODE_HTML_REDIRECT,
            self::MODE_HTML_OUTPUT,
        ];
    }

    public static function mode_labels(): array
    {
        return [
            self::MODE_WP_PAGE => __('WordPress 站内页面', 'cloak-api'),
            self::MODE_EXTERNAL => __('外部链接', 'cloak-api'),
            self::MODE_HTML_REDIRECT => __('服务器 HTML 文件（URL 跳转）', 'cloak-api'),
            self::MODE_HTML_OUTPUT => __('服务器 HTML 文件（PHP 直接输出）', 'cloak-api'),
        ];
    }

    public static function review_mode_labels(): array
    {
        return self::mode_labels();
    }

    public static function allowed_external_hosts(array $settings): array
    {
        $lines = preg_split('/\r\n|\r|\n/', (string) ($settings['external_allowed_hosts'] ?? ''));
        $hosts = [];

        foreach ($lines as $line) {
            $host = strtolower(trim($line));
            if ($host !== '') {
                $hosts[] = $host;
            }
        }

        foreach (['review_external_url', 'real_external_url'] as $key) {
            if (empty($settings[$key])) {
                continue;
            }
            $parsed = wp_parse_url($settings[$key], PHP_URL_HOST);
            if ($parsed) {
                $hosts[] = strtolower($parsed);
            }
        }

        return array_values(array_unique($hosts));
    }

    /**
     * @return string[]
     */
    public static function exclude_path_patterns(array $settings): array
    {
        $lines = preg_split('/\r\n|\r|\n/', (string) ($settings['exclude_paths'] ?? ''));
        $patterns = [];

        foreach ($lines as $line) {
            $line = trim($line);
            if ($line !== '') {
                $patterns[] = $line;
            }
        }

        return $patterns;
    }

    public static function is_excluded_request(array $settings): bool
    {
        $patterns = self::exclude_path_patterns($settings);
        if (empty($patterns)) {
            return false;
        }

        $request_uri = (string) ($_SERVER['REQUEST_URI'] ?? '/');
        $path = strtok($request_uri, '?');
        if ($path === false || $path === '') {
            $path = '/';
        }

        foreach ($patterns as $pattern) {
            if (self::path_matches_pattern($path, $pattern)) {
                return true;
            }
        }

        return false;
    }

    public static function path_matches_pattern(string $path, string $pattern): bool
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
            if ($prefix === '') {
                return true;
            }

            return $path === $prefix || strpos($path, $prefix . '/') === 0;
        }

        if (substr($pattern, -1) === '*') {
            $prefix = rtrim($pattern, '*');

            return strpos($path, $prefix) === 0;
        }

        $normalized = untrailingslashit($path);
        $target = untrailingslashit($pattern);

        return $normalized === $target;
    }

    private static function sanitize_hosts($value): string
    {
        $lines = preg_split('/\r\n|\r|\n/', (string) $value);
        $hosts = [];

        foreach ($lines as $line) {
            $line = trim($line);
            if ($line === '') {
                continue;
            }
            $host = wp_parse_url('https://' . ltrim($line, '/'), PHP_URL_HOST) ?: $line;
            $hosts[] = strtolower(sanitize_text_field($host));
        }

        return implode("\n", array_unique($hosts));
    }

    private static function sanitize_exclude_paths($value): string
    {
        $lines = preg_split('/\r\n|\r|\n/', (string) $value);
        $paths = [];

        foreach ($lines as $line) {
            $line = trim($line);
            if ($line === '') {
                continue;
            }
            if ($line[0] !== '/') {
                $line = '/' . $line;
            }
            $paths[] = sanitize_text_field($line);
        }

        return implode("\n", array_unique($paths));
    }

    private static function sanitize_path($value): string
    {
        $path = trim((string) $value);
        $path = str_replace('\\', '/', $path);
        $path = preg_replace('#/+#', '/', $path);

        return sanitize_text_field($path);
    }
}
