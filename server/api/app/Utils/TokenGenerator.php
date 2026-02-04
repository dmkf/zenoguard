<?php

namespace App\Utils;

class TokenGenerator
{
    /**
     * Generate a 64-character random token.
     */
    public static function generate(): string
    {
        return bin2hex(random_bytes(32));
    }

    /**
     * Generate a token with a specific prefix.
     */
    public static function generateWithPrefix(string $prefix): string
    {
        return $prefix . '_' . self::generate();
    }
}
