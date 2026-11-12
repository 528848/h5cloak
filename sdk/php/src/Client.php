<?php

namespace Cloak\Sdk;

/**
 * 斗篷服务端 SDK（PHP 参考实现，框架无关）。
 *
 * 在客户服务器入口调用判定接口，决定「放行真实内容」还是「审核」；审核时在本机深度代理
 * 审核站（回源 → 把审核站域名改写成客户域名 → 输出）。访客流量不经过斗篷网关。
 *
 *   $client = new \Cloak\Sdk\Client([
 *       'apiBase'      => 'https://gw.example.com',
 *       'secretKey'    => 'sk_xxx',
 *       'reviewOrigin' => 'https://review.com',
 *   ]);
 *   $client->handle();   // 一行接入：放行 return，审核直接输出并 exit
 *
 * WordPress 等框架可复用 decide() / proxyReview() 自行分流。
 */
final class Client
{
    /** @var array<string,mixed> */
    private array $cfg;

    /** HTTP 回源/判定委托；为 null 时用内置 curl。便于在 WP 里换成 wp_remote_*。 */
    private ?HttpTransport $http;

    private const DEFAULT_SKIP_PATHS = [
        '/wp-admin', '/wp-login', '/wp-json', '/wp-cron', '/xmlrpc.php',
        '/_cloak', '/favicon.ico', '/robots.txt', '/sitemap',
    ];
    private const DEFAULT_SKIP_SUFFIXES = [
        '.css', '.js', '.mjs', '.png', '.jpg', '.jpeg', '.gif', '.webp', '.svg', '.ico',
        '.woff', '.woff2', '.ttf', '.eot', '.map', '.mp4', '.webm', '.pdf', '.zip',
    ];
    private const DEFAULT_IP_HEADERS = ['HTTP_CF_CONNECTING_IP', 'HTTP_X_REAL_IP', 'HTTP_X_FORWARDED_FOR', 'REMOTE_ADDR'];

    private const MAX_PROXY_BODY = 8388608; // 8MB

    /** @param array<string,mixed> $cfg */
    public function __construct(array $cfg, ?HttpTransport $http = null)
    {
        $cfg += [
            'apiBase' => '',
            'secretKey' => '',
            'reviewOrigin' => '',
            'visitorHost' => '',
            'reviewHost' => '',
            'cookieName' => '_ck_vid',
            'timeout' => 3,
            'failOpen' => false,
            'skipPaths' => self::DEFAULT_SKIP_PATHS,
            'skipSuffixes' => self::DEFAULT_SKIP_SUFFIXES,
            'clientIpHeaders' => self::DEFAULT_IP_HEADERS,
            'userAgent' => 'cloak-sdk-php/1.0',
            'cacheDir' => '',
            'cacheTtl' => 0,
        ];
        $cfg['apiBase'] = rtrim((string) $cfg['apiBase'], '/');
        $cfg['reviewOrigin'] = rtrim((string) $cfg['reviewOrigin'], '/');
        $this->cfg = $cfg;
        $this->http = $http ?? (function_exists('curl_init') ? new CurlTransport() : null);
    }

    /** @return mixed 配置项。 */
    public function config(string $key)
    {
        return $this->cfg[$key] ?? null;
    }

    /**
     * 一行接入：判定 + 分流。放行时 return（调用方继续渲染真实内容）；审核/拦截时直接输出并 exit。
     */
    public function handle(): void
    {
        if ($this->shouldSkip()) {
            return;
        }
        $d = $this->decide();
        if ($d === null) {
            if (!empty($this->cfg['failOpen'])) {
                return;
            }
            $d = ['content' => 'review', 'failed' => true];
        }
        $this->apply($d);
    }

    /**
     * 按判定结果分流。放行 return；其它直接输出并 exit。
     * @param array<string,mixed> $d
     */
    public function apply(array $d): void
    {
        if (!empty($d['cookie']) && !empty($d['visitorToken'])) {
            $this->writeCookie($d);
        }
        $content = (string) ($d['content'] ?? 'review');
        if ($content === 'real' || $content === 'bypass') {
            return;
        }
        if ($content === 'block') {
            $this->status(403);
            echo '<!doctype html><title>403</title><h1>403</h1>';
            exit;
        }
        if ($content === 'not_found') {
            $this->status(404);
            echo '<!doctype html><title>404</title><h1>404</h1>';
            exit;
        }
        // review：平台模板直出，否则本机深度代理
        $hint = $d['renderHint'] ?? [];
        if (($hint['type'] ?? '') === 'html' && !empty($hint['html'])) {
            $this->reviewHeaders();
            header('Content-Type: text/html; charset=utf-8');
            echo $hint['html'];
            exit;
        }
        if (!$this->proxyReview()) {
            $this->status(502);
            echo '<!doctype html><title>502</title><h1>502</h1>';
            exit;
        }
        exit;
    }

    /**
     * 调用 v2 判定接口。返回 data 数组；失败返回 null。
     * @return array<string,mixed>|null
     */
    public function decide(): ?array
    {
        if ($this->cfg['apiBase'] === '' || strlen((string) $this->cfg['secretKey']) < 12 || $this->http === null) {
            return null;
        }
        $url = $this->currentUrl();
        $body = json_encode([
            'url' => $url,
            'method' => $_SERVER['REQUEST_METHOD'] ?? 'GET',
            'visitorToken' => $this->visitorToken(),
            'ip' => $this->clientIp(),
            'userAgent' => (string) ($_SERVER['HTTP_USER_AGENT'] ?? ''),
            'referer' => (string) ($_SERVER['HTTP_REFERER'] ?? ''),
            'acceptLanguage' => (string) ($_SERVER['HTTP_ACCEPT_LANGUAGE'] ?? ''),
        ], JSON_UNESCAPED_SLASHES);
        $path = '/api/open/v2/decision';
        $headers = Signer::headers((string) $this->cfg['secretKey'], 'POST', $path, $body);
        $resp = $this->http->request('POST', $this->cfg['apiBase'] . $path, $headers, $body, (int) $this->cfg['timeout']);
        if ($resp === null || $resp->status !== 200) {
            return null;
        }
        $json = json_decode($resp->body, true);
        return is_array($json['data'] ?? null) ? $json['data'] : null;
    }

    /**
     * 在客户服务器内深度代理审核站：回源 → 改写头与正文 → 输出。只代理配置的 reviewOrigin（防 SSRF）。
     * 成功时已输出响应并返回 true；失败返回 false。
     */
    public function proxyReview(): bool
    {
        $origin = (string) $this->cfg['reviewOrigin'];
        if ($origin === '' || $this->http === null) {
            return false;
        }
        $op = parse_url($origin);
        if (empty($op['host'])) {
            return false;
        }
        $originHostPort = $op['host'] . (isset($op['port']) ? ':' . $op['port'] : '');
        $basePath = rtrim((string) ($op['path'] ?? ''), '/');
        $reqPath = (string) parse_url((string) ($_SERVER['REQUEST_URI'] ?? '/'), PHP_URL_PATH);
        $query = (string) ($_SERVER['QUERY_STRING'] ?? '');
        $target = $origin . ($basePath !== '' ? '' : '') . $this->joinPath($basePath, $reqPath);
        if ($query !== '') {
            $target .= '?' . $query;
        }

        $hostHeader = (string) $this->cfg['reviewHost'];
        if ($hostHeader === '') {
            $hostHeader = $originHostPort;
        }
        $pub = $this->visitorHost();
        $scheme = $this->scheme();
        $method = strtoupper((string) ($_SERVER['REQUEST_METHOD'] ?? 'GET'));
        $cache = $this->responseCache();
        $cacheKey = $cache === null ? '' : ResponseCache::key('GET', (string) $op['host'], $pub, $scheme, $reqPath === '' ? '/' : $reqPath, $query);
        if ($cache !== null && $cacheKey !== '' && $method === 'GET') {
            $hit = $cache->get($cacheKey);
            if (is_array($hit)) {
                $this->status((int) $hit['status']);
                if ($hit['contentType'] !== '') {
                    header('Content-Type: ' . $hit['contentType']);
                }
                header('Cache-Control: no-store');
                header('X-Robots-Tag: noindex, nofollow');
                header('X-Cloak-Cache: hit');
                echo $hit['body'];
                return true;
            }
        }

        $headers = ['Accept-Encoding' => 'identity', 'Host' => Rewriter::stripPort($hostHeader)];
        if (!empty($_SERVER['HTTP_USER_AGENT'])) {
            $headers['User-Agent'] = (string) $_SERVER['HTTP_USER_AGENT'];
        }
        if (!empty($_SERVER['HTTP_ACCEPT_LANGUAGE'])) {
            $headers['Accept-Language'] = (string) $_SERVER['HTTP_ACCEPT_LANGUAGE'];
        }
        if (!empty($_SERVER['HTTP_ACCEPT'])) {
            $headers['Accept'] = (string) $_SERVER['HTTP_ACCEPT'];
        }
        $ip = $this->clientIp();
        if ($ip !== '') {
            $headers['X-Forwarded-For'] = $ip;
            $headers['X-Real-IP'] = $ip;
        }
        $headers['X-Forwarded-Proto'] = $scheme;

        $reqBody = ($method === 'GET' || $method === 'HEAD') ? null : file_get_contents('php://input');
        $sni = self::tlsServerName($op['host'], $hostHeader);

        $resp = $this->http->request($method, $target, $headers, $reqBody, (int) $this->cfg['timeout'], self::MAX_PROXY_BODY + 1, $sni);
        if ($resp === null) {
            return false;
        }

        $hosts = [Rewriter::stripPort($op['host'])];
        $hh = Rewriter::stripPort($hostHeader);
        if ($hh !== '' && strcasecmp($hh, $hosts[0]) !== 0) {
            $hosts[] = $hh;
        }

        $this->status($resp->status);
        $contentType = '';
        foreach ($resp->headers as [$k, $v]) {
            $lk = strtolower($k);
            if (in_array($lk, ['server', 'x-powered-by', 'via', 'content-security-policy', 'strict-transport-security', 'content-encoding', 'content-length', 'transfer-encoding', 'connection'], true)) {
                continue;
            }
            if ($lk === 'content-type') {
                $contentType = $v;
            }
            if ($lk === 'location') {
                $v = $this->rewriteLocation($v, $originHostPort, $pub, $scheme);
            }
            if ($lk === 'set-cookie') {
                $v = $this->rewriteCookieDomain($v, $op['host'], $hh, $pub);
                header($k . ': ' . $v, false); // 允许多个 Set-Cookie
                continue;
            }
            header($k . ': ' . $v, true);
        }
        header('Cache-Control: no-store', true);
        header('X-Robots-Tag: noindex, nofollow', true);

        $out = $resp->body;
        $needRewrite = $contentType !== ''
            ? Rewriter::rewritableContentType($contentType)
            : Rewriter::looksLikeHtml($out);
        if (strlen($out) <= self::MAX_PROXY_BODY && $needRewrite) {
            $out = Rewriter::rewriteBodyHost($out, $hosts, $pub, $scheme);
        }
        if ($cache !== null && $cacheKey !== '' && $method === 'GET' && $resp->status === 200) {
            $cache->put($cacheKey, 200, $contentType, $out);
        }
        header('X-Cloak-Cache: miss');
        echo $out;
        return true;
    }

    private function responseCache(): ?ResponseCache
    {
        $dir = (string) $this->cfg['cacheDir'];
        $ttl = (int) $this->cfg['cacheTtl'];
        if ($dir === '' || $ttl <= 0) {
            return null;
        }
        return new ResponseCache($dir, $ttl);
    }

    // —— 请求上下文 ——

    public function shouldSkip(): bool
    {
        $method = strtoupper((string) ($_SERVER['REQUEST_METHOD'] ?? 'GET'));
        if ($method !== 'GET' && $method !== 'HEAD') {
            return true;
        }
        $path = strtolower((string) parse_url((string) ($_SERVER['REQUEST_URI'] ?? '/'), PHP_URL_PATH));
        foreach ((array) $this->cfg['skipPaths'] as $sp) {
            if (strncmp($path, (string) $sp, strlen((string) $sp)) === 0) {
                return true;
            }
        }
        foreach ((array) $this->cfg['skipSuffixes'] as $sx) {
            if (substr($path, -strlen((string) $sx)) === $sx) {
                return true;
            }
        }
        return false;
    }

    public function clientIp(): string
    {
        foreach ((array) $this->cfg['clientIpHeaders'] as $key) {
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

    public function visitorHost(): string
    {
        if ($this->cfg['visitorHost'] !== '') {
            return (string) $this->cfg['visitorHost'];
        }
        return (string) ($_SERVER['HTTP_HOST'] ?? '');
    }

    public function currentUrl(): string
    {
        return $this->scheme() . '://' . $this->visitorHost() . (string) ($_SERVER['REQUEST_URI'] ?? '/');
    }

    public function visitorToken(): string
    {
        $name = (string) $this->cfg['cookieName'];
        return isset($_COOKIE[$name]) ? (string) $_COOKIE[$name] : '';
    }

    private function scheme(): string
    {
        if ((!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off') || (int) ($_SERVER['SERVER_PORT'] ?? 0) === 443) {
            return 'https';
        }
        if (!empty($_SERVER['HTTP_X_FORWARDED_PROTO'])) {
            return trim(explode(',', (string) $_SERVER['HTTP_X_FORWARDED_PROTO'])[0]);
        }
        return 'http';
    }

    // —— 响应处理 ——

    private function status(int $code): void
    {
        if (!headers_sent()) {
            http_response_code($code);
            header('Cache-Control: no-store');
        }
    }

    private function reviewHeaders(): void
    {
        if (!headers_sent()) {
            http_response_code(200);
            header('Cache-Control: no-store');
            header('X-Robots-Tag: noindex, nofollow');
        }
    }

    /** @param array<string,mixed> $d */
    private function writeCookie(array $d): void
    {
        if (headers_sent() || !is_array($d['cookie'] ?? null)) {
            return;
        }
        $c = $d['cookie'];
        $name = (string) ($c['name'] ?? $this->cfg['cookieName']);
        if ($name === '' || preg_match('/[=,; \t\r\n]/', $name)) {
            return;
        }
        $opts = [
            'expires' => time() + max(60, (int) ($c['maxAge'] ?? 2592000)),
            'path' => (string) ($c['path'] ?? '/') ?: '/',
            'secure' => !empty($c['secure']),
            'httponly' => !empty($c['httpOnly']),
            'samesite' => (string) ($c['sameSite'] ?? 'Lax') ?: 'Lax',
        ];
        $domain = trim((string) ($c['domain'] ?? ''));
        if ($domain !== '') {
            $opts['domain'] = $domain;
        }
        try {
            setcookie($name, (string) $d['visitorToken'], $opts);
        } catch (\Throwable $e) {
            // 忽略，不影响页面
        }
    }

    private function rewriteLocation(string $loc, string $originHostPort, string $pub, string $scheme): string
    {
        $u = parse_url($loc);
        if (empty($u['host'])) {
            return $loc;
        }
        $hp = $u['host'] . (isset($u['port']) ? ':' . $u['port'] : '');
        if (strcasecmp(Rewriter::stripPort($hp), Rewriter::stripPort($originHostPort)) !== 0) {
            return $loc;
        }
        $path = (string) ($u['path'] ?? '');
        $q = isset($u['query']) ? '?' . $u['query'] : '';
        $frag = isset($u['fragment']) ? '#' . $u['fragment'] : '';
        return $scheme . '://' . $pub . ($path === '' ? '/' : $path) . $q . $frag;
    }

    private function rewriteCookieDomain(string $c, string $originHostname, string $originHost, string $pub): string
    {
        $parts = explode(';', $c);
        $out = [];
        foreach ($parts as $p) {
            $kv = trim($p);
            if (stripos($kv, 'domain=') === 0) {
                $d = ltrim(substr($kv, 7), '.');
                $pubh = Rewriter::stripPort($pub);
                if (strcasecmp($d, $originHostname) === 0
                    || strcasecmp($d, Rewriter::stripPort($originHost)) === 0
                    || (strcasecmp($d, $pubh) !== 0 && substr($pubh, -strlen('.' . $d)) !== '.' . $d)) {
                    continue;
                }
            }
            $out[] = $p;
        }
        return implode(';', $out);
    }

    private function joinPath(string $base, string $path): string
    {
        if ($path === '') {
            return $base !== '' ? $base : '/';
        }
        if ($path[0] !== '/') {
            $path = '/' . $path;
        }
        return $base . $path;
    }

    /** 回源 Host 是另一个域名（非 IP）且与 URL 主机不同时，用它做 SNI。 */
    private static function tlsServerName(string $urlHost, string $hostHeader): string
    {
        $name = rtrim(Rewriter::stripPort(trim($hostHeader)), '.');
        if ($name === '' || filter_var($name, FILTER_VALIDATE_IP)) {
            return '';
        }
        if (strcasecmp($name, Rewriter::stripPort($urlHost)) === 0) {
            return '';
        }
        return $name;
    }
}
