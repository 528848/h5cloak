<?php

namespace Cloak\Sdk;

/**
 * 深度代理域名改写器。与 Go SDK sdk/go/rewrite.go、网关 internal/proxy/rewrite.go 行为一致，
 * 并以 sdk/testdata/rewrite-golden.json 做跨语言对拍。全程按字节处理，避免 mb_* 破坏二进制。
 */
final class Rewriter
{
    /** 带前缀/编码形态匹配后，主机名结尾必须是这些边界字符之一，否则不替换（挡 review.comx）。 */
    private const BOUNDARY = "/?#:\"'` \t\n\r<>();,&\\]%=";

    /**
     * 把响应体里的审核站域名替换成客户域名。
     *
     * @param string   $body   原始响应体字节
     * @param string[] $hosts  审核站域名（可多个）
     * @param string   $public 客户对外域名
     * @param string   $scheme 访客协议 http|https
     */
    public static function rewriteBodyHost(string $body, array $hosts, string $public, string $scheme = 'https'): string
    {
        $pub = self::stripPort($public);
        if ($pub === '' || $body === '') {
            return $body;
        }
        if ($scheme === '') {
            $scheme = 'https';
        }
        foreach ($hosts as $oh) {
            $oh = self::stripPort((string) $oh);
            if ($oh === '' || strcasecmp($oh, $pub) === 0) {
                continue;
            }
            $body = self::replaceHostFold($body, 'https://' . $oh, $scheme . '://' . $pub);
            $body = self::replaceHostFold($body, 'http://' . $oh, $scheme . '://' . $pub);
            $body = self::replaceHostFold($body, '//' . $oh, '//' . $pub);
            $body = self::replaceHostFold($body, 'https:\/\/' . $oh, $scheme . ':\/\/' . $pub);
            $body = self::replaceHostFold($body, 'http:\/\/' . $oh, $scheme . ':\/\/' . $pub);
            $body = self::replaceHostFold($body, '\/\/' . $oh, '\/\/' . $pub);
            $body = self::replaceHostFold($body, 'https%3A%2F%2F' . $oh, 'https%3A%2F%2F' . $pub);
            $body = self::replaceHostFold($body, 'http%3A%2F%2F' . $oh, 'https%3A%2F%2F' . $pub);
            $body = self::replaceHostFold($body, '%2F%2F' . $oh, '%2F%2F' . $pub);
            $body = self::replaceBareHost($body, $oh, $pub);
        }
        return $body;
    }

    /** 大小写不敏感替换，且仅在主机名结尾是边界字符时替换。 */
    private static function replaceHostFold(string $body, string $from, string $to): string
    {
        $flen = strlen($from);
        if ($flen === 0 || strlen($body) < $flen) {
            return $body;
        }
        $out = '';
        $i = 0;
        $n = strlen($body);
        while ($i < $n) {
            $j = stripos($body, $from, $i);
            if ($j === false) {
                $out .= substr($body, $i);
                break;
            }
            $end = $j + $flen;
            if (!self::boundaryOK($body, $end)) {
                $out .= substr($body, $i, $end - $i);
                $i = $end;
                continue;
            }
            $out .= substr($body, $i, $j - $i) . $to;
            $i = $end;
        }
        return $out;
    }

    /** 替换正文里裸写的主机名：前不能是字母/数字/连字符/点，后不能是字母/数字/连字符。 */
    private static function replaceBareHost(string $body, string $oh, string $pub): string
    {
        $flen = strlen($oh);
        if ($flen === 0 || strlen($body) < $flen) {
            return $body;
        }
        $out = '';
        $i = 0;
        $n = strlen($body);
        while ($i < $n) {
            $j = stripos($body, $oh, $i);
            if ($j === false) {
                $out .= substr($body, $i);
                break;
            }
            $end = $j + $flen;
            if (!self::bareStartOK($body, $j) || !self::bareEndOK($body, $end)) {
                $out .= substr($body, $i, $end - $i);
                $i = $end;
                continue;
            }
            $out .= substr($body, $i, $j - $i) . $pub;
            $i = $end;
        }
        return $out;
    }

    private static function boundaryOK(string $b, int $i): bool
    {
        if ($i >= strlen($b)) {
            return true;
        }
        return strpos(self::BOUNDARY, $b[$i]) !== false;
    }

    private static function bareStartOK(string $b, int $i): bool
    {
        if ($i === 0) {
            return true;
        }
        $c = $b[$i - 1];
        return !self::isHostByte($c) && $c !== '.';
    }

    private static function bareEndOK(string $b, int $i): bool
    {
        if ($i >= strlen($b)) {
            return true;
        }
        return !self::isHostByte($b[$i]);
    }

    private static function isHostByte(string $c): bool
    {
        return ($c >= 'a' && $c <= 'z') || ($c >= 'A' && $c <= 'Z') || ($c >= '0' && $c <= '9') || $c === '-';
    }

    public static function stripPort(string $h): string
    {
        $i = strrpos($h, ':');
        if ($i !== false && strpos(substr($h, $i), ']') === false) {
            return substr($h, 0, $i);
        }
        return $h;
    }

    /** 深度代理需要改写正文的类型：页面、样式、脚本，及带绝对地址的 JSON/XML。 */
    public static function rewritableContentType(string $contentType): bool
    {
        $ct = strtolower(trim($contentType));
        $semi = strpos($ct, ';');
        if ($semi !== false) {
            $ct = trim(substr($ct, 0, $semi));
        }
        $exact = [
            'text/html', 'application/xhtml+xml',
            'text/css',
            'text/javascript', 'application/javascript', 'application/x-javascript',
            'text/ecmascript', 'application/ecmascript',
            'application/json', 'text/json', 'application/ld+json', 'application/manifest+json',
            'text/xml', 'application/xml', 'application/rss+xml', 'application/atom+xml',
            'image/svg+xml',
        ];
        if (in_array($ct, $exact, true)) {
            return true;
        }
        return substr($ct, -5) === '+json' || substr($ct, -4) === '+xml';
    }

    /**
     * 上游没给 Content-Type（或给了空值）时，按正文嗅探是否文本/HTML，决定要不要改写。
     * 前 1KB 含 NUL 视为二进制不改；以 <!doctype / <html / <?xml / < 开头或可打印文本则改。
     */
    /** gzip 响应先解开，否则正文里的域名是压缩字节，替换不会发生。 */
    public static function inflate(string $body, string $encoding = ''): string
    {
        $gzip = stripos($encoding, 'gzip') !== false
            || (strlen($body) >= 2 && $body[0] === "\x1f" && $body[1] === "\x8b");
        if (!$gzip || !function_exists('gzdecode')) {
            return $body;
        }
        $decoded = gzdecode($body);
        return is_string($decoded) && $decoded !== '' ? $decoded : $body;
    }

    /**
     * 发给访客前压缩文本正文。图片、字体本身已压缩，不再套一层。
     * 小于 1KB 或对方不接受 gzip 时返回 null，调用方原样输出。
     */
    public static function gzipForVisitor(string $body, string $contentType, string $acceptEncoding): ?string
    {
        if ($body === '' || strlen($body) < 1024 || stripos($acceptEncoding, 'gzip') === false || !function_exists('gzencode')) {
            return null;
        }
        if ($contentType !== '') {
            if (!self::rewritableContentType($contentType)) {
                return null;
            }
        } elseif (!self::looksLikeHtml($body)) {
            return null;
        }
        $gz = gzencode($body, 1);
        if (!is_string($gz) || $gz === '' || strlen($gz) >= strlen($body)) {
            return null;
        }
        return $gz;
    }

    /**
     * HTML 同一地址会按访客换成真实页，必须 no-store。
     * 样式、脚本、图片、字体对所有访客相同，允许浏览器缓存一天。
     */
    public static function visitorCacheControl(string $path): string
    {
        $path = strtolower((string) strtok($path, '?'));
        $suffixes = [
            '.css', '.js', '.mjs', '.map',
            '.png', '.jpg', '.jpeg', '.gif', '.webp', '.svg', '.ico',
            '.woff', '.woff2', '.ttf', '.eot', '.otf',
        ];
        foreach ($suffixes as $suffix) {
            if (substr($path, -strlen($suffix)) === $suffix) {
                return 'public, max-age=86400';
            }
        }
        return 'no-store';
    }

    public static function looksLikeHtml(string $body): bool
    {
        $probe = substr($body, 0, 1024);
        if ($probe === '' || strpos($probe, "\0") !== false) {
            return false;
        }
        $head = ltrim($probe);
        if ($head === '') {
            return false;
        }
        return stripos($head, '<!doctype') === 0
            || stripos($head, '<html') === 0
            || stripos($head, '<?xml') === 0
            || $head[0] === '<';
    }
}
