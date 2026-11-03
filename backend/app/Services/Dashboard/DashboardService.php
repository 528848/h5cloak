<?php

namespace App\Services\Dashboard;

use App\Models\ApiCallLog;
use App\Models\Cloak;
use App\Models\CloakAccessLog;
use App\Models\Domain;
use App\Models\Merchant;

class DashboardService
{
    public function admin(): array
    {
        $today = now()->startOfDay();

        return [
            'todayVisits' => CloakAccessLog::query()->where('created_at', '>=', $today)->count(),
            'todayWhiteHits' => CloakAccessLog::query()->where('created_at', '>=', $today)->where('final_action', 'white')->count(),
            'todayBlackHits' => CloakAccessLog::query()->where('created_at', '>=', $today)->where('final_action', 'black')->count(),
            'todayBlocks' => CloakAccessLog::query()->where('created_at', '>=', $today)->where('final_action', 'block')->count(),
            'todayApiCalls' => ApiCallLog::query()->where('created_at', '>=', $today)->count(),
            'merchantCount' => Merchant::query()->count(),
            'enabledMerchantCount' => Merchant::query()->where('status', 'enabled')->count(),
            'domainCount' => Domain::query()->count(),
            'abnormalDomainCount' => Domain::query()->where('status', 'abnormal')->count(),
            'cloakCount' => Cloak::query()->count(),
            'enabledCloakCount' => Cloak::query()->where('status', 'enabled')->count(),
        ];
    }

    public function merchant(int $merchantId): array
    {
        $today = now()->startOfDay();

        return [
            'todayVisits' => CloakAccessLog::query()->where('merchant_id', $merchantId)->where('created_at', '>=', $today)->count(),
            'todayWhiteHits' => CloakAccessLog::query()->where('merchant_id', $merchantId)->where('created_at', '>=', $today)->where('final_action', 'white')->count(),
            'todayBlackHits' => CloakAccessLog::query()->where('merchant_id', $merchantId)->where('created_at', '>=', $today)->where('final_action', 'black')->count(),
            'todayBlocks' => CloakAccessLog::query()->where('merchant_id', $merchantId)->where('created_at', '>=', $today)->where('final_action', 'block')->count(),
            'todayApiCalls' => ApiCallLog::query()->where('merchant_id', $merchantId)->where('created_at', '>=', $today)->count(),
            'domainCount' => Domain::query()->where('merchant_id', $merchantId)->count(),
            'enabledDomainCount' => Domain::query()->where('merchant_id', $merchantId)->where('status', 'bound')->count(),
            'cloakCount' => Cloak::query()->where('merchant_id', $merchantId)->count(),
            'enabledCloakCount' => Cloak::query()->where('merchant_id', $merchantId)->where('status', 'enabled')->count(),
        ];
    }
}
