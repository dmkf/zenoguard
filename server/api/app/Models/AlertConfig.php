<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;

class AlertConfig extends Model
{
    use HasFactory;

    /**
     * The table associated with the model.
     */
    protected $table = 'alert_configs';

    /**
     * The attributes that are mass assignable.
     *
     * @var array<int, string>
     */
    protected $fillable = [
        'platform',
        'webhook_url',
        'secret',
        'is_active',
    ];

    /**
     * The attributes that should be cast.
     *
     * @var array<string, string>
     */
    protected $casts = [
        'is_active' => 'boolean',
    ];

    /**
     * Scope to get the active configuration.
     */
    public function scopeActive($query)
    {
        return $query->where('is_active', true);
    }

    /**
     * Scope to get by platform.
     */
    public function scopeByPlatform($query, string $platform)
    {
        return $query->where('platform', $platform);
    }

    /**
     * Get the active configuration for a platform.
     */
    public static function getActiveByPlatform(string $platform)
    {
        return static::active()->byPlatform($platform)->first();
    }
}
