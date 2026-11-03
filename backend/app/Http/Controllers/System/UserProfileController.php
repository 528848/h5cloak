<?php

namespace App\Http\Controllers\System;

use App\Http\Controllers\Controller;
use App\Models\User;
use App\Support\ApiResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Hash;
use Illuminate\Support\Facades\Storage;
use Illuminate\Support\Str;

class UserProfileController extends Controller
{
    public function show(Request $request)
    {
        $user = $request->user()->load('roles');

        return response()->json([
            'code' => 200,
            'msg' => '查询成功',
            'data' => $this->formatProfile($user),
            'roleGroup' => $user->roles->pluck('name')->implode('、') ?: '-',
            'postGroup' => '-',
        ]);
    }

    public function update(Request $request)
    {
        $user = $request->user();
        $data = $request->validate([
            'nickName' => ['required', 'string', 'max:255'],
            'phonenumber' => ['nullable', 'string', 'max:32'],
            'email' => ['nullable', 'email', 'max:255'],
            'sex' => ['nullable', 'string', 'max:8'],
        ]);

        $user->update([
            'nickname' => $data['nickName'],
            'phone' => $data['phonenumber'] ?? null,
            'email' => $data['email'] ?? null,
            'sex' => $data['sex'] ?? null,
        ]);

        return ApiResponse::success(null, '修改成功');
    }

    public function updateAvatar(Request $request)
    {
        $request->validate([
            'avatarfile' => ['required', 'file', 'image', 'max:2048'],
        ]);

        $user = $request->user();
        $file = $request->file('avatarfile');
        $filename = Str::uuid() . '.' . $file->getClientOriginalExtension();
        $path = $file->storeAs('avatars', $filename, 'public');

        if ($user->avatar) {
            $oldPath = parse_url($user->avatar, PHP_URL_PATH) ?: '';
            $oldPath = ltrim(str_replace('/storage/', '', $oldPath), '/');
            if ($oldPath && Storage::disk('public')->exists($oldPath)) {
                Storage::disk('public')->delete($oldPath);
            }
        }

        $imgUrl = url('storage/' . $path);
        $user->update(['avatar' => $imgUrl]);

        return response()->json([
            'code' => 200,
            'msg' => '上传成功',
            'imgUrl' => $imgUrl,
        ]);
    }

    public function updatePassword(Request $request)
    {
        $data = $request->validate([
            'oldPassword' => ['required', 'string'],
            'newPassword' => ['required', 'string', 'min:6'],
        ]);

        $user = $request->user();
        if (!Hash::check($data['oldPassword'], $user->password)) {
            return ApiResponse::error('旧密码错误', 400);
        }

        $user->update([
            'password' => Hash::make($data['newPassword']),
        ]);

        return ApiResponse::success(null, '修改成功');
    }

    private function formatProfile(User $user): array
    {
        return [
            'userId' => $user->id,
            'userName' => $user->username,
            'nickName' => $user->nickname,
            'phonenumber' => $user->phone,
            'email' => $user->email,
            'sex' => $user->sex,
            'avatar' => $user->avatar,
        ];
    }
}
