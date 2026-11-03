<?php

namespace App\Services\Cloak;

use App\Models\Cloak;
use App\Models\CloakAccessLog;
use App\Models\Domain;
use App\Support\CountryRegionLabels;
use App\Models\User;

class CloakService
{
    public function paginateForAdmin(array $filters): array
    {
        return $this->paginate($filters);
    }

    public function paginateForMerchant(User $user, array $filters): array
    {
        return $this->paginate($filters, $user->merchant_id);
    }

    public function createForMerchant(User $user, array $data): Cloak
    {
        $domain = Domain::query()
            ->where('merchant_id', $user->merchant_id)
            ->findOrFail($data['domainId']);

        $rules = CloakRuleConfig::normalize($data, $domain->domain);
        $realUrls = $rules['realUrls'] ?: [$data['blackUrl']];

        return Cloak::query()->create([
            'merchant_id' => $user->merchant_id,
            'domain_id' => $data['domainId'],
            'name' => $data['name'],
            'status' => $data['status'] ?? 'disabled',
            'white_url' => $data['whiteUrl'],
            'black_url' => $realUrls[0],
            'default_action' => $data['defaultAction'] ?? 'block',
            'rules' => $rules,
            'remark' => $data['remark'] ?? null,
        ]);
    }

    public function updateStatus(int $cloakId, string $status, ?int $merchantId = null): void
    {
        Cloak::query()
            ->when($merchantId, fn ($query) => $query->where('merchant_id', $merchantId))
            ->whereKey($cloakId)
            ->update(['status' => $status]);
    }

    public function updateForMerchant(User $user, int $cloakId, array $data): Cloak
    {
        $cloak = Cloak::query()
            ->where('merchant_id', $user->merchant_id)
            ->findOrFail($cloakId);

        $domain = Domain::query()
            ->where('merchant_id', $user->merchant_id)
            ->findOrFail($data['domainId']);

        $inputRules = $data['rules'] ?? [];
        if (!empty($cloak->rules['shortLinkUrl']) && ($inputRules['shortLinkType'] ?? 'system') === 'system') {
            $inputRules['shortLinkUrl'] = $cloak->rules['shortLinkUrl'];
        }

        $data['rules'] = $inputRules;
        $rules = CloakRuleConfig::normalize($data, $domain->domain);
        $realUrls = $rules['realUrls'] ?: [$data['blackUrl']];

        $cloak->fill([
            'domain_id' => $data['domainId'],
            'name' => $data['name'],
            'status' => $data['status'] ?? $cloak->status,
            'white_url' => $data['whiteUrl'],
            'black_url' => $realUrls[0],
            'default_action' => $data['defaultAction'] ?? $cloak->default_action,
            'rules' => $rules,
            'remark' => $data['remark'] ?? null,
        ])->save();

        return $cloak->refresh();
    }

    public function copyForMerchant(User $user, int $cloakId): Cloak
    {
        $source = Cloak::query()
            ->where('merchant_id', $user->merchant_id)
            ->findOrFail($cloakId);

        $rules = is_array($source->rules) ? $source->rules : [];
        unset($rules['shortLinkUrl']);

        return $this->createForMerchant($user, [
            'domainId' => $source->domain_id,
            'name' => $this->makeCopyName($source->name, $user->merchant_id),
            'whiteUrl' => $source->white_url,
            'blackUrl' => $source->black_url,
            'defaultAction' => $source->default_action,
            'status' => 'disabled',
            'remark' => $source->remark,
            'rules' => array_merge($rules, ['shortLinkType' => $rules['shortLinkType'] ?? 'system']),
        ]);
    }

    public function deleteForMerchant(User $user, int $cloakId): void
    {
        Cloak::query()
            ->where('merchant_id', $user->merchant_id)
            ->whereKey($cloakId)
            ->delete();
    }

    public function logs(array $filters, ?int $merchantId = null): array
    {
        $pageSize = (int) ($filters['pageSize'] ?? 10);
        $result = CloakAccessLog::query()
            ->with(['merchant', 'domain', 'cloak'])
            ->when($merchantId, fn ($query) => $query->where('merchant_id', $merchantId))
            ->when($filters['domainId'] ?? null, fn ($query, $domainId) => $query->where('domain_id', $domainId))
            ->when($filters['cloakId'] ?? null, fn ($query, $cloakId) => $query->where('cloak_id', $cloakId))
            ->when($filters['finalAction'] ?? null, fn ($query, $action) => $query->where('final_action', $action))
            ->latest('created_at')
            ->paginate($pageSize);

        return [
            'rows' => $result->getCollection()->map(fn (CloakAccessLog $log) => [
                'logId' => $log->id,
                'requestId' => $log->request_id,
                'merchantName' => $log->merchant?->name,
                'domain' => $log->domain?->domain,
                'cloakName' => $log->cloak?->name,
                'ipSummary' => $log->ip_hash,
                'country' => $log->country,
                'region' => $log->region,
                'deviceType' => $log->device_type,
                'hitRule' => $log->hit_rule,
                'finalAction' => $log->final_action,
                'latencyMs' => $log->latency_ms,
                'createdAt' => optional($log->created_at)->toDateTimeString(),
            ])->all(),
            'total' => $result->total(),
        ];
    }

    private function paginate(array $filters, ?int $merchantId = null): array
    {
        $pageSize = (int) ($filters['pageSize'] ?? 10);
        $today = now()->startOfDay();
        $result = Cloak::query()
            ->with(['merchant', 'domain'])
            ->withCount(['accessLists'])
            ->withCount([
                'accessLogs as whiteHits' => fn ($query) => $query->where('final_action', 'white'),
                'accessLogs as blackHits' => fn ($query) => $query->where('final_action', 'black'),
                'accessLogs as todayWhiteHits' => fn ($query) => $query
                    ->where('final_action', 'white')
                    ->where('created_at', '>=', $today),
                'accessLogs as todayBlackHits' => fn ($query) => $query
                    ->where('final_action', 'black')
                    ->where('created_at', '>=', $today),
            ])
            ->when($merchantId, fn ($query) => $query->where('merchant_id', $merchantId))
            ->when($filters['name'] ?? null, fn ($query, $name) => $query->where('name', 'like', '%' . $this->escapeLike($name) . '%'))
            ->when($filters['shortLinkUrl'] ?? null, function ($query, $value) {
                $value = trim((string) $value);
                if ($value !== '') {
                    $query->where('rules->shortLinkUrl', 'like', '%' . $this->escapeLike($value) . '%');
                }
            })
            ->when($filters['blackUrl'] ?? null, function ($query, $value) {
                $value = trim((string) $value);
                if ($value !== '') {
                    $like = '%' . $this->escapeLike($value) . '%';
                    $query->where(function ($subQuery) use ($like) {
                        $subQuery->where('black_url', 'like', $like)
                            ->orWhere('rules', 'like', $like);
                    });
                }
            })
            ->when($filters['mode'] ?? null, fn ($query, $mode) => $query->where('rules->mode', $mode))
            ->when($filters['domainId'] ?? null, fn ($query, $domainId) => $query->where('domain_id', $domainId))
            ->when($filters['status'] ?? null, fn ($query, $status) => $query->where('status', $status))
            ->latest()
            ->paginate($pageSize);

        return [
            'rows' => $result->getCollection()->map(fn (Cloak $cloak) => $this->formatListRow($cloak))->all(),
            'total' => $result->total(),
        ];
    }

    private function formatListRow(Cloak $cloak): array
    {
        $rules = is_array($cloak->rules) ? $cloak->rules : [];
        $targetRegions = $rules['targetRegions'] ?? [];

        return [
            'cloakId' => $cloak->id,
            'merchantId' => $cloak->merchant_id,
            'merchantName' => $cloak->merchant?->name,
            'domainId' => $cloak->domain_id,
            'domain' => $cloak->domain?->domain,
            'cloakName' => $cloak->name,
            'status' => $cloak->status,
            'whiteUrl' => $cloak->white_url,
            'blackUrl' => $cloak->black_url,
            'realUrls' => $rules['realUrls'] ?? [$cloak->black_url],
            'defaultAction' => $cloak->default_action,
            'rules' => $rules,
            'mode' => $rules['mode'] ?? 'cloak',
            'shortLinkUrl' => $rules['shortLinkUrl'] ?? null,
            'targetRegions' => $targetRegions,
            'targetRegionsText' => CountryRegionLabels::format($targetRegions),
            'remark' => $cloak->remark ?? '',
            'deviceWhitelist' => $this->formatDeviceWhitelist($rules['deviceWhitelist'] ?? ''),
            'blockPc' => !empty($rules['blockPc']) ? '开启' : '未开启',
            'blockProxy' => !empty($rules['blockProxy']) ? '开启' : '未开启',
            'allowedSourcesText' => $this->formatAllowedSources($rules['allowedSources'] ?? ['all']),
            'intermediatePage' => $this->formatIntermediatePage($rules['intermediatePage'] ?? 'none'),
            'intermediatePageContent' => '-',
            'whiteHits' => (int) ($cloak->whiteHits ?? 0),
            'blackHits' => (int) ($cloak->blackHits ?? 0),
            'todayWhiteHits' => (int) ($cloak->todayWhiteHits ?? 0),
            'todayBlackHits' => (int) ($cloak->todayBlackHits ?? 0),
            'todayVisits' => (int) (($cloak->todayWhiteHits ?? 0) + ($cloak->todayBlackHits ?? 0)),
            'accessListCount' => $cloak->access_lists_count,
            'updatedAt' => optional($cloak->updated_at)->toDateTimeString(),
            'createdAt' => optional($cloak->created_at)->toDateTimeString(),
        ];
    }

    private function formatDeviceWhitelist(string $value): string
    {
        return match ($value) {
            'mobile' => '仅移动端',
            'tablet' => '仅平板',
            'pc' => '仅 PC',
            'mobile,tablet' => '移动端 + 平板',
            default => '-',
        };
    }

    private function formatAllowedSources(array $sources): string
    {
        if ($sources === [] || in_array('all', $sources, true)) {
            return '不限制';
        }

        return strtoupper(implode(', ', $sources));
    }

    private function formatIntermediatePage(string $value): string
    {
        return match ($value) {
            'system_template' => '系统模版',
            'custom_template' => '自定义模版',
            default => '不使用',
        };
    }

    private function escapeLike(string $value): string
    {
        return str_replace(['\\', '%', '_'], ['\\\\', '\%', '\_'], $value);
    }

    private function makeCopyName(string $name, int $merchantId): string
    {
        $base = mb_strlen($name) > 47 ? mb_substr($name, 0, 47) : $name;
        $candidate = $base . '-副本';
        $suffix = 1;

        while (
            Cloak::query()
                ->where('merchant_id', $merchantId)
                ->where('name', $candidate)
                ->exists()
        ) {
            $candidate = $base . '-副本' . $suffix;
            $suffix++;
        }

        return mb_substr($candidate, 0, 50);
    }
}
