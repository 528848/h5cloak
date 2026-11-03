<?php

namespace App\Services\Api;

use App\Services\Cloak\CloakRuleConfig;

class ApiKeyRuleConfig
{
    public static function defaults(): array
    {
        return [
            'mode' => 'cloak',
            'targetRegions' => [],
            'spiderWhitelist' => '',
            'blockPc' => false,
            'blockProxy' => false,
        ];
    }

    public static function normalize(array $rules): array
    {
        $rules = array_merge(self::defaults(), $rules);
        $rules['mode'] = in_array($rules['mode'], ['cloak', 'green', 'full_open', 'review'], true)
            ? $rules['mode']
            : 'cloak';
        $rules['targetRegions'] = CloakRuleConfig::normalizeTargetRegions($rules['targetRegions'] ?? []);
        $rules['spiderWhitelist'] = trim((string) ($rules['spiderWhitelist'] ?? ''));
        $rules['blockPc'] = (bool) ($rules['blockPc'] ?? false);
        $rules['blockProxy'] = (bool) ($rules['blockProxy'] ?? false);

        if ($rules['mode'] !== 'cloak') {
            $rules['targetRegions'] = [];
        }

        return $rules;
    }
}
