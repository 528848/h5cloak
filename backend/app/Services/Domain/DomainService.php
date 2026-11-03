<?php

namespace App\Services\Domain;

use App\Models\Domain;
use App\Models\User;
use Illuminate\Support\Str;

class DomainService
{
    public function paginateForAdmin(array $filters): array
    {
        return $this->paginate($filters);
    }

    public function paginateForMerchant(User $user, array $filters): array
    {
        return $this->paginate($filters, $user->merchant_id);
    }

    public function createForMerchant(User $user, array $data): Domain
    {
        $verifyType = $data['verifyType'] ?? 'CNAME';

        return Domain::query()->create([
            'merchant_id' => $user->merchant_id,
            'domain' => $this->normalizeDomain($data['domain']),
            'verify_type' => $verifyType,
            'verify_value' => $this->makeVerifyValue($data['domain'], $verifyType),
            'status' => 'pending',
            'remark' => $data['remark'] ?? null,
        ]);
    }

    public function dnsConfig(): array
    {
        $nameservers = $this->resolveNameservers();
        $entryHost = $this->resolveEntryHost();

        return [
            'bindMode' => $nameservers !== [] ? 'nameserver' : 'record',
            'entryHost' => $entryHost,
            'entryIps' => $this->resolveEntryIps(),
            'entryNameservers' => $nameservers,
        ];
    }

    public function formatDnsPayload(Domain $domain): array
    {
        $config = $this->dnsConfig();

        return array_merge($config, [
            'domainId' => $domain->id,
            'verifyType' => $domain->verify_type,
            'verifyValue' => $domain->verify_value,
        ]);
    }

    public function verifyForMerchant(User $user, int $domainId): Domain
    {
        $domain = Domain::query()
            ->where('merchant_id', $user->merchant_id)
            ->findOrFail($domainId);

        // TODO: replace this optimistic check with real CNAME/TXT DNS lookup.
        $domain->forceFill([
            'status' => 'bound',
            'last_checked_at' => now(),
        ])->save();

        return $domain;
    }

    public function updateStatus(int $domainId, string $status, ?int $merchantId = null): void
    {
        Domain::query()
            ->when($merchantId, fn ($query) => $query->where('merchant_id', $merchantId))
            ->whereKey($domainId)
            ->update(['status' => $status]);
    }

    private function paginate(array $filters, ?int $merchantId = null): array
    {
        $pageSize = (int) ($filters['pageSize'] ?? 10);

        $result = Domain::query()
            ->with('merchant')
            ->when($merchantId, fn ($query) => $query->where('merchant_id', $merchantId))
            ->when($filters['domain'] ?? null, fn ($query, $domain) => $query->where('domain', 'like', "%{$domain}%"))
            ->when($filters['status'] ?? null, fn ($query, $status) => $query->where('status', $status))
            ->latest()
            ->paginate($pageSize);

        return [
            'rows' => $result->getCollection()->map(fn (Domain $domain) => [
                'domainId' => $domain->id,
                'merchantId' => $domain->merchant_id,
                'merchantName' => $domain->merchant?->name,
                'domain' => $domain->domain,
                'verifyType' => $domain->verify_type,
                'verifyValue' => $domain->verify_value,
                'status' => $domain->status,
                'cloakCount' => 0,
                'lastCheckedAt' => optional($domain->last_checked_at)->toDateTimeString(),
                'createdAt' => optional($domain->created_at)->toDateTimeString(),
            ])->all(),
            'total' => $result->total(),
        ];
    }

    private function normalizeDomain(string $domain): string
    {
        $domain = Str::lower(trim($domain));
        $domain = preg_replace('#^https?://#', '', $domain) ?: $domain;
        return trim($domain, "/ \t\n\r\0\x0B");
    }

    private function makeVerifyValue(string $domain, string $verifyType = 'CNAME'): string
    {
        if ($verifyType === 'CNAME') {
            return $this->resolveEntryHost();
        }

        return 'cloak-verify=' . substr(sha1($domain . (string) config('app.key')), 0, 16);
    }

    private function resolveEntryHost(): string
    {
        $host = trim((string) config('cloak.entry_host'));
        if ($host !== '') {
            return Str::lower($host);
        }

        $fromApp = parse_url((string) config('app.url'), PHP_URL_HOST);

        return $fromApp ? Str::lower((string) $fromApp) : 'cloak-entry.example.com';
    }

    private function resolveEntryIps(): array
    {
        return config('cloak.entry_ips', []);
    }

    private function resolveNameservers(): array
    {
        return config('cloak.entry_nameservers', []);
    }
}
