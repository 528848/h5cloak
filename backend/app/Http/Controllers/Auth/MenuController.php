<?php

namespace App\Http\Controllers\Auth;

use App\Http\Controllers\Controller;
use App\Services\Permission\PermissionService;
use Illuminate\Http\Request;
use Illuminate\Http\JsonResponse;

class MenuController extends Controller
{
    public function routers(Request $request, PermissionService $permissionService): JsonResponse
    {
        return response()->json([
            'code' => 200,
            'msg' => '查询成功',
            'data' => $permissionService->routers($request->user()),
        ]);
    }
}
