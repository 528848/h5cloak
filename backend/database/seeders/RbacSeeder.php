<?php

namespace Database\Seeders;

use App\Models\Menu;
use App\Models\Merchant;
use App\Models\Role;
use App\Models\User;
use Illuminate\Database\Seeder;
use Illuminate\Support\Facades\Hash;

class RbacSeeder extends Seeder
{
    public function run(): void
    {
        $adminRole = Role::query()->updateOrCreate(
            ['role_key' => 'admin'],
            ['name' => '超级管理员', 'scope' => 'platform', 'status' => 'enabled', 'sort' => 1]
        );

        $merchantRole = Role::query()->updateOrCreate(
            ['role_key' => 'merchant_admin'],
            ['name' => '商户管理员', 'scope' => 'merchant', 'status' => 'enabled', 'sort' => 10]
        );

        $admin = User::query()->updateOrCreate(
            ['username' => 'admin'],
            [
                'nickname' => '超级管理员',
                'password' => Hash::make('admin123'),
                'status' => 'enabled',
            ]
        );
        $admin->roles()->syncWithoutDetaching([$adminRole->id]);

        $merchant = Merchant::query()->updateOrCreate(
            ['name' => '演示商户'],
            ['status' => 'enabled', 'domain_quota' => 10, 'cloak_quota' => 50, 'api_quota' => 10000]
        );

        $merchantUser = User::query()->updateOrCreate(
            ['username' => 'merchant'],
            [
                'merchant_id' => $merchant->id,
                'nickname' => '演示商户管理员',
                'password' => Hash::make('merchant123'),
                'status' => 'enabled',
            ]
        );
        $merchantUser->roles()->syncWithoutDetaching([$merchantRole->id]);

        $menus = collect([
            ['name' => '平台首页', 'path' => '/admin/dashboard', 'component' => 'admin/dashboard', 'perms' => 'admin:dashboard:view', 'icon' => 'dashboard', 'sort' => 10, 'roles' => [$adminRole->id]],
            ['name' => '商户账号', 'path' => '/admin/merchant', 'component' => 'admin/merchant', 'perms' => 'merchant:account:list', 'icon' => 'peoples', 'sort' => 20, 'roles' => [$adminRole->id]],
            ['name' => '全部域名', 'path' => '/admin/domain', 'component' => 'admin/domain', 'perms' => 'cloak:domain:list', 'icon' => 'tree', 'sort' => 30, 'roles' => [$adminRole->id]],
            ['name' => '全部斗篷', 'path' => '/admin/cloak', 'component' => 'admin/cloak', 'perms' => 'cloak:strategy:list', 'icon' => 'system', 'sort' => 40, 'roles' => [$adminRole->id]],
            ['name' => '斗篷日志', 'path' => '/admin/cloak-log', 'component' => 'admin/cloak-log', 'perms' => 'cloak:log:list', 'icon' => 'log', 'sort' => 50, 'roles' => [$adminRole->id]],
            ['name' => 'API 调用日志', 'path' => '/admin/api-log', 'component' => 'admin/api-log', 'perms' => 'cloak:apiLog:list', 'icon' => 'log', 'sort' => 60, 'roles' => [$adminRole->id]],
            ['name' => '商户首页', 'path' => '/merchant/dashboard', 'component' => 'merchant/dashboard', 'perms' => 'merchant:dashboard:view', 'icon' => 'dashboard', 'sort' => 110, 'roles' => [$merchantRole->id]],
            ['name' => '域名列表', 'path' => '/merchant/domain', 'component' => 'merchant/domain', 'perms' => 'cloak:domain:list', 'icon' => 'tree', 'sort' => 120, 'roles' => [$merchantRole->id]],
            ['name' => '斗篷列表', 'path' => '/merchant/cloak', 'component' => 'merchant/cloak', 'perms' => 'cloak:strategy:list', 'icon' => 'system', 'sort' => 130, 'roles' => [$merchantRole->id]],
            ['name' => '访问日志', 'path' => '/merchant/cloak-log', 'component' => 'merchant/cloak-log', 'perms' => 'cloak:log:list', 'icon' => 'log', 'sort' => 140, 'roles' => [$merchantRole->id]],
            ['name' => '黑白名单', 'path' => '/merchant/access-list', 'component' => 'merchant/access-list', 'perms' => 'cloak:accessList:list', 'icon' => 'peoples', 'sort' => 150, 'roles' => [$merchantRole->id]],
            ['name' => 'API Key', 'path' => '/merchant/api-key', 'component' => 'merchant/api-key', 'perms' => 'cloak:apiKey:list', 'icon' => 'link', 'sort' => 160, 'roles' => [$merchantRole->id]],
            ['name' => 'API 调用日志', 'path' => '/merchant/api-log', 'component' => 'merchant/api-log', 'perms' => 'cloak:apiLog:list', 'icon' => 'log', 'sort' => 170, 'roles' => [$merchantRole->id]],
        ]);

        $menus->each(function (array $item) {
            $roles = $item['roles'];
            unset($item['roles']);

            $menu = Menu::query()->updateOrCreate(
                ['path' => $item['path']],
                [
                    ...$item,
                    'parent_id' => null,
                    'menu_type' => 'C',
                    'visible' => true,
                    'status' => 'enabled',
                ]
            );
            $menu->roles()->syncWithoutDetaching($roles);
        });

        $this->seedButtonMenus([
            ['parentPath' => '/admin/merchant', 'name' => '商户新增', 'perms' => 'merchant:account:add', 'roles' => [$adminRole->id]],
            ['parentPath' => '/admin/merchant', 'name' => '商户状态', 'perms' => 'merchant:account:status', 'roles' => [$adminRole->id]],
            ['parentPath' => '/merchant/domain', 'name' => '域名新增', 'perms' => 'cloak:domain:add', 'roles' => [$merchantRole->id]],
            ['parentPath' => '/merchant/domain', 'name' => '域名校验', 'perms' => 'cloak:domain:verify', 'roles' => [$merchantRole->id]],
            ['parentPath' => '/merchant/cloak', 'name' => '斗篷新增', 'perms' => 'cloak:strategy:add', 'roles' => [$merchantRole->id]],
            ['parentPath' => '/merchant/cloak', 'name' => '斗篷编辑', 'perms' => 'cloak:strategy:edit', 'roles' => [$merchantRole->id]],
            ['parentPath' => '/merchant/cloak', 'name' => '斗篷复制', 'perms' => 'cloak:strategy:copy', 'roles' => [$merchantRole->id]],
            ['parentPath' => '/merchant/cloak', 'name' => '斗篷删除', 'perms' => 'cloak:strategy:remove', 'roles' => [$merchantRole->id]],
            ['parentPath' => '/merchant/cloak', 'name' => '斗篷状态', 'perms' => 'cloak:strategy:status', 'roles' => [$merchantRole->id]],
            ['parentPath' => '/merchant/access-list', 'name' => '名单新增', 'perms' => 'cloak:accessList:add', 'roles' => [$merchantRole->id]],
            ['parentPath' => '/merchant/access-list', 'name' => '名单编辑', 'perms' => 'cloak:accessList:edit', 'roles' => [$merchantRole->id]],
            ['parentPath' => '/merchant/access-list', 'name' => '名单删除', 'perms' => 'cloak:accessList:remove', 'roles' => [$merchantRole->id]],
            ['parentPath' => '/merchant/api-key', 'name' => 'Key新增', 'perms' => 'cloak:apiKey:add', 'roles' => [$merchantRole->id]],
            ['parentPath' => '/merchant/api-key', 'name' => 'Key重置', 'perms' => 'cloak:apiKey:reset', 'roles' => [$merchantRole->id]],
            ['parentPath' => '/merchant/api-key', 'name' => 'Key删除', 'perms' => 'cloak:apiKey:remove', 'roles' => [$merchantRole->id]],
        ]);
    }

    private function seedButtonMenus(array $buttons): void
    {
        foreach ($buttons as $button) {
            $parent = Menu::query()->where('path', $button['parentPath'])->first();
            $menu = Menu::query()->updateOrCreate(
                ['perms' => $button['perms']],
                [
                    'parent_id' => $parent?->id,
                    'name' => $button['name'],
                    'path' => '',
                    'component' => '',
                    'menu_type' => 'F',
                    'visible' => false,
                    'status' => 'enabled',
                    'sort' => 0,
                ]
            );
            $menu->roles()->syncWithoutDetaching($button['roles']);
        }
    }
}
