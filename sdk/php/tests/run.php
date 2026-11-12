<?php
/**
 * 纯 PHP 跑测（不依赖 PHPUnit）：改写黄金用例 + 签名已知答案向量 + Content-Type 判定。
 *
 *   php -n tests/run.php
 */

require __DIR__ . '/../src/Rewriter.php';
require __DIR__ . '/../src/Signer.php';
require __DIR__ . '/../src/ResponseCache.php';
require __DIR__ . '/../src/Browser.php';

use Cloak\Sdk\Browser;
use Cloak\Sdk\ResponseCache;
use Cloak\Sdk\Rewriter;
use Cloak\Sdk\Signer;

$fail = 0;
$pass = 0;

function check($name, $got, $expect)
{
    global $fail, $pass;
    if ($got === $expect) {
        $pass++;
        return;
    }
    $fail++;
    fwrite(STDERR, "FAIL $name\n  expect: " . var_export($expect, true) . "\n  got:    " . var_export($got, true) . "\n");
}

$testdata = __DIR__ . '/../../testdata';

// 1. 改写黄金用例集（跨语言共享）
$golden = json_decode(file_get_contents($testdata . '/rewrite-golden.json'), true);
foreach ($golden['cases'] as $c) {
    $got = Rewriter::rewriteBodyHost($c['input'], $c['originHosts'], $c['publicHost'], $c['scheme']);
    check('rewrite:' . $c['name'], $got, $c['expect']);
}

// 2. 签名已知答案向量（跨语言共享）
$sg = json_decode(file_get_contents($testdata . '/sign-golden.json'), true);
$sig = Signer::sign($sg['secret'], $sg['ts'], $sg['nonce'], $sg['method'], $sg['path'], $sg['body']);
check('sign:golden', $sig, $sg['signature']);

// 3. Content-Type 判定
foreach (['text/html', 'text/html; charset=utf-8', 'application/json', 'application/ld+json', 'image/svg+xml', 'application/vnd.api+json'] as $ct) {
    check('ct:yes:' . $ct, Rewriter::rewritableContentType($ct), true);
}
foreach (['image/png', 'application/octet-stream', 'font/woff2', 'video/mp4', ''] as $ct) {
    check('ct:no:' . $ct, Rewriter::rewritableContentType($ct), false);
}

// 4. Content-Type 缺失时的正文嗅探
check('sniff:doctype', Rewriter::looksLikeHtml("<!DOCTYPE html><html></html>"), true);
check('sniff:leading-space', Rewriter::looksLikeHtml("  \n<html>hi</html>"), true);
check('sniff:binary', Rewriter::looksLikeHtml("PNG\0\0binary"), false);
check('sniff:empty', Rewriter::looksLikeHtml(''), false);
check('sniff:plain-start-lt', Rewriter::looksLikeHtml('<rss>...</rss>'), true);

// 5. gzip 先解开再改写，压缩正文里的域名也能替换
$gz = gzencode('<a href="https://toptime.cc/index.php/product-category/rolex/">ROLEX</a><title>toptime.cc</title>');
$plain = Rewriter::inflate($gz, 'gzip');
$rewritten = Rewriter::rewriteBodyHost($plain, ['toptime.cc'], 'iris6666.com', 'https');
check('gzip:inflate-rewrite', $rewritten, '<a href="https://iris6666.com/index.php/product-category/rolex/">ROLEX</a><title>iris6666.com</title>');

// 6. 深度代理缓存：键、过期、清理、个人请求不命中
$cacheDir = sys_get_temp_dir() . '/cloak-sdk-cache-' . bin2hex(random_bytes(4));
$cache = new ResponseCache($cacheDir, 60);
$k1 = ResponseCache::key('GET', 'toptime.cc', 'iris6666.com', 'https', '/', 'utm_source=tiktok&ver=1');
$k2 = ResponseCache::key('GET', 'toptime.cc', 'iris6666.com', 'https', '/', 'ver=1');
$k3 = ResponseCache::key('GET', 'toptime.cc', 'iris6666.com', 'https', '/product/', 'ver=1');
check('cache:tracking-query-ignored', $k1, $k2);
check('cache:path-changes-key', $k1 === $k3, false);
$now = time();
check('cache:put', $cache->put($k1, 200, 'text/html', '<html>cached</html>', $now), true);
$hit = $cache->get($k1, $now + 10);
check('cache:hit-body', is_array($hit) ? $hit['body'] : '', '<html>cached</html>');
check('cache:hit-type', is_array($hit) ? $hit['contentType'] : '', 'text/html');
check('cache:skip-non-200', $cache->put($k3, 404, 'text/html', 'no', $now), false);
check('cache:skip-empty', $cache->put($k3, 200, 'text/html', '', $now), false);
$_SERVER['HTTP_COOKIE'] = 'wordpress_logged_in_abc=1';
check('cache:personal-miss', $cache->get($k1, $now + 10), null);
unset($_SERVER['HTTP_COOKIE']);
check('cache:expired', $cache->get($k1, $now + 61), null);
check('cache:put-again', $cache->put($k1, 200, 'text/css', 'body{}', $now), true);
$removed = $cache->purge();
check('cache:purge', $removed > 0, true);
check('cache:after-purge', $cache->get($k1, $now + 10), null);
@rmdir($cacheDir);

$android = Browser::openHref('https://iris6666.com/a?x=1', 'Android');
check('browser:android-default', strpos($android, 'intent://iris6666.com/a?x=1#Intent;scheme=https;action=android.intent.action.VIEW;category=android.intent.category.BROWSABLE;') === 0, true);
check('browser:android-fallback', strpos($android, 'S.browser_fallback_url=https%3A%2F%2Firis6666.com%2Fa%3Fx%3D1;end') !== false, true);
check('browser:ios', Browser::openHref('https://iris6666.com/a', 'ios'), 'x-safari-https://iris6666.com/a');
check('browser:other', Browser::openHref('https://iris6666.com/a', 'windows'), 'https://iris6666.com/a');

echo "PHP SDK: $pass passed, $fail failed\n";
exit($fail === 0 ? 0 : 1);
