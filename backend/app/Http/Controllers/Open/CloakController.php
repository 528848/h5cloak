<?php

namespace App\Http\Controllers\Open;

use App\Http\Controllers\Controller;
use App\Models\Cloak;
use App\Models\User;
use App\Services\Api\ApiKeyService;
use App\Services\Api\ApiLogService;
use App\Services\Cloak\CloakService;
use App\Support\ApiResponse;
use Illuminate\Http\Request;

class CloakController extends Controller
{
    public function index(Request $request, ApiKeyService $apiKeyService, ApiLogService $apiLogService, CloakService $cloakService)
    {
        $startedAt = microtime(true);
        $apiKey = $this->apiKey($request, $apiKeyService);
        if (!$apiKey) {
            $apiLogService->write($request, null, 'open.cloaks.index', 401, 'invalid_api_key', $startedAt);
            return ApiResponse::error('API Key 无效', 401);
        }

        $apiKeyService->touchCalled($apiKey);
        $result = $cloakService->paginateForMerchant($this->apiUser($apiKey), $request->all());
        $apiLogService->write($request, $apiKey, 'open.cloaks.index', 200, null, $startedAt);
        return ApiResponse::page($result['rows'], $result['total']);
    }

    public function store(Request $request, ApiKeyService $apiKeyService, ApiLogService $apiLogService, CloakService $cloakService)
    {
        $startedAt = microtime(true);
        $apiKey = $this->apiKey($request, $apiKeyService);
        if (!$apiKey) {
            $apiLogService->write($request, null, 'open.cloaks.store', 401, 'invalid_api_key', $startedAt);
            return ApiResponse::error('API Key 无效', 401);
        }

        $data = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'domainId' => ['required', 'integer'],
            'whiteUrl' => ['required', 'url'],
            'blackUrl' => ['required', 'url'],
            'defaultAction' => ['nullable', 'string', 'in:white,black,block,not_found'],
            'rules' => ['nullable', 'array'],
            'status' => ['nullable', 'string', 'in:enabled,disabled'],
            'remark' => ['nullable', 'string', 'max:255'],
        ]);

        $cloak = $cloakService->createForMerchant($this->apiUser($apiKey), $data);
        $apiKeyService->touchCalled($apiKey);
        $apiLogService->write($request, $apiKey, 'open.cloaks.store', 200, null, $startedAt);
        return ApiResponse::success(['cloakId' => $cloak->id], '斗篷创建成功');
    }

    public function show(Request $request, ApiKeyService $apiKeyService, ApiLogService $apiLogService, string $cloakId)
    {
        $startedAt = microtime(true);
        $apiKey = $this->apiKey($request, $apiKeyService);
        if (!$apiKey) {
            $apiLogService->write($request, null, 'open.cloaks.show', 401, 'invalid_api_key', $startedAt);
            return ApiResponse::error('API Key 无效', 401);
        }

        $cloak = Cloak::query()->where('merchant_id', $apiKey->merchant_id)->findOrFail($cloakId);
        $apiKeyService->touchCalled($apiKey);
        $apiLogService->write($request, $apiKey, 'open.cloaks.show', 200, null, $startedAt);
        return ApiResponse::success($cloak);
    }

    public function update(Request $request, ApiKeyService $apiKeyService, ApiLogService $apiLogService, string $cloakId)
    {
        $startedAt = microtime(true);
        $apiKey = $this->apiKey($request, $apiKeyService);
        if (!$apiKey) {
            $apiLogService->write($request, null, 'open.cloaks.update', 401, 'invalid_api_key', $startedAt);
            return ApiResponse::error('API Key 无效', 401);
        }

        $data = $request->validate([
            'name' => ['nullable', 'string', 'max:255'],
            'whiteUrl' => ['nullable', 'url'],
            'blackUrl' => ['nullable', 'url'],
            'defaultAction' => ['nullable', 'string', 'in:white,black,block,not_found'],
            'rules' => ['nullable', 'array'],
            'remark' => ['nullable', 'string', 'max:255'],
        ]);

        $payload = [];
        foreach (['name', 'rules', 'remark'] as $field) {
            if (array_key_exists($field, $data)) {
                $payload[$field] = $data[$field];
            }
        }
        foreach (['whiteUrl' => 'white_url', 'blackUrl' => 'black_url', 'defaultAction' => 'default_action'] as $from => $to) {
            if (array_key_exists($from, $data)) {
                $payload[$to] = $data[$from];
            }
        }

        Cloak::query()->where('merchant_id', $apiKey->merchant_id)->whereKey($cloakId)->update($payload);
        $apiKeyService->touchCalled($apiKey);
        $apiLogService->write($request, $apiKey, 'open.cloaks.update', 200, null, $startedAt);
        return ApiResponse::success(null, '斗篷更新成功');
    }

    public function changeStatus(Request $request, ApiKeyService $apiKeyService, ApiLogService $apiLogService, CloakService $cloakService, string $cloakId)
    {
        $startedAt = microtime(true);
        $apiKey = $this->apiKey($request, $apiKeyService);
        if (!$apiKey) {
            $apiLogService->write($request, null, 'open.cloaks.status', 401, 'invalid_api_key', $startedAt);
            return ApiResponse::error('API Key 无效', 401);
        }

        $data = $request->validate([
            'status' => ['required', 'string', 'in:enabled,disabled'],
        ]);

        $cloakService->updateStatus((int) $cloakId, $data['status'], $apiKey->merchant_id);
        $apiKeyService->touchCalled($apiKey);
        $apiLogService->write($request, $apiKey, 'open.cloaks.status', 200, null, $startedAt);
        return ApiResponse::success(null, '斗篷状态修改成功');
    }

    private function apiKey(Request $request, ApiKeyService $apiKeyService)
    {
        $secret = $request->header('X-API-Key');
        if (!$secret && str_starts_with((string) $request->bearerToken(), 'ck_')) {
            $secret = $request->bearerToken();
        }

        return $apiKeyService->authenticate($secret);
    }

    private function apiUser($apiKey): User
    {
        $user = new User();
        $user->merchant_id = $apiKey->merchant_id;
        return $user;
    }
}
