<?php

if (!defined('ABSPATH')) {
    exit;
}

class Cloak_Admin
{
    public function __construct()
    {
        add_action('admin_menu', [$this, 'register_menu']);
        add_action('admin_init', [$this, 'register_settings']);
        add_action('admin_enqueue_scripts', [$this, 'enqueue_assets']);
        add_action('wp_ajax_cloak_api_test_connection', [$this, 'ajax_test_connection']);
    }

    public function register_menu(): void
    {
        add_options_page(
            __('API 斗篷', 'cloak-api'),
            __('API 斗篷', 'cloak-api'),
            'manage_options',
            'cloak-api',
            [$this, 'render_settings_page']
        );
    }

    public function register_settings(): void
    {
        register_setting(
            'cloak_api_settings_group',
            Cloak_Settings::OPTION_KEY,
            [
                'type' => 'array',
                'sanitize_callback' => function ($input) {
                    return Cloak_Settings::sanitize(is_array($input) ? $input : []);
                },
                'default' => Cloak_Settings::defaults(),
            ]
        );
    }

    public function enqueue_assets(string $hook): void
    {
        if ($hook !== 'settings_page_cloak-api') {
            return;
        }

        wp_enqueue_style(
            'cloak-api-admin',
            CLOAK_API_URL . 'admin/css/admin.css',
            [],
            CLOAK_API_VERSION
        );

        wp_enqueue_script(
            'cloak-api-admin',
            CLOAK_API_URL . 'admin/js/admin.js',
            ['jquery'],
            CLOAK_API_VERSION,
            true
        );

        wp_localize_script('cloak-api-admin', 'cloakApiAdmin', [
            'ajaxUrl' => admin_url('admin-ajax.php'),
            'nonce' => wp_create_nonce('cloak_api_test'),
            'testing' => __('测试中...', 'cloak-api'),
            'test' => __('测试 API 连接', 'cloak-api'),
        ]);
    }

    public function ajax_test_connection(): void
    {
        check_ajax_referer('cloak_api_test', 'nonce');

        if (!current_user_can('manage_options')) {
            wp_send_json_error(['message' => __('无权限', 'cloak-api')], 403);
        }

        $settings = Cloak_Settings::get();

        if (!empty($_POST['api_url'])) {
            $settings['api_url'] = esc_url_raw(wp_unslash($_POST['api_url']));
        }
        if (!empty($_POST['api_token'])) {
            $settings['api_token'] = sanitize_text_field(wp_unslash($_POST['api_token']));
        }

        $result = Cloak_Api_Client::test_connection($settings);

        if ($result['ok']) {
            wp_send_json_success($result);
        }

        wp_send_json_error($result);
    }

    public function render_settings_page(): void
    {
        if (!current_user_can('manage_options')) {
            return;
        }

        $settings = Cloak_Settings::get();
        $pages = get_pages(['sort_column' => 'post_title', 'sort_order' => 'ASC']);
        $review_modes = Cloak_Settings::review_mode_labels();
        $token_from_config = defined('CLOAK_API_TOKEN') && CLOAK_API_TOKEN;
        $url_from_config = defined('CLOAK_API_URL') && CLOAK_API_URL;

        include CLOAK_API_PATH . 'admin/views/settings-page.php';
    }
}
