<?php

namespace App\Http\Controllers\Merchant;

use App\Http\Controllers\Controller;
use App\Services\Api\ApiKeyService;
use App\Support\ApiResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;

class ApiKeyController extends Controller
{
    public function index(Request $request, ApiKeyService $apiKeyService)
    {
        $result = $apiKeyService->paginateForMerchant($request->user(), $request->all());
        return ApiResponse::page($result['rows'], $result['total']);
    }

    public function store(Request $request, ApiKeyService $apiKeyService)
    {
        $data = $request->validate([
            'name' => ['required', 'string', 'max:50'],
            'remark' => ['nullable', 'string', 'max:200'],
            'rules' => ['required', 'array'],
            'rules.mode' => ['required', 'string', Rule::in(['cloak', 'green', 'full_open', 'review'])],
            'rules.targetRegions' => ['required_if:rules.mode,cloak', 'array', 'min:1'],
            'rules.targetRegions.*' => ['string', 'size:2'],
            'rules.spiderWhitelist' => ['nullable', 'string', 'max:255'],
            'rules.blockPc' => ['nullable', 'boolean'],
            'rules.blockProxy' => ['nullable', 'boolean'],
        ]);

        [$key, $secret] = $apiKeyService->createForMerchant($request->user(), $data);
        return ApiResponse::success([
            'apiKeyId' => $key->id,
            'keyPrefix' => $key->key_prefix,
            'secret' => $secret,
        ], 'API 斗篷规则创建成功');
    }

    public function changeStatus(Request $request, ApiKeyService $apiKeyService)
    {
        $data = $request->validate([
            'apiKeyId' => ['required', 'integer'],
            'status' => ['required', 'string', 'in:enabled,disabled'],
        ]);

        $apiKeyService->updateStatus((int) $data['apiKeyId'], $request->user()->merchant_id, $data['status']);
        return ApiResponse::success(null, 'API Key 状态修改成功');
    }

    public function reset(Request $request, ApiKeyService $apiKeyService, int $apiKeyId)
    {
        [$key, $secret] = $apiKeyService->resetForMerchant($request->user(), $apiKeyId);

        return ApiResponse::success([
            'apiKeyId' => $key->id,
            'keyPrefix' => $key->key_prefix,
            'secret' => $secret,
        ], 'API Key 重置成功');
    }

    public function destroy(Request $request, ApiKeyService $apiKeyService, int $apiKeyId)
    {
        $apiKeyService->deleteForMerchant($request->user(), $apiKeyId);

        return ApiResponse::success(null, 'API Key 删除成功');
    }
}
