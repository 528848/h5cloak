<?php

namespace App\Services\Api;

use App\Models\AccessList;
use App\Models\ApiKey;
use App\Services\Cloak\CloakGeoResolver;
use Illuminate\Http\Request;

class ApiKeyDecisionService
{
    private string $lastHitRule = 'rule:cloak:review';

    public function __construct(
        private readonly CloakGeoResolver $geoResolver = new CloakGeoResolver()
    ) {
    }

    public function decide(Request $request, ApiKey $apiKey): array
    {
        $startedAt = microtime(true);
        $rules = ApiKeyRuleConfig::normalize(is_array($apiKey->rules) ? $apiKey->rules : []);
        $listHit = $this->matchMerchantList($request, $apiKey->merchant_id);

        if ($listHit?->list_type === 'black') {
            return $this->result('white', 'list:black:' . $listHit->id, $request, $startedAt);
        }

        if ($listHit?->list_type === 'white') {
            return $this->result('white', 'list:white:' . $listHit->id, $request, $startedAt);
        }

        $mode = $rules['mode'] ?? 'cloak';
        if ($mode === 'green' || $mode === 'review') {
            return $this->result('white', 'mode:' . $mode, $request, $startedAt);
        }

        if ($mode === 'full_open') {
            return $this->result('black', 'mode:full_open', $request, $startedAt);
        }

        if ($this->shouldServeReviewPage($request, $rules)) {
            return $this->result('white', $this->lastHitRule, $request, $startedAt);
        }

        return $this->result('black', 'rule:pass:black', $request, $startedAt);
    }

    public function buildVisitorRequest(Request $request, array $payload): Request
    {
        $server = $request->server->all();
        if (!empty($payload['ip'])) {
            $server['REMOTE_ADDR'] = $payload['ip'];
        }

        $visitorRequest = Request::create(
            $request->path(),
            'POST',
            [],
            $request->cookies->all(),
            [],
            $server,
            $request->getContent()
        );

        if (!empty($payload['userAgent'])) {
            $visitorRequest->headers->set('User-Agent', $payload['userAgent']);
        }
        if (!empty($payload['referer'])) {
            $visitorRequest->headers->set('Referer', $payload['referer']);
        }
        if (!empty($payload['acceptLanguage'])) {
            $visitorRequest->headers->set('Accept-Language', $payload['acceptLanguage']);
        }

        return $visitorRequest;
    }

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

        if (!empty($rules['spiderWhitelist']) && $this->isBot($request)) {
            $spiders = array_map('trim', explode(',', $rules['spiderWhitelist']));
            $ua = strtolower($request->userAgent() ?? '');
            $matched = collect($spiders)->contains(
                fn ($item) => $item !== '' && str_contains($ua, strtolower($item))
            );
            if ($matched) {
                $this->lastHitRule = 'rule:spider_whitelist';

                return true;
            }
        }

        $targetRegions = $rules['targetRegions'] ?? [];
        if ($targetRegions !== [] && !$this->geoResolver->matchTargetRegions($request, $targetRegions)) {
            $this->lastHitRule = 'rule:region_mismatch';

            return true;
        }

        return false;
    }

    private function matchMerchantList(Request $request, int $merchantId): ?AccessList
    {
        $ip = $request->ip();
        $ua = $request->userAgent() ?? '';
        $referer = $request->headers->get('referer', '');

        return AccessList::query()
            ->where('merchant_id', $merchantId)
            ->whereNull('cloak_id')
            ->where('status', 'enabled')
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

    private function isPc(Request $request): bool
    {
        $ua = strtolower($request->userAgent() ?? '');

        return !str_contains($ua, 'mobile')
            && !str_contains($ua, 'android')
            && !str_contains($ua, 'iphone')
            && !str_contains($ua, 'ipad');
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

    private function result(string $action, string $hitRule, Request $request, float $startedAt): array
    {
        return [
            'action' => $action,
            'page' => $action === 'white' ? 'review' : 'real',
            'hitRule' => $hitRule,
            'country' => $this->geoResolver->resolveCountryCode($request),
            'latencyMs' => (int) ((microtime(true) - $startedAt) * 1000),
        ];
    }
}
