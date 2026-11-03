<?php

namespace App\Services\Api;

use App\Models\ApiKey;
use App\Models\User;
use Illuminate\Support\Facades\Hash;
use Illuminate\Support\Str;

class ApiKeyService
{
    public function paginateForMerchant(User $user, array $filters): array
    {
        $pageSize = (int) ($filters['pageSize'] ?? 10);
        $result = ApiKey::query()
            ->where('merchant_id', $user->merchant_id)
            ->when($filters['name'] ?? null, fn ($query, $name) => $query->where('name', 'like', '%' . $this->escapeLike($name) . '%'))
            ->when($filters['status'] ?? null, fn ($query, $status) => $query->where('status', $status))
            ->when($filters['mode'] ?? null, fn ($query, $mode) => $query->where('rules->mode', $mode))
            ->latest()
            ->paginate($pageSize);

        return [
            'rows' => $result->getCollection()->map(fn (ApiKey $key) => $this->formatListRow($key))->all(),
            'total' => $result->total(),
        ];
    }

    public function createForMerchant(User $user, array $data): array
    {
        $secret = 'ck_' . Str::random(48);
        $rules = ApiKeyRuleConfig::normalize($data['rules'] ?? []);
        $key = ApiKey::query()->create([
            'merchant_id' => $user->merchant_id,
            'name' => $data['name'],
            'key_prefix' => substr($secret, 0, 10),
            'secret_hash' => Hash::make($secret),
            'permissions' => $data['permissions'] ?? ['cloak:decision'],
            'rules' => $rules,
            'remark' => $data['remark'] ?? null,
            'status' => 'enabled',
        ]);

        return [$key, $secret];
    }

    public function updateStatus(int $apiKeyId, int $merchantId, string $status): void
    {
        ApiKey::query()
            ->where('merchant_id', $merchantId)
            ->whereKey($apiKeyId)
            ->update(['status' => $status]);
    }

    public function resetForMerchant(User $user, int $apiKeyId): array
    {
        $key = ApiKey::query()
            ->where('merchant_id', $user->merchant_id)
            ->findOrFail($apiKeyId);

        $secret = 'ck_' . Str::random(48);
        $key->fill([
            'key_prefix' => substr($secret, 0, 10),
            'secret_hash' => Hash::make($secret),
        ])->save();

        return [$key->refresh(), $secret];
    }

    public function deleteForMerchant(User $user, int $apiKeyId): void
    {
        ApiKey::query()
            ->where('merchant_id', $user->merchant_id)
            ->whereKey($apiKeyId)
            ->delete();
    }

    public function authenticate(?string $secret): ?ApiKey
    {
        if (!$secret) {
            return null;
        }

        $prefix = substr($secret, 0, 10);
        return ApiKey::query()
            ->with('merchant')
            ->where('key_prefix', $prefix)
            ->where('status', 'enabled')
            ->get()
            ->first(fn (ApiKey $key) => Hash::check($secret, $key->secret_hash) && $key->merchant?->status === 'enabled');
    }

    public function touchCalled(ApiKey $apiKey): void
    {
        $apiKey->forceFill(['last_called_at' => now()])->save();
    }

    private function formatListRow(ApiKey $key): array
    {
        $rules = ApiKeyRuleConfig::normalize(is_array($key->rules) ? $key->rules : []);

        return [
            'apiKeyId' => $key->id,
            'name' => $key->name,
            'keyPrefix' => $key->key_prefix,
            'mode' => $rules['mode'],
            'modeText' => $this->formatMode($rules['mode']),
            'rules' => $rules,
            'remark' => $key->remark ?? '',
            'permissions' => implode(',', $key->permissions ?? []),
            'status' => $key->status,
            'lastCalledAt' => optional($key->last_called_at)->toDateTimeString(),
            'createdAt' => optional($key->created_at)->toDateTimeString(),
        ];
    }

    private function formatMode(string $mode): string
    {
        return match ($mode) {
            'green' => '绿色',
            'full_open' => '全开',
            'review' => '审核',
            default => '斗篷',
        };
    }

    private function escapeLike(string $value): string
    {
        return str_replace(['\\', '%', '_'], ['\\\\', '\%', '\_'], $value);
    }
}
