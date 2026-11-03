<?php

namespace App\Http\Middleware;

use App\Support\ApiResponse;
use Closure;
use Illuminate\Http\Request;

class EnsureMerchantEnabled
{
    public function handle(Request $request, Closure $next)
    {
        $user = $request->user();
        if (!$user || !$user->merchant_id) {
            return ApiResponse::error('商户身份无效', 403);
        }

        if (!$user->merchant || $user->merchant->status !== 'enabled') {
            return ApiResponse::error('商户已禁用或不可用', 403);
        }

        app()->instance('currentMerchant', $user->merchant);

        return $next($request);
    }
}
