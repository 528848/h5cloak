<?php

namespace App\Services\Cloak;

use Illuminate\Support\Str;

class CloakRuleConfig
{
    public static function defaults(): array
    {
        return [
            'reviewPageMode' => 'external',
            'realUrls' => [],
            'shortLinkType' => 'system',
            'shortLinkUrl' => null,
            'allowedSources' => ['all'],
            'mode' => 'cloak',
            'targetRegions' => [],
            'spiderWhitelist' => '',
            'deviceWhitelist' => '',
            'blockEmptyLanguage' => false,
            'blockPc' => false,
            'blockProxy' => false,
            'intermediatePage' => 'none',
            'reviewQuickTag' => '',
        ];
    }

    public static function normalize(array $input, ?string $domainHost = null): array
    {
        $rules = array_merge(self::defaults(), $input['rules'] ?? []);
        $realUrls = array_values(array_filter($rules['realUrls'] ?? []));
        if (empty($realUrls) && !empty($input['blackUrl'])) {
            $realUrls = [$input['blackUrl']];
        }

        $rules['realUrls'] = $realUrls;
        $rules['reviewPageMode'] = in_array($rules['reviewPageMode'], ['external', 'local_template'], true)
            ? $rules['reviewPageMode']
            : 'external';
        $rules['shortLinkType'] = in_array($rules['shortLinkType'], ['system', 'custom'], true)
            ? $rules['shortLinkType']
            : 'system';
        $rules['mode'] = in_array($rules['mode'], ['cloak', 'green', 'full_open', 'review'], true)
            ? $rules['mode']
            : 'cloak';
        $rules['intermediatePage'] = in_array($rules['intermediatePage'], ['none', 'system_template', 'custom_template'], true)
            ? $rules['intermediatePage']
            : 'none';
        $rules['allowedSources'] = self::normalizeSources($rules['allowedSources'] ?? ['all']);
        $rules['targetRegions'] = self::normalizeTargetRegions($rules['targetRegions'] ?? []);
        $rules['blockEmptyLanguage'] = (bool) ($rules['blockEmptyLanguage'] ?? false);
        $rules['blockPc'] = (bool) ($rules['blockPc'] ?? false);
        $rules['blockProxy'] = (bool) ($rules['blockProxy'] ?? false);

        if ($rules['shortLinkType'] === 'system' && $domainHost) {
            $existingShortLink = trim((string) ($rules['shortLinkUrl'] ?? ''));
            if ($existingShortLink === '') {
                $rules['shortLinkUrl'] = 'https://' . $domainHost . '/c/' . Str::lower(Str::random(6));
            }
        }

        return $rules;
    }

    public static function normalizeSources(mixed $sources): array
    {
        if (!is_array($sources) || $sources === []) {
            return ['all'];
        }

        $allowed = ['facebook', 'google', 'tiktok', 'kwai', 'all'];
        $normalized = array_values(array_intersect($sources, $allowed));

        return $normalized === [] ? ['all'] : $normalized;
    }

    public static function normalizeTargetRegions(mixed $regions): array
    {
        if (!is_array($regions)) {
            return [];
        }

        return array_values(array_unique(array_filter(array_map(
            fn ($code) => strtoupper(trim((string) $code)),
            $regions
        ))));
    }
}
