<?php

namespace App\Http\Controllers\Open;

use App\Http\Controllers\Controller;
use App\Services\Api\ApiKeyDecisionService;
use App\Services\Api\ApiKeyService;
use App\Services\Api\ApiLogService;
use App\Support\ApiResponse;
use Illuminate\Http\Request;

class DecisionController extends Controller
{
    public function decide(
        Request $request,
        ApiKeyService $apiKeyService,
        ApiLogService $apiLogService,
        ApiKeyDecisionService $decisionService
    ) {
        $startedAt = microtime(true);
        $apiKey = $this->apiKey($request, $apiKeyService);
        if (!$apiKey) {
            $apiLogService->write($request, null, 'open.decision', 401, 'invalid_api_key', $startedAt);

            return ApiResponse::error('API Key 无效', 401);
        }

        $payload = $request->validate([
            'ip' => ['nullable', 'string', 'max:64'],
            'userAgent' => ['nullable', 'string', 'max:1000'],
            'referer' => ['nullable', 'string', 'max:1000'],
            'acceptLanguage' => ['nullable', 'string', 'max:255'],
        ]);

        $visitorRequest = $decisionService->buildVisitorRequest($request, $payload);
        $result = $decisionService->decide($visitorRequest, $apiKey);
        $apiKeyService->touchCalled($apiKey);
        $apiLogService->write($request, $apiKey, 'open.decision', 200, null, $startedAt);

        return ApiResponse::success($result, '判定成功');
    }

    private function apiKey(Request $request, ApiKeyService $apiKeyService)
    {
        $secret = $request->header('X-API-Key');
        if (!$secret && str_starts_with((string) $request->bearerToken(), 'ck_')) {
            $secret = $request->bearerToken();
        }

        return $apiKeyService->authenticate($secret);
    }
}
