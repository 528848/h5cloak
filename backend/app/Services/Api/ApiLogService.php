<?php

namespace App\Services\Api;

use App\Models\ApiCallLog;
use App\Models\ApiKey;
use Illuminate\Http\Request;
use Illuminate\Support\Str;

class ApiLogService
{
    public function write(Request $request, ?ApiKey $apiKey, string $apiName, int $statusCode, ?string $errorCode, float $startedAt): void
    {
        try {
            ApiCallLog::query()->create([
                'request_id' => (string) Str::uuid(),
                'merchant_id' => $apiKey?->merchant_id,
                'api_key_id' => $apiKey?->id,
                'api_name' => $apiName,
                'method' => $request->method(),
                'path' => $request->path(),
                'request_ip' => $request->ip(),
                'status_code' => $statusCode,
                'error_code' => $errorCode,
                'latency_ms' => (int) ((microtime(true) - $startedAt) * 1000),
                'created_at' => now(),
            ]);
        } catch (\Throwable) {
            // API log failures should not affect the API response path.
        }
    }

    public function paginate(array $filters, ?int $merchantId = null): array
    {
        $pageSize = (int) ($filters['pageSize'] ?? 10);
        $result = ApiCallLog::query()
            ->with(['merchant', 'apiKey'])
            ->when($merchantId, fn ($query) => $query->where('merchant_id', $merchantId))
            ->when($filters['apiName'] ?? null, fn ($query, $apiName) => $query->where('api_name', $apiName))
            ->when($filters['statusCode'] ?? null, fn ($query, $status) => $query->where('status_code', $status))
            ->latest('created_at')
            ->paginate($pageSize);

        return [
            'rows' => $result->getCollection()->map(fn (ApiCallLog $log) => [
                'logId' => $log->id,
                'requestId' => $log->request_id,
                'merchantName' => $log->merchant?->name,
                'apiKeyName' => $log->apiKey?->name,
                'apiName' => $log->api_name,
                'method' => $log->method,
                'path' => $log->path,
                'requestIp' => $log->request_ip,
                'statusCode' => $log->status_code,
                'errorCode' => $log->error_code,
                'latencyMs' => $log->latency_ms,
                'createdAt' => optional($log->created_at)->toDateTimeString(),
            ])->all(),
            'total' => $result->total(),
        ];
    }
}
