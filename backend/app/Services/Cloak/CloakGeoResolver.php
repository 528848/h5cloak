<?php

namespace App\Services\Cloak;

use Illuminate\Http\Request;

class CloakGeoResolver
{
    public function resolveCountryCode(Request $request): ?string
    {
        foreach (['CF-IPCountry', 'X-Country-Code', 'X-Country', 'CloudFront-Viewer-Country'] as $header) {
            $value = $request->header($header);
            if ($value && strtoupper(trim($value)) !== 'XX') {
                return strtoupper(trim($value));
            }
        }

        return null;
    }

    public function matchTargetRegions(Request $request, array $targetRegions): bool
    {
        if ($targetRegions === []) {
            return true;
        }

        $country = $this->resolveCountryCode($request);
        if (!$country) {
            return false;
        }

        $targets = array_map('strtoupper', $targetRegions);

        return in_array($country, $targets, true);
    }
}
