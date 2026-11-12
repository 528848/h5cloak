<?php

if (!defined('ABSPATH')) {
    exit;
}

/**
 * v2 签名。直接继承服务端 SDK 的 \Cloak\Sdk\Signer，保证与 Go/网关完全一致；
 * 保留本类名以兼容既有调用方。
 */
class Cloak_Attr_Signer
{
    public static function headers(string $secret, string $method, string $path, string $body): array
    {
        return \Cloak\Sdk\Signer::headers($secret, $method, $path, $body);
    }
}
