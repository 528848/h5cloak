<?php

if (!defined('ABSPATH')) {
    exit;
}

class Cloak_Router
{
    public function __construct()
    {
        add_action('template_redirect', [$this, 'handle'], 1);
    }

    public function handle(): void
    {
        if (!$this->should_run()) {
            return;
        }

        $settings = Cloak_Settings::get();
        if (empty($settings['enabled'])) {
            return;
        }

        if (Cloak_Settings::is_excluded_request($settings)) {
            return;
        }

        if (($settings['scope'] ?? 'entire_site') === 'homepage_only' && !is_front_page()) {
            return;
        }

        $page = Cloak_Api_Client::decide_page($settings);

        if ($page === 'review') {
            $this->dispatch_page($settings, 'review');
            return;
        }

        $this->dispatch_page($settings, 'real');
    }

    private function should_run(): bool
    {
        if (is_admin() || wp_doing_ajax() || wp_doing_cron() || wp_doing_autosave()) {
            return false;
        }

        if (defined('REST_REQUEST') && REST_REQUEST) {
            return false;
        }

        if (defined('WP_CLI') && WP_CLI) {
            return false;
        }

        return true;
    }

    private function dispatch_page(array $settings, string $side): void
    {
        $mode = $settings[$side . '_mode'] ?? Cloak_Settings::MODE_WP_PAGE;

        switch ($mode) {
            case Cloak_Settings::MODE_EXTERNAL:
                $this->redirect_external((string) ($settings[$side . '_external_url'] ?? ''), $settings);
                break;

            case Cloak_Settings::MODE_HTML_REDIRECT:
                $this->redirect_html((string) ($settings[$side . '_html_url'] ?? ''));
                break;

            case Cloak_Settings::MODE_HTML_OUTPUT:
                $this->output_html((string) ($settings[$side . '_html_path'] ?? ''));
                break;

            case Cloak_Settings::MODE_WP_PAGE:
            default:
                $this->redirect_wp_page((int) ($settings[$side . '_page_id'] ?? 0));
                break;
        }
    }

    private function redirect_wp_page(int $page_id): void
    {
        if ($page_id <= 0) {
            return;
        }

        if (is_page($page_id)) {
            return;
        }

        $permalink = get_permalink($page_id);
        if (!$permalink) {
            return;
        }

        wp_safe_redirect($permalink, 302);
        exit;
    }

    private function redirect_external(string $url, array $settings): void
    {
        $url = trim($url);
        if ($url === '' || !$this->is_allowed_external_url($url, $settings)) {
            return;
        }

        if ($this->is_current_url($url)) {
            return;
        }

        wp_redirect(esc_url_raw($url), 302);
        exit;
    }

    private function redirect_html(string $url): void
    {
        $url = trim($url);
        if ($url === '') {
            return;
        }

        if ($this->is_current_url($url)) {
            return;
        }

        $parsed = wp_parse_url($url);
        $path = $parsed['path'] ?? '';
        if ($path !== '' && strpos((string) ($_SERVER['REQUEST_URI'] ?? ''), $path) !== false) {
            return;
        }

        wp_safe_redirect($url, 302);
        exit;
    }

    private function output_html(string $path): void
    {
        $path = trim($path);
        if ($path === '' || !is_readable($path)) {
            return;
        }

        status_header(200);
        nocache_headers();
        header('Content-Type: text/html; charset=utf-8');
        header('X-Robots-Tag: noindex, nofollow');

        readfile($path);
        exit;
    }

    private function is_allowed_external_url(string $url, array $settings): bool
    {
        if (!wp_http_validate_url($url)) {
            return false;
        }

        $scheme = wp_parse_url($url, PHP_URL_SCHEME);
        if ($scheme !== 'https') {
            return false;
        }

        $host = strtolower((string) wp_parse_url($url, PHP_URL_HOST));
        $allowed = Cloak_Settings::allowed_external_hosts($settings);

        return $host !== '' && in_array($host, $allowed, true);
    }

    private function is_current_url(string $url): bool
    {
        $current = (is_ssl() ? 'https://' : 'http://') . ($_SERVER['HTTP_HOST'] ?? '') . ($_SERVER['REQUEST_URI'] ?? '');
        $current = untrailingslashit(strtok($current, '?'));
        $target = untrailingslashit(strtok($url, '?'));

        return $current === $target;
    }
}
