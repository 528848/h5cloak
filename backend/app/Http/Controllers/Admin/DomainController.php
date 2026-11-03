<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Services\Domain\DomainService;
use App\Support\ApiResponse;
use Illuminate\Http\Request;

class DomainController extends Controller
{
    public function index(Request $request, DomainService $domainService)
    {
        $result = $domainService->paginateForAdmin($request->all());
        return ApiResponse::page($result['rows'], $result['total']);
    }

    public function changeStatus(Request $request, DomainService $domainService)
    {
        $data = $request->validate([
            'domainId' => ['required', 'integer'],
            'status' => ['required', 'string', 'in:pending,bound,abnormal,disabled'],
        ]);

        $domainService->updateStatus((int) $data['domainId'], $data['status']);
        return ApiResponse::success(null, '域名状态修改成功');
    }
}
