<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Services\Cloak\CloakService;
use App\Support\ApiResponse;
use Illuminate\Http\Request;

class CloakController extends Controller
{
    public function index(Request $request, CloakService $cloakService)
    {
        $result = $cloakService->paginateForAdmin($request->all());
        return ApiResponse::page($result['rows'], $result['total']);
    }

    public function logs(Request $request, CloakService $cloakService)
    {
        $result = $cloakService->logs($request->all());
        return ApiResponse::page($result['rows'], $result['total']);
    }

    public function changeStatus(Request $request, CloakService $cloakService)
    {
        $data = $request->validate([
            'cloakId' => ['required', 'integer'],
            'status' => ['required', 'string', 'in:enabled,disabled'],
        ]);

        $cloakService->updateStatus((int) $data['cloakId'], $data['status']);
        return ApiResponse::success(null, '斗篷状态修改成功');
    }
}
