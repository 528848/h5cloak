<?php

namespace App\Http\Controllers\Merchant;

use App\Http\Controllers\Controller;
use App\Services\Cloak\AccessListService;
use App\Support\ApiResponse;
use Illuminate\Http\Request;

class AccessListController extends Controller
{
    public function index(Request $request, AccessListService $accessListService)
    {
        $result = $accessListService->paginateForMerchant($request->user(), $request->all());
        return ApiResponse::page($result['rows'], $result['total']);
    }

    public function store(Request $request, AccessListService $accessListService)
    {
        $data = $request->validate($this->listRules());

        $item = $accessListService->createForMerchant($request->user(), $data);
        return ApiResponse::success(['listId' => $item->id], '名单添加成功');
    }

    public function update(Request $request, int $listId, AccessListService $accessListService)
    {
        $data = $request->validate($this->listRules());

        $accessListService->updateForMerchant($listId, $request->user()->merchant_id, $data);
        return ApiResponse::success(null, '名单修改成功');
    }

    public function destroy(Request $request, int $listId, AccessListService $accessListService)
    {
        $accessListService->deleteForMerchant($listId, $request->user()->merchant_id);
        return ApiResponse::success(null, '名单删除成功');
    }

    public function changeStatus(Request $request, AccessListService $accessListService)
    {
        $data = $request->validate([
            'listId' => ['required', 'integer'],
            'status' => ['required', 'string', 'in:enabled,disabled'],
        ]);

        $accessListService->updateStatus((int) $data['listId'], $request->user()->merchant_id, $data['status']);
        return ApiResponse::success(null, '名单状态修改成功');
    }

    private function listRules(): array
    {
        return [
            'cloakId' => ['nullable', 'integer'],
            'listType' => ['required', 'string', 'in:white,black'],
            'targetType' => ['required', 'string', 'in:ip,ua,referer'],
            'targetValue' => [
                'required',
                'string',
                'max:255',
                function (string $attribute, mixed $value, \Closure $fail): void {
                    if (request()->input('targetType') !== 'ip') {
                        return;
                    }
                    if (!filter_var(trim((string) $value), FILTER_VALIDATE_IP, FILTER_FLAG_IPV4)) {
                        $fail('IP 格式不正确');
                    }
                },
            ],
            'scope' => ['nullable', 'string', 'in:merchant,cloak'],
            'status' => ['nullable', 'string', 'in:enabled,disabled'],
            'remark' => ['nullable', 'string', 'max:255'],
        ];
    }
}
