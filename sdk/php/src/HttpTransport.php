<?php

namespace Cloak\Sdk;

/** 回源/判定的 HTTP 响应。headers 为保序的 [name, value] 列表，可含重复 Set-Cookie。 */
final class HttpResponse
{
    public int $status;
    /** @var array<int,array{0:string,1:string}> */
    public array $headers;
    public string $body;

    /** @param array<int,array{0:string,1:string}> $headers */
    public function __construct(int $status, array $headers, string $body)
    {
        $this->status = $status;
        $this->headers = $headers;
        $this->body = $body;
    }
}

/**
 * HTTP 传输抽象。默认 CurlTransport；在 WordPress 等环境里可注入 wp_remote_* 适配，
 * 这样 SDK 的判定与深度代理逻辑在任何框架内都复用同一套代码。
 */
interface HttpTransport
{
    /**
     * @param array<string,string> $headers
     * @param string|null          $body      请求体；GET/HEAD 传 null
     * @param int                  $maxBytes  响应体读取上限；<=0 不限制
     * @param string               $sni       TLS SNI（回源 Host 为域名且与 URL 主机不同时）
     */
    public function request(string $method, string $url, array $headers, ?string $body, int $timeout, int $maxBytes = 0, string $sni = ''): ?HttpResponse;
}
