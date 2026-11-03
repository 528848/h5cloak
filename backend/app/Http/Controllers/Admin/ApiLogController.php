<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Services\Api\ApiLogService;
use App\Support\ApiResponse;
use Illuminate\Http\Request;

class ApiLogController extends Controller
{
    public function index(Request $request, ApiLogService $apiLogService)
    {
        $result = $apiLogService->paginate($request->all());
        return ApiResponse::page($result['rows'], $result['total']);
    }
}
