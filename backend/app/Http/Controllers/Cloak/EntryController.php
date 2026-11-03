<?php

namespace App\Http\Controllers\Cloak;

use App\Http\Controllers\Controller;
use App\Models\CloakAccessLog;
use App\Services\Cloak\RuleEngine;
use Illuminate\Http\Request;
use Illuminate\Http\RedirectResponse;
use Illuminate\Http\Response;
use Illuminate\Support\Str;

class EntryController extends Controller
{
    public function __invoke(Request $request, RuleEngine $ruleEngine): Response|RedirectResponse
    {
        $decision = $ruleEngine->decide($request);
        $this->writeAccessLog($request, $decision);

        if ($decision['action'] === 'block') {
            return response('Access denied', 403);
        }

        if ($decision['action'] === 'not_found') {
            return response('Not Found', 404);
        }

        if ($decision['target']) {
            return redirect()->away($decision['target']);
        }

        return response('OK');
    }

    private function writeAccessLog(Request $request, array $decision): void
    {
        try {
            CloakAccessLog::query()->create([
                'request_id' => (string) Str::uuid(),
                'merchant_id' => $decision['domain']?->merchant_id,
                'domain_id' => $decision['domain']?->id,
                'cloak_id' => $decision['cloak']?->id,
                'ip_hash' => sha1((string) $request->ip()),
                'country' => $decision['country'] ?? null,
                'device_type' => str_contains(strtolower($request->userAgent() ?? ''), 'mobile') ? 'mobile' : 'pc',
                'user_agent_summary' => Str::limit($request->userAgent() ?? '', 180, ''),
                'referer_summary' => Str::limit($request->headers->get('referer', ''), 180, ''),
                'hit_rule' => $decision['hitRule'],
                'final_action' => $decision['finalAction'],
                'latency_ms' => $decision['latencyMs'],
                'created_at' => now(),
            ]);
        } catch (\Throwable) {
            // Access decisions must still return even if log storage is temporarily unavailable.
        }
    }
}
