<?php

namespace Cloak\Sdk;

/**
 * 深度代理响应缓存。存的是已经改写好的正文，命中后不再回源、不再改写。
 *
 * 各语言 SDK 使用同一套键：
 * sha256(METHOD \n originHost \n publicHost \n scheme \n path \n normalizedQuery)
 * 广告点击参数不进入键。登录、购物车、文章密码这些 Cookie 不读也不写缓存。
 * 只缓存 200，单条正文不超过 2MB。TTL 由调用方传入，0 表示关闭。
 */
final class ResponseCache
{
    public const MAX_OBJECT = 2097152;

    private const TRACKING = [
        'utm_source', 'utm_medium', 'utm_campaign', 'utm_term', 'utm_content', 'utm_id',
        'fbclid', 'gclid', 'gbraid', 'wbraid', 'msclkid', 'ttclid', 'twclid',
        'mc_cid', 'mc_eid', 'yclid', 'igshid', 'srsltid', '_ga', 'ref',
    ];

    /** @var string */
    private $dir;

    /** @var int */
    private $ttl;

    public function __construct(string $dir, int $ttl)
    {
        $this->dir = rtrim($dir, '/\\');
        $this->ttl = $ttl;
    }

    public function enabled(): bool
    {
        return $this->ttl > 0 && $this->dir !== '';
    }

    public static function key(string $method, string $originHost, string $publicHost, string $scheme, string $path, string $query): string
    {
        $material = strtoupper($method) . "\n"
            . strtolower($originHost) . "\n"
            . strtolower($publicHost) . "\n"
            . strtolower($scheme) . "\n"
            . $path . "\n"
            . self::normalizeQuery($query);
        return hash('sha256', $material);
    }

    public static function normalizeQuery(string $query): string
    {
        if ($query === '') {
            return '';
        }
        parse_str($query, $params);
        if (!is_array($params)) {
            return '';
        }
        foreach (self::TRACKING as $name) {
            unset($params[$name]);
        }
        ksort($params);
        return http_build_query($params);
    }

    /** 登录、购物车、评论作者、文章密码会让页面因人而异，这类请求绕过缓存。 */
    public static function isPersonalRequest(): bool
    {
        $cookie = (string) ($_SERVER['HTTP_COOKIE'] ?? '');
        if ($cookie === '') {
            return false;
        }
        return (bool) preg_match('/(?:^|;\s*)(?:wordpress_logged_in_|wp_woocommerce_session_|comment_author_|wp-postpass_)/i', $cookie);
    }

    /**
     * @return array{status:int,contentType:string,body:string}|null
     */
    public function get(string $key, ?int $now = null): ?array
    {
        if (!$this->enabled() || !self::validKey($key) || self::isPersonalRequest()) {
            return null;
        }
        $path = $this->path($key);
        if (!is_file($path)) {
            return null;
        }
        $raw = file_get_contents($path);
        if (!is_string($raw) || strncmp($raw, "CKC1\n", 5) !== 0) {
            return null;
        }
        $nl = strpos($raw, "\n", 5);
        if ($nl === false) {
            return null;
        }
        $meta = json_decode(substr($raw, 5, $nl - 5), true);
        if (!is_array($meta)) {
            return null;
        }
        $now = $now ?? time();
        if ((int) ($meta['exp'] ?? 0) < $now) {
            @unlink($path);
            return null;
        }
        return [
            'status' => (int) ($meta['status'] ?? 200),
            'contentType' => (string) ($meta['type'] ?? ''),
            'body' => substr($raw, $nl + 1),
        ];
    }

    public function put(string $key, int $status, string $contentType, string $body, ?int $now = null): bool
    {
        if (!$this->enabled() || !self::validKey($key) || self::isPersonalRequest()) {
            return false;
        }
        if ($status !== 200 || $body === '' || strlen($body) > self::MAX_OBJECT) {
            return false;
        }
        if (!$this->ensure()) {
            return false;
        }
        $meta = json_encode([
            'status' => $status,
            'type' => $contentType,
            'exp' => ($now ?? time()) + $this->ttl,
        ], JSON_UNESCAPED_SLASHES);
        if (!is_string($meta)) {
            return false;
        }
        $dir = dirname($this->path($key));
        if (!is_dir($dir) && !@mkdir($dir, 0755, true)) {
            return false;
        }
        $tmp = $dir . DIRECTORY_SEPARATOR . '.' . $key . '.' . bin2hex(random_bytes(4)) . '.tmp';
        if (file_put_contents($tmp, "CKC1\n" . $meta . "\n" . $body, LOCK_EX) === false) {
            return false;
        }
        if (!@rename($tmp, $this->path($key))) {
            @unlink($tmp);
            return false;
        }
        return true;
    }

    /** 删掉缓存正文，保留目录和防直接访问的文件。返回删除的文件数。 */
    public function purge(): int
    {
        if ($this->dir === '' || !is_dir($this->dir)) {
            return 0;
        }
        return $this->deleteInside($this->dir, false);
    }

    private function deleteInside(string $dir, bool $removeDir): int
    {
        $n = 0;
        $items = @scandir($dir);
        if (!is_array($items)) {
            return 0;
        }
        foreach ($items as $item) {
            if ($item === '.' || $item === '..' || $item === '.htaccess' || $item === 'index.php') {
                continue;
            }
            $path = $dir . DIRECTORY_SEPARATOR . $item;
            if (is_dir($path)) {
                $n += $this->deleteInside($path, true);
                continue;
            }
            if (@unlink($path)) {
                $n++;
            }
        }
        if ($removeDir) {
            @rmdir($dir);
        }
        return $n;
    }

    private function ensure(): bool
    {
        if (!is_dir($this->dir) && !@mkdir($this->dir, 0755, true)) {
            return false;
        }
        $ht = $this->dir . DIRECTORY_SEPARATOR . '.htaccess';
        if (!is_file($ht)) {
            @file_put_contents($ht, "Require all denied\nDeny from all\n");
        }
        $idx = $this->dir . DIRECTORY_SEPARATOR . 'index.php';
        if (!is_file($idx)) {
            @file_put_contents($idx, "<?php\n// Silence is golden.\n");
        }
        return is_writable($this->dir);
    }

    private function path(string $key): string
    {
        return $this->dir . DIRECTORY_SEPARATOR . substr($key, 0, 2) . DIRECTORY_SEPARATOR . $key;
    }

    private static function validKey(string $key): bool
    {
        return (bool) preg_match('/^[a-f0-9]{64}$/', $key);
    }
}
