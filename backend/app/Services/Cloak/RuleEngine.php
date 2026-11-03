<?php

namespace App\Services\Cloak;

use App\Models\AccessList;
use App\Models\Cloak;
use App\Models\Domain;
use Illuminate\Http\Request;
use Illuminate\Support\Str;

class RuleEngine
{
    public function __construct(
        private readonly CloakGeoResolver $geoResolver = new CloakGeoResolver()
    ) {
    }

    public function decide(Request $request): array
    {
        $startedAt = microtime(true);
        $host = Str::lower($request->getHost());
        $domain = Domain::query()
            ->with('merchant')
            ->where('domain', $host)
            ->where('status', 'bound')
            ->first();

        if (!$domain || !$domain->merchant || $domain->merchant->status !== 'enabled') {
            return $this->decision($request, $host, null, null, 'block', null, 'domain:unavailable', $startedAt);
        }

        $cloak = Cloak::query()
            ->where('domain_id', $domain->id)
            ->where('merchant_id', $domain->merchant_id)
            ->where('status', 'enabled')
            ->latest()
            ->first();

        if (!$cloak) {
            return $this->decision($request, $host, $domain, null, 'block', null, 'cloak:missing', $startedAt);
        }

        $rules = array_merge(CloakRuleConfig::defaults(), $cloak->rules ?? []);
        $listHit = $this->matchList($request, $cloak);

        if ($listHit?->list_type === 'black') {
            return $this->decision(
                $request,
                $host,
                $domain,
                $cloak,
                'white',
                $cloak->white_url,
                'list:black:' . $listHit->id,
                $startedAt
            );
        }

        if ($listHit?->list_type === 'white') {
            return $this->decision($request, $host, $domain, $cloak, 'white', $cloak->white_url, 'list:white:' . $listHit->id, $startedAt);
        }

        $mode = $rules['mode'] ?? 'cloak';
        if ($mode === 'green' || $mode === 'review') {
            return $this->decision($request, $host, $domain, $cloak, 'white', $cloak->white_url, 'mode:' . $mode, $startedAt);
        }

        if ($mode === 'full_open') {
            return $this->decision($request, $host, $domain, $cloak, 'black', $cloak->black_url, 'mode:full_open', $startedAt);
        }

        if ($this->shouldServeReviewPage($request, $rules)) {
            return $this->decision($request, $host, $domain, $cloak, 'white', $cloak->white_url, $this->lastHitRule, $startedAt);
        }

        return $this->decision($request, $host, $domain, $cloak, 'black', $cloak->black_url, 'rule:pass:black', $startedAt);
    }

    private string $lastHitRule = 'rule:cloak:review';

    private function shouldServeReviewPage(Request $request, array $rules): bool
    {
        if (!empty($rules['blockPc']) && $this->isPc($request)) {
            $this->lastHitRule = 'rule:block_pc';

            return true;
        }

        if (!empty($rules['blockProxy']) && $this->isProxy($request)) {
            $this->lastHitRule = 'rule:block_proxy';

            return true;
        }

        if (!empty($rules['blockEmptyLanguage']) && trim((string) $request->header('Accept-Language', '')) === '') {
            $this->lastHitRule = 'rule:block_empty_language';

            return true;
        }

        $targetRegions = $rules['targetRegions'] ?? [];
        if ($targetRegions !== [] && !$this->geoResolver->matchTargetRegions($request, $targetRegions)) {
            $this->lastHitRule = 'rule:region_mismatch';

            return true;
        }

        $allowedSources = $rules['allowedSources'] ?? ['all'];
        if (!$this->matchAllowedSources($request->header('referer', ''), $allowedSources)) {
            $this->lastHitRule = 'rule:source_mismatch';

            return true;
        }

        if (!empty($rules['spiderWhitelist']) && $this->isBot($request)) {
            $spiders = array_map('trim', explode(',', $rules['spiderWhitelist']));
            $ua = strtolower($request->userAgent() ?? '');
            $matched = collect($spiders)->contains(fn ($item) => $item !== '' && str_contains($ua, strtolower($item)));
            if ($matched) {
                $this->lastHitRule = 'rule:spider_whitelist';

                return true;
            }
        }

        if (!empty($rules['deviceWhitelist'])) {
            $devices = array_map('trim', explode(',', $rules['deviceWhitelist']));
            $device = $this->detectDevice($request);
            if (!in_array($device, $devices, true)) {
                $this->lastHitRule = 'rule:device_whitelist';

                return true;
            }
        }

        return false;
    }

    private function matchList(Request $request, Cloak $cloak): ?AccessList
    {
        $ip = $request->ip();
        $ua = $request->userAgent() ?? '';
        $referer = $request->headers->get('referer', '');

        return AccessList::query()
            ->where('merchant_id', $cloak->merchant_id)
            ->where('status', 'enabled')
            ->where(function ($query) use ($cloak) {
                $query->whereNull('cloak_id')->orWhere('cloak_id', $cloak->id);
            })
            ->orderByRaw("case when list_type = 'black' then 0 when list_type = 'white' then 1 else 2 end")
            ->get()
            ->first(function (AccessList $item) use ($ip, $ua, $referer) {
                return match ($item->target_type) {
                    'ip' => $item->target_value === $ip,
                    'ua' => $item->target_value !== '' && str_contains($ua, $item->target_value),
                    'referer' => $item->target_value !== '' && str_contains($referer, $item->target_value),
                    default => false,
                };
            });
    }

    private function matchAllowedSources(?string $referer, array $allowedSources): bool
    {
        if (in_array('all', $allowedSources, true)) {
            return true;
        }

        $referer = strtolower($referer ?? '');
        foreach ($allowedSources as $source) {
            if ($source === 'facebook' && (str_contains($referer, 'facebook.com') || str_contains($referer, 'fb.com'))) {
                return true;
            }
            if ($source === 'google' && (str_contains($referer, 'google.') || str_contains($referer, 'gclid='))) {
                return true;
            }
            if ($source === 'tiktok' && (str_contains($referer, 'tiktok.com') || str_contains($referer, 'ttclid='))) {
                return true;
            }
            if ($source === 'kwai' && str_contains($referer, 'kwai.')) {
                return true;
            }
        }

        return false;
    }

    private function isPc(Request $request): bool
    {
        return $this->detectDevice($request) === 'pc';
    }

    private function detectDevice(Request $request): string
    {
        $ua = strtolower($request->userAgent() ?? '');
        if ($this->isBot($request)) {
            return 'bot';
        }
        if (str_contains($ua, 'ipad') || str_contains($ua, 'tablet')) {
            return 'tablet';
        }
        if (str_contains($ua, 'mobile') || str_contains($ua, 'android') || str_contains($ua, 'iphone')) {
            return 'mobile';
        }

        return 'pc';
    }

    private function isBot(Request $request): bool
    {
        $ua = strtolower($request->userAgent() ?? '');

        return str_contains($ua, 'bot')
            || str_contains($ua, 'spider')
            || str_contains($ua, 'crawl')
            || str_contains($ua, 'slurp');
    }

    private function isProxy(Request $request): bool
    {
        return $request->headers->has('Via')
            || $request->headers->has('X-Forwarded-For')
            || str_contains(strtolower($request->userAgent() ?? ''), 'proxy');
    }

    private function decision(
        Request $request,
        string $host,
        ?Domain $domain,
        ?Cloak $cloak,
        string $action,
        ?string $target,
        string $hitRule,
        float $startedAt
    ): array {
        return [
            'host' => $host,
            'domain' => $domain,
            'cloak' => $cloak,
            'action' => $action,
            'finalAction' => $action,
            'target' => $target,
            'hitRule' => $hitRule,
            'latencyMs' => (int) ((microtime(true) - $startedAt) * 1000),
            'country' => $this->geoResolver->resolveCountryCode($request),
        ];
    }
}
