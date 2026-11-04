<?php

if (!defined('ABSPATH')) {
    exit;
}

/** v2 签名：hex(HMAC-SHA256(secret, ts \n nonce \n METHOD \n path \n hex(sha256(body)))) */
class Cloak_Attr_Signer
{
    public static function headers(string $secret, string $method, string $path, string $body): array
    {
        $ts = (string) time();
        $nonce = bin2hex(random_bytes(16));
        $msg = $ts . "\n" . $nonce . "\n" . strtoupper($method) . "\n" . $path . "\n" . hash('sha256', $body);
        return [
            'Content-Type' => 'application/json',
            'X-Cloak-Key' => substr($secret, 0, 12),
            'X-Cloak-Timestamp' => $ts,
            'X-Cloak-Nonce' => $nonce,
            'X-Cloak-Signature' => hash_hmac('sha256', $msg, $secret),
        ];
    }
}
