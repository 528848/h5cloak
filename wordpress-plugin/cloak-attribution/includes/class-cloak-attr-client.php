<?php

if (!defined('ABSPATH')) {
    exit;
}

class Cloak_Attr_Client
{
    public const COOKIE_DEFAULT = '_ck_vid';

    public static function client_ip(): string
    {
        foreach (['HTTP_CF_CONNECTING_IP', 'HTTP_X_REAL_IP', 'HTTP_X_FORWARDED_FOR', 'REMOTE_ADDR'] as $key) {
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

    public static function current_url(): string
    {
        $scheme = is_ssl() ? 'https' : 'http';
        $host = (string) ($_SERVER['HTTP_HOST'] ?? wp_parse_url(home_url(), PHP_URL_HOST));
        return $scheme . '://' . $host . (string) ($_SERVER['REQUEST_URI'] ?? '/');
    }

    public static function visitor_token(): string
    {
        $name = (string) get_option('cloak_attr_cookie_name', self::COOKIE_DEFAULT);
        return isset($_COOKIE[$name]) ? (string) $_COOKIE[$name] : '';
    }

    /**
     * 调用 v2 判定接口。返回 data 数组；失败返回 null。
     */
    public static function decide(array $settings): ?array
    {
        $base = rtrim((string) $settings['api_base'], '/');
        $secret = (string) $settings['secret'];
        if ($base === '' || strlen($secret) < 12) {
            return null;
        }
        $url = self::current_url();
        $token = self::visitor_token();
        $path = (string) parse_url($url, PHP_URL_PATH);
        $query = (string) parse_url($url, PHP_URL_QUERY);
        $ttl = (int) $settings['cache_ttl'];
        $has_params = $query !== '';
        // 只缓存「无参数 + 已有令牌」的回访判定；带参数的请求必须每次判定以便归因与种 cookie
        $cache_key = '';
        if ($ttl > 0 && $token !== '' && !$has_params) {
            $cache_key = 'cloak_attr_' . substr(hash('sha256', $token . '|' . $path . '|' . ($_SERVER['HTTP_USER_AGENT'] ?? '')), 0, 40);
            $cached = get_transient($cache_key);
            if (is_array($cached)) {
                $cached['cached'] = true;
                return $cached;
            }
        }
        $body = wp_json_encode([
            'url' => $url,
            'method' => $_SERVER['REQUEST_METHOD'] ?? 'GET',
            'visitorToken' => $token,
            'ip' => self::client_ip(),
            'userAgent' => (string) ($_SERVER['HTTP_USER_AGENT'] ?? ''),
            'referer' => (string) ($_SERVER['HTTP_REFERER'] ?? ''),
            'acceptLanguage' => (string) ($_SERVER['HTTP_ACCEPT_LANGUAGE'] ?? ''),
        ], JSON_UNESCAPED_SLASHES);
        $data = self::post($base, '/api/open/v2/decision', $body, $secret, (int) $settings['timeout']);
        if ($data === null) {
            return null;
        }
        if (!empty($data['cookie']['name'])) {
            update_option('cloak_attr_cookie_name', sanitize_text_field($data['cookie']['name']), false);
        }
        if ($cache_key !== '' && ($data['content'] ?? '') !== '' && empty($data['visitorToken'])) {
            set_transient($cache_key, $data, $ttl);
        }
        return $data;
    }

    /** @return array{ok:bool,message:string,data?:array} */
    public static function test(array $settings): array
    {
        $base = rtrim((string) $settings['api_base'], '/');
        $secret = (string) $settings['secret'];
        if ($base === '' || strlen($secret) < 12) {
            return ['ok' => false, 'message' => __('请先填写网关地址与服务端密钥。', 'cloak-attribution')];
        }
        $ping = wp_remote_get($base . '/api/open/v2/ping', ['timeout' => 5]);
        if (is_wp_error($ping)) {
            return ['ok' => false, 'message' => sprintf(__('无法连接网关：%s', 'cloak-attribution'), $ping->get_error_message())];
        }
        if ((int) wp_remote_retrieve_response_code($ping) !== 200) {
            return ['ok' => false, 'message' => sprintf(__('网关探活返回 HTTP %d，请检查地址。', 'cloak-attribution'), wp_remote_retrieve_response_code($ping))];
        }
        $body = wp_json_encode([
            'url' => home_url('/?cloak_test=1'),
            'ip' => self::client_ip() ?: '127.0.0.1',
            'userAgent' => 'CloakAttributionWP/' . CLOAK_ATTR_VERSION,
        ], JSON_UNESCAPED_SLASHES);
        $raw = self::post_raw($base, '/api/open/v2/decision', $body, $secret, 8);
        if (is_wp_error($raw)) {
            return ['ok' => false, 'message' => $raw->get_error_message()];
        }
        $status = (int) wp_remote_retrieve_response_code($raw);
        $json = json_decode(wp_remote_retrieve_body($raw), true);
        if ($status !== 200) {
            $err = $json['data']['error'] ?? '';
            $hint = [
                'unknown_key' => __('密钥前缀不存在：请确认复制的是站点服务端密钥 sk_...', 'cloak-attribution'),
                'bad_signature' => __('签名不匹配：密钥可能不完整，或服务器时间不准。', 'cloak-attribution'),
                'timestamp_expired' => __('服务器时间与网关相差超过 5 分钟，请校时。', 'cloak-attribution'),
                'secret_unavailable' => __('该站点密钥需要在后台重置一次。', 'cloak-attribution'),
                'ip_not_allowed' => __('本服务器 IP 不在站点密钥白名单内。', 'cloak-attribution'),
                'site_disabled' => __('站点已停用。', 'cloak-attribution'),
            ][$err] ?? sprintf(__('接口返回 HTTP %d（%s）', 'cloak-attribution'), $status, $json['msg'] ?? $err);
            return ['ok' => false, 'message' => $hint, 'data' => $json];
        }
        $d = $json['data'] ?? [];
        $known = !empty($d['hostKnown']) ? __('已绑定', 'cloak-attribution') : __('未绑定（服务端接入可忽略，托管接入需绑定）', 'cloak-attribution');
        return [
            'ok' => true,
            'message' => sprintf(__('连接成功。无参数访问判定为「%s」，本站域名在该站点：%s。', 'cloak-attribution'), $d['content'] ?? '?', $known),
            'data' => $d,
        ];
    }

    private static function post(string $base, string $path, string $body, string $secret, int $timeout): ?array
    {
        $raw = self::post_raw($base, $path, $body, $secret, $timeout);
        if (is_wp_error($raw) || (int) wp_remote_retrieve_response_code($raw) !== 200) {
            return null;
        }
        $json = json_decode(wp_remote_retrieve_body($raw), true);
        return is_array($json['data'] ?? null) ? $json['data'] : null;
    }

    private static function post_raw(string $base, string $path, string $body, string $secret, int $timeout)
    {
        return wp_remote_post($base . $path, [
            'timeout' => max(1, $timeout),
            'headers' => Cloak_Attr_Signer::headers($secret, 'POST', $path, $body),
            'body' => $body,
        ]);
    }
}
