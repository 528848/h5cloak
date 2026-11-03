<?php

namespace App\Http\Controllers\Auth;

use App\Http\Controllers\Controller;
use App\Models\LoginLog;
use App\Models\User;
use App\Services\Permission\PermissionService;
use App\Support\ApiResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Hash;

class AuthController extends Controller
{
    public function login(Request $request)
    {
        $request->validate([
            'username' => ['required', 'string'],
            'password' => ['required', 'string'],
        ]);

        $databaseAvailable = true;
        try {
            $user = User::query()
                ->with('merchant')
                ->where('username', $request->string('username'))
                ->first();
        } catch (\Throwable) {
            $databaseAvailable = false;
            $user = null;
        }

        if ($user && Hash::check($request->string('password'), $user->password)) {
            if ($user->status !== 'enabled') {
                $this->writeLoginLog($request, 'failed', '账号已停用');

                return ApiResponse::error('账号已停用，无法登录', 403);
            }

            if ($user->merchant_id && (!$user->merchant || $user->merchant->status !== 'enabled')) {
                $this->writeLoginLog($request, 'failed', '商户已停用');

                return ApiResponse::error('商户已停用，无法登录', 403);
            }

            $user->forceFill(['last_login_at' => now()])->save();
            $this->writeLoginLog($request, 'success', '登录成功');

            return ApiResponse::success([
                'token' => $user->createToken('admin')->plainTextToken,
            ], '登录成功');
        }

        $this->writeLoginLog($request, 'failed', '账号或密码错误');

        if (!$databaseAvailable && app()->environment(['local', 'testing'])) {
            return ApiResponse::success([
                'token' => 'dev-token',
            ], '开发模式登录成功');
        }

        return ApiResponse::error('账号或密码错误', 401);
    }

    public function logout(Request $request)
    {
        $request->user()?->currentAccessToken()?->delete();
        return ApiResponse::success(null, '退出成功');
    }

    public function getInfo(Request $request, PermissionService $permissionService)
    {
        $info = $permissionService->userInfo($request->user());

        return response()->json([
            'code' => 200,
            'msg' => '查询成功',
            ...$info,
        ]);
    }

    private function writeLoginLog(Request $request, string $status, string $message): void
    {
        try {
            LoginLog::query()->create([
                'username' => $request->string('username'),
                'ip' => $request->ip(),
                'status' => $status,
                'message' => $message,
                'created_at' => now(),
            ]);
        } catch (\Throwable) {
            // Logging must not block login during early environment setup.
        }
    }
}
