<?php
// 本文件由 sdk/php/src/Browser.php 同步而来，请勿直接修改，改动请回到 SDK 源再同步。

namespace Cloak\Sdk;

if (!defined('ABSPATH')) {
    exit;
}

/**
 * 把 https 地址换成「交给系统默认浏览器打开」的链接。
 * Android 使用 VIEW + BROWSABLE，由系统解析用户设置的默认浏览器，不指定 Chrome。
 * iOS 使用 x-safari-https。其它系统保持原 https 链接。
 */
final class Browser
{
    public static function openHref(string $target, string $os): string
    {
        $os = strtolower($os);
        if ($os === 'ios' && strpos($target, 'https://') === 0) {
            return 'x-safari-' . $target;
        }
        if ($os === 'android') {
            $parts = parse_url($target);
            $host = (string) ($parts['host'] ?? '');
            $scheme = (string) ($parts['scheme'] ?? '');
            if ($host !== '' && ($scheme === 'https' || $scheme === 'http')) {
                $path = (string) ($parts['path'] ?? '/');
                if ($path === '') {
                    $path = '/';
                }
                if (!empty($parts['query'])) {
                    $path .= '?' . $parts['query'];
                }
                if (isset($parts['port'])) {
                    $host .= ':' . $parts['port'];
                }
                return 'intent://' . $host . $path
                    . '#Intent;scheme=' . $scheme
                    . ';action=android.intent.action.VIEW'
                    . ';category=android.intent.category.BROWSABLE'
                    . ';S.browser_fallback_url=' . rawurlencode($target)
                    . ';end';
            }
        }
        return $target;
    }
}
