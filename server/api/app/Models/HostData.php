<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class HostData extends Model
{
    use HasFactory;

    /**
     * The attributes that are mass assignable.
     *
     * @var array<int, string>
     */
    protected $fillable = [
        'host_id',
        'report_time',
        'ssh_logins',
        'system_load',
        'network_traffic',
        'public_ip',
        'ip_location',
        'llm_summary',
        'is_alert',
    ];

    /**
     * The attributes that should be cast.
     *
     * @var array<string, string>
     */
    protected $casts = [
        'report_time' => 'datetime',
        'ssh_logins' => 'array',
        'system_load' => 'array',
        'network_traffic' => 'array',
        'is_alert' => 'boolean',
    ];

    /**
     * Get the host that owns the data.
     */
    public function host(): BelongsTo
    {
        return $this->belongsTo(Host::class);
    }

    /**
     * Scope to only include alert data.
     */
    public function scopeAlerts($query)
    {
        return $query->where('is_alert', true);
    }

    /**
     * Scope to filter by date range.
     */
    public function scopeInDateRange($query, $startDate, $endDate)
    {
        return $query->whereBetween('report_time', [$startDate, $endDate]);
    }
}
