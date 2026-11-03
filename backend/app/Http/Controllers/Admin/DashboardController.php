<?php

namespace App\Http\Controllers\Admin;

use App\Http\Controllers\Controller;
use App\Services\Dashboard\DashboardService;
use App\Support\ApiResponse;

class DashboardController extends Controller
{
    public function index(DashboardService $dashboardService)
    {
        return ApiResponse::success($dashboardService->admin());
    }
}
