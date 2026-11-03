<?php

namespace App\Services\Cloak;

use App\Models\AccessList;
use App\Models\User;

class AccessListService
{
    public function paginateForMerchant(User $user, array $filters): array
    {
        $pageSize = (int) ($filters['pageSize'] ?? 10);
        $result = AccessList::query()
            ->with('cloak')
            ->where('merchant_id', $user->merchant_id)
            ->when($filters['listType'] ?? null, fn ($query, $type) => $query->where('list_type', $type))
            ->when($filters['targetType'] ?? null, fn ($query, $type) => $query->where('target_type', $type))
            ->when($filters['targetValue'] ?? null, function ($query, $value) {
                $value = trim((string) $value);
                if ($value !== '') {
                    $query->where('target_value', 'like', '%' . $value . '%')
                        ->where('target_type', 'ip');
                }
            })
            ->when($filters['status'] ?? null, fn ($query, $status) => $query->where('status', $status))
            ->latest()
            ->paginate($pageSize);

        return [
            'rows' => $result->getCollection()->map(fn (AccessList $item) => [
                'listId' => $item->id,
                'cloakId' => $item->cloak_id,
                'cloakName' => $item->cloak?->name,
                'listType' => $item->list_type,
                'targetType' => $item->target_type,
                'targetValue' => $item->target_value,
                'scope' => $item->scope,
                'status' => $item->status,
                'remark' => $item->remark,
                'createdAt' => optional($item->created_at)->toDateTimeString(),
            ])->all(),
            'total' => $result->total(),
        ];
    }

    public function createForMerchant(User $user, array $data): AccessList
    {
        return AccessList::query()->create([
            'merchant_id' => $user->merchant_id,
            'cloak_id' => ($data['scope'] ?? 'merchant') === 'cloak' ? ($data['cloakId'] ?? null) : null,
            'list_type' => $data['listType'],
            'target_type' => $data['targetType'],
            'target_value' => $data['targetValue'],
            'scope' => $data['scope'] ?? 'merchant',
            'status' => $data['status'] ?? 'enabled',
            'remark' => $data['remark'] ?? null,
        ]);
    }

    public function updateForMerchant(int $listId, int $merchantId, array $data): void
    {
        AccessList::query()
            ->where('merchant_id', $merchantId)
            ->whereKey($listId)
            ->update([
                'cloak_id' => ($data['scope'] ?? 'merchant') === 'cloak' ? ($data['cloakId'] ?? null) : null,
                'list_type' => $data['listType'],
                'target_type' => $data['targetType'],
                'target_value' => $data['targetValue'],
                'scope' => $data['scope'] ?? 'merchant',
                'status' => $data['status'] ?? 'enabled',
                'remark' => $data['remark'] ?? null,
            ]);
    }

    public function deleteForMerchant(int $listId, int $merchantId): void
    {
        AccessList::query()
            ->where('merchant_id', $merchantId)
            ->whereKey($listId)
            ->delete();
    }

    public function updateStatus(int $listId, int $merchantId, string $status): void
    {
        AccessList::query()
            ->where('merchant_id', $merchantId)
            ->whereKey($listId)
            ->update(['status' => $status]);
    }
}
