<?php

namespace App\Services\Merchant;

use App\Models\Merchant;
use App\Models\Role;
use App\Models\User;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Hash;

class MerchantService
{
    public function paginate(array $filters): array
    {
        $pageSize = (int) ($filters['pageSize'] ?? 10);
        $loginUrl = $this->merchantLoginUrl();

        $result = Merchant::query()
            ->withCount(['domains'])
            ->with(['users.roles'])
            ->when($filters['name'] ?? null, fn ($query, $name) => $query->where('name', 'like', "%{$name}%"))
            ->when($filters['status'] ?? null, fn ($query, $status) => $query->where('status', $status))
            ->latest()
            ->paginate($pageSize);

        return [
            'rows' => $result->getCollection()->map(fn (Merchant $merchant) => [
                'merchantId' => $merchant->id,
                'merchantName' => $merchant->name,
                'contactName' => $merchant->contact_name,
                'contactEmail' => $merchant->contact_email,
                'contactPhone' => $merchant->contact_phone,
                'status' => $merchant->status,
                'domainCount' => $merchant->domains_count,
                'cloakCount' => 0,
                'loginUrl' => $loginUrl,
                'adminUsername' => $this->resolveAdminUsername($merchant),
                'createdAt' => optional($merchant->created_at)->toDateTimeString(),
            ])->all(),
            'total' => $result->total(),
            'loginUrl' => $loginUrl,
        ];
    }

    public function create(array $data): Merchant
    {
        return DB::transaction(function () use ($data) {
            $merchant = Merchant::query()->create([
                'name' => $data['name'],
                'contact_name' => $data['contactName'] ?? null,
                'contact_email' => $data['contactEmail'] ?? null,
                'contact_phone' => $data['contactPhone'] ?? null,
                'status' => $data['status'] ?? 'enabled',
                'domain_quota' => $data['domainQuota'] ?? 10,
                'cloak_quota' => $data['cloakQuota'] ?? 50,
                'api_quota' => $data['apiQuota'] ?? 10000,
                'remark' => $data['remark'] ?? null,
            ]);

            if (!empty($data['adminUsername']) && !empty($data['adminPassword'])) {
                $role = Role::query()->where('role_key', 'merchant_admin')->first();
                $user = User::query()->create([
                    'merchant_id' => $merchant->id,
                    'username' => $data['adminUsername'],
                    'nickname' => $data['adminNickname'] ?? '商户管理员',
                    'password' => Hash::make($data['adminPassword']),
                    'status' => 'enabled',
                ]);
                if ($role) {
                    $user->roles()->attach($role->id);
                }
            }

            return $merchant;
        });
    }

    public function merchantLoginUrl(): string
    {
        return rtrim((string) config('app.merchant_portal_url', ''), '/') . '/login';
    }

    public function resolveAdminUsername(Merchant $merchant): ?string
    {
        $adminUser = $merchant->users->first(
            fn (User $user) => $user->roles->contains('role_key', 'merchant_admin')
        );

        return $adminUser?->username ?? $merchant->users->first()?->username;
    }

    public function updateStatus(int $merchantId, string $status): void
    {
        Merchant::query()->whereKey($merchantId)->update(['status' => $status]);
    }
}
