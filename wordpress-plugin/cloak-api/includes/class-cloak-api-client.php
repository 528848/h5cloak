<?php

if (!defined('ABSPATH')) {
    exit;
}

class Cloak_Api_Client
{
    public static function get_client_ip(): string
    {
        $keys = ['HTTP_CF_CONNECTING_IP', 'HTTP_X_FORWARDED_FOR', 'REMOTE_ADDR'];

        foreach ($keys as $key) {
            if (empty($_SERVER[$key])) {
                continue;
            }

            $ip = trim(explode(',', (string) $_SERVER[$key])[0]);
            if (filter_var($ip, FILTER_VALIDATE_IP)) {
                return $ip;
            }
        }

        return '';
    }

    /**
     * @return string review|real
     */
    public static function decide_page(array $settings): string
    {
        $api_url = trim((string) ($settings['api_url'] ?? ''));
        $api_token = trim((string) ($settings['api_token'] ?? ''));

        if ($api_url === '' || $api_token === '') {
            return self::fallback_page($settings);
        }

        $body = wp_json_encode([
            'ip' => self::get_client_ip(),
            'userAgent' => $_SERVER['HTTP_USER_AGENT'] ?? '',
            'referer' => $_SERVER['HTTP_REFERER'] ?? '',
            'acceptLanguage' => $_SERVER['HTTP_ACCEPT_LANGUAGE'] ?? '',
        ]);

        $response = wp_remote_post($api_url, [
            'timeout' => 5,
            'headers' => [
                'Content-Type' => 'application/json',
                'X-API-Key' => $api_token,
            ],
            'body' => $body,
        ]);

        if (is_wp_error($response)) {
            return self::fallback_page($settings);
        }

        $status = (int) wp_remote_retrieve_response_code($response);
        if ($status < 200 || $status >= 300) {
            return self::fallback_page($settings);
        }

        $data = json_decode(wp_remote_retrieve_body($response), true);
        $page = $data['data']['page'] ?? null;

        return in_array($page, ['review', 'real'], true) ? $page : self::fallback_page($settings);
    }

    /**
     * @return array{ok:bool,message:string,page?:string,raw?:mixed}
     */
    public static function test_connection(array $settings): array
    {
        $api_url = trim((string) ($settings['api_url'] ?? ''));
        $api_token = trim((string) ($settings['api_token'] ?? ''));

        if ($api_url === '' || $api_token === '') {
            return [
                'ok' => false,
                'message' => __('请先填写 API 地址和 Token。', 'cloak-api'),
            ];
        }

        $body = wp_json_encode([
            'ip' => self::get_client_ip() ?: '127.0.0.1',
            'userAgent' => 'CloakApiWordPressPlugin/1.0',
            'referer' => home_url('/'),
            'acceptLanguage' => 'en-US',
        ]);

        $response = wp_remote_post($api_url, [
            'timeout' => 8,
            'headers' => [
                'Content-Type' => 'application/json',
                'X-API-Key' => $api_token,
            ],
            'body' => $body,
        ]);

        if (is_wp_error($response)) {
            return [
                'ok' => false,
                'message' => $response->get_error_message(),
            ];
        }

        $status = (int) wp_remote_retrieve_response_code($response);
        $decoded = json_decode(wp_remote_retrieve_body($response), true);

        if ($status === 401) {
            return [
                'ok' => false,
                'message' => __('API Key 无效或已停用。', 'cloak-api'),
                'raw' => $decoded,
            ];
        }

        if ($status < 200 || $status >= 300) {
            return [
                'ok' => false,
                'message' => sprintf(__('接口返回 HTTP %d', 'cloak-api'), $status),
                'raw' => $decoded,
            ];
        }

        $page = $decoded['data']['page'] ?? '';

        return [
            'ok' => true,
            'message' => sprintf(__('连接成功，当前测试判定为：%s', 'cloak-api'), $page === 'real' ? 'real（真实页）' : 'review（审核页）'),
            'page' => $page,
            'raw' => $decoded,
        ];
    }

    private static function fallback_page(array $settings): string
    {
        $fail_mode = $settings['fail_mode'] ?? 'review';

        return in_array($fail_mode, ['review', 'real'], true) ? $fail_mode : 'review';
    }
}
