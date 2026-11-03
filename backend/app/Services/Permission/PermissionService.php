<?php

namespace App\Services\Permission;

use App\Models\Menu;
use App\Models\User;
use Illuminate\Support\Collection;

class PermissionService
{
    public const SUPER_ADMIN = '*:*:*';

    public function userInfo(?User $user): array
    {
        if (!$user) {
            return $this->devUserInfo();
        }

        $roles = $user->roles()->where('status', 'enabled')->get();
        $permissions = $this->permissions($roles);

        return [
            'user' => [
                'userId' => $user->id,
                'userName' => $user->username,
                'nickName' => $user->nickname,
                'avatar' => $user->avatar,
                'merchantId' => $user->merchant_id,
            ],
            'roles' => $roles->pluck('role_key')->values()->all(),
            'permissions' => $permissions,
        ];
    }

    public function routers(?User $user): array
    {
        if (!$user) {
            return $this->devAdminMenus();
        }

        $roles = $user->roles()->where('status', 'enabled')->get();
        $menus = $roles
            ->flatMap(fn ($role) => $role->menus()->where('status', 'enabled')->get())
            ->unique('id')
            ->sortBy('sort')
            ->values();

        return $this->buildMenuTree($menus);
    }

    private function permissions(Collection $roles): array
    {
        if ($roles->contains('role_key', 'admin')) {
            return [self::SUPER_ADMIN];
        }

        $perms = $roles
            ->flatMap(fn ($role) => $role->menus()->whereNotNull('perms')->pluck('perms'))
            ->filter()
            ->unique()
            ->values()
            ->all();

        if ($roles->contains('role_key', 'merchant_admin')) {
            $perms = array_values(array_unique(array_merge($perms, $this->merchantAdminPermissions())));
        }

        return $perms;
    }

    private function merchantAdminPermissions(): array
    {
        return [
            'merchant:dashboard:view',
            'cloak:domain:list',
            'cloak:domain:add',
            'cloak:domain:verify',
            'cloak:strategy:list',
            'cloak:strategy:add',
            'cloak:strategy:edit',
            'cloak:strategy:copy',
            'cloak:strategy:remove',
            'cloak:strategy:status',
            'cloak:log:list',
            'cloak:accessList:list',
            'cloak:accessList:add',
            'cloak:accessList:edit',
            'cloak:accessList:remove',
            'cloak:apiKey:list',
            'cloak:apiKey:add',
            'cloak:apiKey:reset',
            'cloak:apiKey:remove',
            'cloak:apiLog:list',
        ];
    }

    private function buildMenuTree(Collection $menus, ?int $parentId = null): array
    {
        return $menus
            ->where('parent_id', $parentId)
            ->sortBy('sort')
            ->map(function (Menu $menu) use ($menus) {
                $item = [
                    'name' => $menu->name,
                    'path' => $menu->path,
                    'component' => $menu->component,
                    'query' => $menu->query,
                    'hidden' => !$menu->visible,
                    'meta' => [
                        'title' => $menu->name,
                        'icon' => $menu->icon,
                    ],
                ];

                $children = $this->buildMenuTree($menus, $menu->id);
                if ($children) {
                    $item['children'] = $children;
                }

                return $item;
            })
            ->values()
            ->all();
    }

    private function devUserInfo(): array
    {
        return [
            'user' => [
                'userId' => 1,
                'userName' => 'admin',
                'nickName' => '超级管理员',
                'merchantId' => null,
            ],
            'roles' => ['admin'],
            'permissions' => [self::SUPER_ADMIN],
        ];
    }

    private function devAdminMenus(): array
    {
        return [
            ['name' => '平台首页', 'path' => '/admin/dashboard', 'component' => 'admin/dashboard', 'meta' => ['title' => '平台首页', 'icon' => 'dashboard']],
            ['name' => '商户账号', 'path' => '/admin/merchant', 'component' => 'admin/merchant', 'meta' => ['title' => '商户账号', 'icon' => 'peoples']],
            ['name' => '全部域名', 'path' => '/admin/domain', 'component' => 'admin/domain', 'meta' => ['title' => '全部域名', 'icon' => 'tree']],
            ['name' => '全部斗篷', 'path' => '/admin/cloak', 'component' => 'admin/cloak', 'meta' => ['title' => '全部斗篷', 'icon' => 'system']],
            ['name' => '斗篷日志', 'path' => '/admin/cloak-log', 'component' => 'admin/cloak-log', 'meta' => ['title' => '斗篷日志', 'icon' => 'log']],
            ['name' => 'API 调用日志', 'path' => '/admin/api-log', 'component' => 'admin/api-log', 'meta' => ['title' => 'API 调用日志', 'icon' => 'log']],
        ];
    }
}
