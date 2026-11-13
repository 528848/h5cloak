<?php

namespace Cloak\Sdk;

/** 基于 ext-curl 的默认 HTTP 传输。 */
final class CurlTransport implements HttpTransport
{
    public function request(string $method, string $url, array $headers, ?string $body, int $timeout, int $maxBytes = 0, string $sni = ''): ?HttpResponse
    {
        $ch = curl_init();
        $hdrList = [];
        $callerEncoding = false;
        foreach ($headers as $k => $v) {
            $hdrList[] = $k . ': ' . $v;
            if (strcasecmp((string) $k, 'Accept-Encoding') === 0) {
                $callerEncoding = true;
            }
        }
        $respHeaders = [];
        $opts = [
            CURLOPT_URL => $url,
            CURLOPT_CUSTOMREQUEST => strtoupper($method),
            CURLOPT_HTTPHEADER => $hdrList,
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_FOLLOWLOCATION => false,
            CURLOPT_TIMEOUT => max(1, $timeout),
            CURLOPT_CONNECTTIMEOUT => max(1, $timeout),
            CURLOPT_HEADERFUNCTION => function ($c, $line) use (&$respHeaders) {
                $len = strlen($line);
                $p = strpos($line, ':');
                if ($p !== false) {
                    $respHeaders[] = [trim(substr($line, 0, $p)), trim(substr($line, $p + 1))];
                }
                return $len;
            },
        ];
        // 调用方要自己解 gzip 时不要让 curl 再解码一次。其它请求保持明文。
        if (!$callerEncoding) {
            $opts[CURLOPT_ENCODING] = 'identity';
        }
        if ($body !== null) {
            $opts[CURLOPT_POSTFIELDS] = $body;
        }
        // 回源 Host 为域名但连的是另一地址时，用 SNI + Host 头（Host 头已由调用方放进 $headers）
        if ($sni !== '') {
            // curl 以 URL 主机做 SNI；此处 URL 主机即审核站域名时无需特殊处理。
            // 若将来支持 IP 回源，可用 CURLOPT_RESOLVE / CURLOPT_CONNECT_TO 指定。
        }
        if ($maxBytes > 0) {
            $received = 0;
            $buf = '';
            $opts[CURLOPT_WRITEFUNCTION] = function ($c, $chunk) use (&$received, &$buf, $maxBytes) {
                $received += strlen($chunk);
                if ($received <= $maxBytes) {
                    $buf .= $chunk;
                } else {
                    // 仍需返回实际写入长度，否则 curl 判为写错误中断
                    $buf .= $chunk;
                }
                return strlen($chunk);
            };
            curl_setopt_array($ch, $opts);
            $ok = curl_exec($ch);
            $status = (int) curl_getinfo($ch, CURLINFO_RESPONSE_CODE);
            $err = curl_errno($ch);
            curl_close($ch);
            if ($ok === false || $err !== 0 || $status === 0) {
                return null;
            }
            return new HttpResponse($status, $respHeaders, $buf);
        }

        curl_setopt_array($ch, $opts);
        $out = curl_exec($ch);
        $status = (int) curl_getinfo($ch, CURLINFO_RESPONSE_CODE);
        $err = curl_errno($ch);
        curl_close($ch);
        if ($out === false || $err !== 0 || $status === 0) {
            return null;
        }
        return new HttpResponse($status, $respHeaders, (string) $out);
    }
}
