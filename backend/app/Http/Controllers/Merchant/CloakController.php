<?php

namespace App\Http\Controllers\Merchant;

use App\Http\Controllers\Controller;
use App\Models\Domain;
use App\Services\Cloak\CloakService;
use App\Support\ApiResponse;
use Illuminate\Http\Request;
use Illuminate\Validation\Rule;

class CloakController extends Controller
{
    public function index(Request $request, CloakService $cloakService)
    {
        $result = $cloakService->paginateForMerchant($request->user(), $request->all());
        return ApiResponse::page($result['rows'], $result['total']);
    }

    public function store(Request $request, CloakService $cloakService)
    {
        $data = $this->validateCloakPayload($request);

        Domain::query()
            ->where('merchant_id', $request->user()->merchant_id)
            ->whereKey($data['domainId'])
            ->where('status', 'bound')
            ->firstOrFail();

        $cloak = $cloakService->createForMerchant($request->user(), $data);

        return ApiResponse::success([
            'cloakId' => $cloak->id,
            'shortLinkUrl' => $cloak->rules['shortLinkUrl'] ?? null,
        ], '斗篷创建成功');
    }

    public function update(Request $request, CloakService $cloakService, int $cloakId)
    {
        $data = $this->validateCloakPayload($request);

        Domain::query()
            ->where('merchant_id', $request->user()->merchant_id)
            ->whereKey($data['domainId'])
            ->where('status', 'bound')
            ->firstOrFail();

        $cloak = $cloakService->updateForMerchant($request->user(), $cloakId, $data);

        return ApiResponse::success([
            'cloakId' => $cloak->id,
            'shortLinkUrl' => $cloak->rules['shortLinkUrl'] ?? null,
        ], '斗篷修改成功');
    }

    public function copy(Request $request, CloakService $cloakService, int $cloakId)
    {
        $cloak = $cloakService->copyForMerchant($request->user(), $cloakId);

        return ApiResponse::success([
            'cloakId' => $cloak->id,
            'shortLinkUrl' => $cloak->rules['shortLinkUrl'] ?? null,
        ], '斗篷复制成功');
    }

    public function destroy(Request $request, CloakService $cloakService, int $cloakId)
    {
        $cloakService->deleteForMerchant($request->user(), $cloakId);

        return ApiResponse::success(null, '斗篷删除成功');
    }

    public function changeStatus(Request $request, CloakService $cloakService)
    {
        $data = $request->validate([
            'cloakId' => ['required', 'integer'],
            'status' => ['required', 'string', 'in:enabled,disabled'],
        ]);

        $cloakService->updateStatus((int) $data['cloakId'], $data['status'], $request->user()->merchant_id);
        return ApiResponse::success(null, '斗篷状态修改成功');
    }

    public function logs(Request $request, CloakService $cloakService)
    {
        $result = $cloakService->logs($request->all(), $request->user()->merchant_id);
        return ApiResponse::page($result['rows'], $result['total']);
    }

    private function validateCloakPayload(Request $request): array
    {
        $data = $request->validate([
            'name' => ['required', 'string', 'max:50'],
            'domainId' => ['required', 'integer'],
            'whiteUrl' => ['required', 'url', 'max:1000'],
            'blackUrl' => ['required', 'url', 'max:1000'],
            'defaultAction' => ['nullable', 'string', 'in:white,black,block,not_found'],
            'status' => ['nullable', 'string', 'in:enabled,disabled'],
            'remark' => ['nullable', 'string', 'max:200'],
            'rules' => ['nullable', 'array'],
            'rules.reviewPageMode' => ['nullable', 'string', Rule::in(['external', 'local_template'])],
            'rules.realUrls' => ['nullable', 'array'],
            'rules.realUrls.*' => ['url', 'max:1000'],
            'rules.shortLinkType' => ['nullable', 'string', Rule::in(['system', 'custom'])],
            'rules.shortLinkUrl' => ['nullable', 'url', 'max:1000'],
            'rules.allowedSources' => ['nullable', 'array'],
            'rules.allowedSources.*' => ['string', Rule::in(['facebook', 'google', 'tiktok', 'kwai', 'all'])],
            'rules.mode' => ['nullable', 'string', Rule::in(['cloak', 'green', 'full_open', 'review'])],
            'rules.targetRegions' => ['required', 'array', 'min:1'],
            'rules.targetRegions.*' => ['string', 'size:2'],
            'rules.spiderWhitelist' => ['nullable', 'string', 'max:255'],
            'rules.deviceWhitelist' => ['nullable', 'string', 'max:255'],
            'rules.blockEmptyLanguage' => ['nullable', 'boolean'],
            'rules.blockPc' => ['nullable', 'boolean'],
            'rules.blockProxy' => ['nullable', 'boolean'],
            'rules.intermediatePage' => ['nullable', 'string', Rule::in(['none', 'system_template', 'custom_template'])],
            'rules.reviewQuickTag' => ['nullable', 'string', 'max:64'],
        ]);

        $data['rules'] = $data['rules'] ?? [];

        return $data;
    }
}
