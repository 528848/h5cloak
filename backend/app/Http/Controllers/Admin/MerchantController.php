<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Services\Merchant\MerchantService;
use App\Support\ApiResponse;
use Illuminate\Http\Request;

class MerchantController extends Controller
{
    public function index(Request $request, MerchantService $merchantService)
    {
        $result = $merchantService->paginate($request->all());

        return response()->json([
            'code' => 200,
            'msg' => '查询成功',
            'rows' => $result['rows'],
            'total' => $result['total'],
            'loginUrl' => $result['loginUrl'],
        ]);
    }

    public function store(Request $request, MerchantService $merchantService)
    {
        $data = $request->validate([
            'name' => ['required', 'string', 'max:255'],
            'contactName' => ['nullable', 'string', 'max:255'],
            'contactEmail' => ['nullable', 'email', 'max:255'],
            'contactPhone' => ['nullable', 'string', 'max:64'],
            'adminUsername' => ['nullable', 'string', 'max:255', 'unique:users,username'],
            'adminPassword' => ['nullable', 'string', 'min:6'],
            'status' => ['nullable', 'string'],
            'domainQuota' => ['nullable', 'integer', 'min:0'],
            'cloakQuota' => ['nullable', 'integer', 'min:0'],
            'apiQuota' => ['nullable', 'integer', 'min:0'],
            'remark' => ['nullable', 'string', 'max:255'],
        ]);

        $merchant = $merchantService->create($data);
        $merchant->load('users.roles');

        return ApiResponse::success([
            'merchantId' => $merchant->id,
            'loginUrl' => $merchantService->merchantLoginUrl(),
            'adminUsername' => $merchantService->resolveAdminUsername($merchant)
                ?? $data['adminUsername']
                ?? null,
        ], '商户创建成功');
    }

    public function changeStatus(Request $request, MerchantService $merchantService)
    {
        $data = $request->validate([
            'merchantId' => ['required', 'integer'],
            'status' => ['required', 'string', 'in:enabled,disabled,expired'],
        ]);

        $merchantService->updateStatus((int) $data['merchantId'], $data['status']);
        return ApiResponse::success(null, '状态修改成功');
    }
}
