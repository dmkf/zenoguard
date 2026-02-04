<?php

namespace App\Services;

use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Log;

class IPService
{
    // 使用多个API源作为备用（按优先级排序）
    protected array $apis = [
        [
            'url' => 'https://ipwhois.app',
            'lang' => 'zh-CN',  // 中文
            'fields' => null
        ],
        [
            'url' => 'https://ipapi.co',
            'lang' => 'zh',  // 中文
            'fields' => null
        ],
        [
            'url' => 'http://ip-api.com/json',
            'lang' => 'zh-CN',  // 中文
            'fields' => 'country,regionName,city'
        ],
    ];
    protected int $cacheTimeout = 86400; // 24 hours

    /**
     * Get the location for an IP address.
     */
    public function getLocation(string $ip): string
    {
        // Check cache first
        $cacheKey = "ip_location_{$ip}";
        if (Cache::has($cacheKey)) {
            return Cache::get($cacheKey);
        }

        // Try each API until one succeeds
        foreach ($this->apis as $apiConfig) {
            $location = $this->getLocationFromAPI($ip, $apiConfig);
            if ($location && $location !== 'Unknown') {
                Cache::put($cacheKey, $location, $this->cacheTimeout);
                return $location;
            }
        }

        return 'Unknown';
    }

    /**
     * Get location from specific API
     */
    protected function getLocationFromAPI(string $ip, array $apiConfig): string
    {
        try {
            $url = $apiConfig['url'] . '/' . $ip;
            $query = [];

            if (isset($apiConfig['lang'])) {
                $query['lang'] = $apiConfig['lang'];
            }

            if (isset($apiConfig['fields'])) {
                $query['fields'] = $apiConfig['fields'];
            }

            if (!empty($query)) {
                $url .= '?' . http_build_query($query);
            }

            $response = Http::timeout(10)->get($url);

            if ($response->successful()) {
                $data = $response->json();

                // Build location string from response (handle different API formats)
                $country = $data['country'] ?? null;
                $region = $data['regionName'] ?? $data['region'] ?? null;
                $city = $data['city'] ?? null;

                $parts = array_filter([$country, $region, $city]);
                $location = implode('', $parts);

                return $location ?: 'Unknown';
            }
        } catch (\Exception $e) {
            Log::error('Failed to get IP location from ' . $apiConfig['url'] . ': ' . $e->getMessage());
        }

        return 'Unknown';
    }

    /**
     * Get detailed information for an IP address.
     */
    public function getDetails(string $ip): ?array
    {
        // Check cache first
        $cacheKey = "ip_details_{$ip}";
        if (Cache::has($cacheKey)) {
            return Cache::get($cacheKey);
        }

        // Try each API until one succeeds
        foreach ($this->apis as $apiConfig) {
            $details = $this->getDetailsFromAPI($ip, $apiConfig);
            if ($details) {
                Cache::put($cacheKey, $details, $this->cacheTimeout);
                return $details;
            }
        }

        return null;
    }

    /**
     * Get details from specific API
     */
    protected function getDetailsFromAPI(string $ip, array $apiConfig): ?array
    {
        try {
            $url = $apiConfig['url'] . '/' . $ip;
            $query = [];

            if (isset($apiConfig['lang'])) {
                $query['lang'] = $apiConfig['lang'];
            }

            if (!empty($query)) {
                $url .= '?' . http_build_query($query);
            }

            $response = Http::timeout(5)->get($url);

            if ($response->successful()) {
                return $response->json();
            }
        } catch (\Exception $e) {
            Log::error('Failed to get IP details from ' . $apiConfig['url'] . ': ' . $e->getMessage());
        }

        return null;
    }

    /**
     * Check if an IP is private/internal.
     */
    public function isPrivateIP(string $ip): bool
    {
        // Check for private IP ranges
        $privateRanges = [
            '10.0.0.0/8',
            '172.16.0.0/12',
            '192.168.0.0/16',
            '127.0.0.0/8',
        ];

        $ipLong = ip2long($ip);

        foreach ($privateRanges as $range) {
            [$rangeIP, $mask] = explode('/', $range);
            $rangeLong = ip2long($rangeIP);
            $maskLong = -1 << (32 - $mask);

            if (($ipLong & $maskLong) == ($rangeLong & $maskLong)) {
                return true;
            }
        }

        return false;
    }
}
