<?php

namespace Cloak\Sdk;

/**
 * v2 签名：hex(HMAC-SHA256(secret, ts \n nonce \n METHOD \n path \n hex(sha256(body))))。
 * 与网关 internal/transport/http/openapi/auth.go 的 Sign 一致。
 */
final class Signer
{
    /** @return array<string,string> 发往判定接口的签名请求头。 */
    public static function headers(string $secret, string $method, string $path, string $body): array
    {
        $ts = (string) time();
        $nonce = bin2hex(random_bytes(16));
        return [
            'Content-Type' => 'application/json',
            'X-Cloak-Key' => substr($secret, 0, 12),
            'X-Cloak-Timestamp' => $ts,
            'X-Cloak-Nonce' => $nonce,
            'X-Cloak-Signature' => self::sign($secret, $ts, $nonce, $method, $path, $body),
        ];
    }

    public static function sign(string $secret, string $ts, string $nonce, string $method, string $path, string $body): string
    {
        $msg = $ts . "\n" . $nonce . "\n" . strtoupper($method) . "\n" . $path . "\n" . hash('sha256', $body);
        return hash_hmac('sha256', $msg, $secret);
    }
}
