<?php

if (!defined('ABSPATH')) {
    exit;
}

use Cloak\Sdk\Rewriter;

/**
 * 深度代理：在本站服务器内回源审核站，把审核站域名改写成本站域名后在当前地址输出。
 * 继承服务端 SDK 的 \Cloak\Sdk\Rewriter（与 Go/网关同一套改写规则），HTTP 用 WordPress 的
 * wp_remote_* 以兼容各类主机环境。访客流量全程留在本站，不经过斗篷网关。
 */
class Cloak_Attr_Deep_Proxy
{
    private const MAX_BODY = 8388608; // 8MB

    /**
     * 回源 + 改写 + 输出。成功时已输出并返回 true；失败返回 false，交回调用方兜底。
     *
     * @param string $origin      审核站地址，如 https://review.com 或 https://1.2.3.4
     * @param string $host_header 回源 Host（源站用 IP + 自定义 Host 时填），留空用 origin 主机
     */
    public static function serve(string $origin, string $host_header = '', ?\Cloak\Sdk\ResponseCache $cache = null, string $inject = ''): bool
    {
        $origin = rtrim(trim($origin), '/');
        if ($origin === '') {
            return false;
        }
        $op = wp_parse_url($origin);
        if (empty($op['host'])) {
            return false;
        }
        $origin_host_port = $op['host'] . (isset($op['port']) ? ':' . $op['port'] : '');
        $base_path = rtrim((string) ($op['path'] ?? ''), '/');

        $req_uri = (string) ($_SERVER['REQUEST_URI'] ?? '/');
        $req_path = (string) (wp_parse_url($req_uri, PHP_URL_PATH) ?? '/');
        $query = (string) ($_SERVER['QUERY_STRING'] ?? '');
        $target = $origin . self::join_path($base_path, $req_path) . ($query !== '' ? '?' . $query : '');

        $pub = (string) ($_SERVER['HTTP_HOST'] ?? '');
        $scheme = is_ssl() ? 'https' : 'http';
        $host_header = $host_header !== '' ? $host_header : $origin_host_port;
        $cache_key = '';
        if ($cache instanceof \Cloak\Sdk\ResponseCache && $cache->enabled()) {
            $cache_key = \Cloak\Sdk\ResponseCache::key('GET', (string) $op['host'], $pub, $scheme, $req_path === '' ? '/' : $req_path, $query);
            $hit = $cache->get($cache_key);
            if (is_array($hit)) {
                self::emit_cached((int) $hit['status'], (string) $hit['contentType'], (string) $hit['body'], $inject);
                return true;
            }
        }

        $headers = [
            'Accept-Encoding' => 'identity',
        ];
        // 只在回源 Host 和地址里的主机不同时改 Host。相同的 Host 交给 HTTP 客户端自己带，
        // 部分主机上手工再写一次会让回源直接失败。
        if (strcasecmp(Rewriter::stripPort($host_header), Rewriter::stripPort($op['host'])) !== 0) {
            $headers['Host'] = Rewriter::stripPort($host_header);
        }
        if (!empty($_SERVER['HTTP_USER_AGENT'])) {
            $headers['User-Agent'] = (string) $_SERVER['HTTP_USER_AGENT'];
        }
        if (!empty($_SERVER['HTTP_ACCEPT_LANGUAGE'])) {
            $headers['Accept-Language'] = (string) $_SERVER['HTTP_ACCEPT_LANGUAGE'];
        }
        if (!empty($_SERVER['HTTP_ACCEPT'])) {
            $headers['Accept'] = (string) $_SERVER['HTTP_ACCEPT'];
        }
        $ip = Cloak_Attr_Client::client_ip();
        if ($ip !== '') {
            $headers['X-Forwarded-For'] = $ip;
            $headers['X-Real-IP'] = $ip;
        }
        $headers['X-Forwarded-Proto'] = $scheme;

        $res = null;
        $status = 0;
        for ($hop = 0; $hop < 4; $hop++) {
            $res = wp_remote_get($target, [
                'timeout' => 6,
                'redirection' => 0,
                'limit_response_size' => self::MAX_BODY + 1,
                'headers' => $headers,
                'sslverify' => true,
            ]);
            if (is_wp_error($res)) {
                return false;
            }
            $status = (int) wp_remote_retrieve_response_code($res);
            if ($status < 300 || $status >= 400) {
                break;
            }
            $next = self::resolve_redirect($target, self::header_value($res, 'location'));
            $next_host = (string) (wp_parse_url($next, PHP_URL_HOST) ?? '');
            $cur_host = (string) (wp_parse_url($target, PHP_URL_HOST) ?? '');
            // 跨站跳转（如 baidu.com → www.baidu.com）不交给浏览器，否则地址栏会离开本站。
            if ($next === '' || $next_host === '' || strcasecmp($next_host, $cur_host) !== 0) {
                return false;
            }
            $target = $next;
        }
        if ($status === 0 || ($status >= 300 && $status < 400)) {
            return false;
        }
        $encoding = self::header_value($res, 'content-encoding');
        $body = Rewriter::inflate((string) wp_remote_retrieve_body($res), $encoding);
        $content_type = self::header_value($res, 'content-type');

        $hosts = [Rewriter::stripPort($op['host'])];
        $hh = Rewriter::stripPort($host_header);
        if ($hh !== '' && strcasecmp($hh, $hosts[0]) !== 0) {
            $hosts[] = $hh;
        }
        $body = self::rewrite_document($body, $hosts, $pub, $scheme, $content_type);

        if (!headers_sent()) {
            status_header($status);
            nocache_headers();
            header('X-Cloak-Content: review');
            header('X-Robots-Tag: noindex, nofollow');
            if ($content_type !== '') {
                header('Content-Type: ' . $content_type);
            }
            header('X-Cloak-Cache: miss');
            self::forward_set_cookies($res, $op['host'], $hh, $pub);
        }
        if ($cache instanceof \Cloak\Sdk\ResponseCache && $cache_key !== '' && $status === 200) {
            $cache->put($cache_key, 200, $content_type, $body);
        }
        echo self::with_browser_bar($body, $content_type, $inject); // phpcs:ignore WordPress.Security.EscapeOutput
        return true;
    }

    /** 按插件设置构造缓存。TTL 为 0 时关闭。 */
    public static function cache_from_settings(array $settings): ?\Cloak\Sdk\ResponseCache
    {
        $ttl = (int) ($settings['proxy_cache_ttl'] ?? 0);
        if ($ttl <= 0) {
            return null;
        }
        return new \Cloak\Sdk\ResponseCache(self::cache_dir(), $ttl);
    }

    public static function purge_cache(): int
    {
        return (new \Cloak\Sdk\ResponseCache(self::cache_dir(), 1))->purge();
    }

    public static function cache_dir(): string
    {
        return WP_CONTENT_DIR . '/cache/cloak-attribution';
    }

    private static function emit_cached(int $status, string $content_type, string $body, string $inject): void
    {
        if (!headers_sent()) {
            status_header($status);
            nocache_headers();
            header('X-Cloak-Content: review');
            header('X-Robots-Tag: noindex, nofollow');
            header('X-Cloak-Cache: hit');
            if ($content_type !== '') {
                header('Content-Type: ' . $content_type);
            }
        }
        echo self::with_browser_bar($body, $content_type, $inject); // phpcs:ignore WordPress.Security.EscapeOutput
    }

    /** 引导条按本次访客现插，不写进缓存，避免把某一个人的接力地址发给下一位。 */
    private static function with_browser_bar(string $body, string $content_type, string $bar): string
    {
        if ($bar === '') {
            return $body;
        }
        $html = stripos($content_type, 'html') !== false || Rewriter::looksLikeHtml($body);
        if (!$html) {
            return $body;
        }
        $pos = strripos($body, '</body>');
        if ($pos === false) {
            return $body . $bar;
        }
        return substr($body, 0, $pos) . $bar . substr($body, $pos);
    }

    /**
     * 把审核站文档里的链接、跳转和正文域名改成访客域名。
     * 文本类型、没给类型、或正文本身就是 HTML 时都改；图片等二进制不动。
     *
     * @param string[] $hosts
     */
    public static function rewrite_document(string $body, array $hosts, string $public, string $scheme, string $contentType = ''): string
    {
        if ($body === '' || strlen($body) > self::MAX_BODY) {
            return $body;
        }
        $textual = $contentType === ''
            || Rewriter::rewritableContentType($contentType)
            || Rewriter::looksLikeHtml($body);
        if (!$textual) {
            return $body;
        }
        return Rewriter::rewriteBodyHost($body, $hosts, $public, $scheme !== '' ? $scheme : 'https');
    }

    /** WordPress 对重复头会返回数组，统一取第一个。 */
    public static function header_value(array $res, string $name): string
    {
        $value = wp_remote_retrieve_header($res, $name);
        if (is_array($value)) {
            $value = reset($value);
        }
        return is_string($value) ? $value : '';
    }

    private static function forward_set_cookies(array $res, string $origin_hostname, string $origin_host, string $pub): void
    {
        $cookies = wp_remote_retrieve_header($res, 'set-cookie');
        if ($cookies === '') {
            return;
        }
        foreach ((array) $cookies as $c) {
            header('Set-Cookie: ' . self::rewrite_cookie_domain((string) $c, $origin_hostname, $origin_host, $pub), false);
        }
    }

    private static function rewrite_location(string $loc, string $origin_host_port, string $pub, string $scheme): string
    {
        $u = wp_parse_url($loc);
        if (empty($u['host'])) {
            return $loc;
        }
        $hp = $u['host'] . (isset($u['port']) ? ':' . $u['port'] : '');
        if (strcasecmp(Rewriter::stripPort($hp), Rewriter::stripPort($origin_host_port)) !== 0) {
            return $loc;
        }
        $path = (string) ($u['path'] ?? '');
        $q = isset($u['query']) ? '?' . $u['query'] : '';
        $frag = isset($u['fragment']) ? '#' . $u['fragment'] : '';
        return $scheme . '://' . $pub . ($path === '' ? '/' : $path) . $q . $frag;
    }

    private static function rewrite_cookie_domain(string $c, string $origin_hostname, string $origin_host, string $pub): string
    {
        $parts = explode(';', $c);
        $out = [];
        foreach ($parts as $p) {
            $kv = trim($p);
            if (stripos($kv, 'domain=') === 0) {
                $d = ltrim(substr($kv, 7), '.');
                $pubh = Rewriter::stripPort($pub);
                if (strcasecmp($d, $origin_hostname) === 0
                    || strcasecmp($d, Rewriter::stripPort($origin_host)) === 0
                    || (strcasecmp($d, $pubh) !== 0 && substr($pubh, -strlen('.' . $d)) !== '.' . $d)) {
                    continue;
                }
            }
            $out[] = $p;
        }
        return implode(';', $out);
    }

    /** 把源站 301/302 的 Location 解析成绝对地址，相对路径按当前请求补全。 */
    private static function resolve_redirect(string $current, string $location): string
    {
        $location = trim($location);
        if ($location === '') {
            return '';
        }
        if (preg_match('#^https?://#i', $location)) {
            return $location;
        }
        $parts = wp_parse_url($current);
        if (empty($parts['host'])) {
            return '';
        }
        $scheme = $parts['scheme'] ?? 'https';
        $port = isset($parts['port']) ? ':' . $parts['port'] : '';
        $origin = $scheme . '://' . $parts['host'] . $port;
        if (isset($location[0]) && $location[0] === '/') {
            return $origin . $location;
        }
        $dir = (string) ($parts['path'] ?? '/');
        $slash = strrpos($dir, '/');
        $dir = $slash === false ? '/' : substr($dir, 0, $slash + 1);
        return $origin . $dir . $location;
    }

    private static function join_path(string $base, string $path): string
    {
        if ($path === '') {
            return $base !== '' ? $base : '/';
        }
        if ($path[0] !== '/') {
            $path = '/' . $path;
        }
        return $base . $path;
    }
}
