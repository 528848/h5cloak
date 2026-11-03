<?php

namespace App\Support;

class CountryRegionLabels
{
    private static ?array $map = null;

    public static function label(string $code): string
    {
        $code = strtoupper(trim($code));

        return self::map()[$code] ?? $code;
    }

    public static function format(array $codes): string
    {
        if ($codes === []) {
            return '-';
        }

        return implode(', ', array_map(fn (string $code) => self::label($code), $codes));
    }

    private static function map(): array
    {
        if (self::$map !== null) {
            return self::$map;
        }

        $path = resource_path('data/country-regions-data.json');
        if (!is_file($path)) {
            return self::$map = [];
        }

        $regions = json_decode((string) file_get_contents($path), true) ?: [];
        $map = [];
        foreach ($regions as $region) {
            foreach ($region['children'] ?? [] as $country) {
                $map[strtoupper((string) $country['value'])] = (string) $country['label'];
            }
        }

        return self::$map = $map;
    }
}
