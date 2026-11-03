<?php

namespace App\Http\Controllers\Merchant;

use App\Http\Controllers\Controller;
use App\Services\Domain\DomainService;
use App\Support\ApiResponse;
use Illuminate\Http\Request;

class DomainController extends Controller
{
    public function index(Request $request, DomainService $domainService)
    {
        $result = $domainService->paginateForMerchant($request->user(), $request->all());
        return ApiResponse::page($result['rows'], $result['total']);
    }

    public function dnsConfig(DomainService $domainService)
    {
        return ApiResponse::success($domainService->dnsConfig());
    }

    public function store(Request $request, DomainService $domainService)
    {
        $data = $request->validate([
            'domain' => ['required', 'string', 'max:255', 'unique:domains,domain'],
            'verifyType' => ['nullable', 'string', 'in:CNAME,TXT'],
            'remark' => ['nullable', 'string', 'max:255'],
        ]);

        $domain = $domainService->createForMerchant($request->user(), $data);
        return ApiResponse::success($domainService->formatDnsPayload($domain), '域名添加成功');
    }

    public function verify(Request $request, DomainService $domainService, string $domainId)
    {
        $domain = $domainService->verifyForMerchant($request->user(), (int) $domainId);
        return ApiResponse::success([
            'domainId' => $domain->id,
            'status' => $domain->status,
            'lastCheckedAt' => optional($domain->last_checked_at)->toDateTimeString(),
        ], '域名校验完成');
    }
}
