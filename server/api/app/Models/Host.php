<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Host extends Model
{
    use HasFactory;

    /**
     * The attributes that are mass assignable.
     *
     * @var array<int, string>
     */
    protected $fillable = [
        'hostname',
        'token',
        'remark',
        'report_interval',
        'alert_rules',
        'is_active',
        'llm_analysis_interval',
    ];

    /**
     * The attributes that should be cast.
     *
     * @var array<string, string>
     */
    protected $casts = [
        'report_interval' => 'integer',
        'llm_analysis_interval' => 'integer',
        'is_active' => 'boolean',
    ];

    /**
     * Get the host data for the host.
     */
    public function hostData(): HasMany
    {
        return $this->hasMany(HostData::class);
    }

    /**
     * Get the LLM summaries for the host.
     */
    public function llmSummaries()
    {
        return $this->hasMany(LLMSummary::class)->orderBy('analysis_time', 'desc');
    }

    /**
     * Get the latest LLM summary for this host.
     */
    public function latestLLMSummary()
    {
        return $this->hasOne(LLMSummary::class)->latestOfMany()->orderBy('analysis_time', 'desc');
    }

    /**
     * Scope to only include active hosts.
     */
    public function scopeActive($query)
    {
        return $query->where('is_active', true);
    }

    /**
     * Get the latest data for this host.
     */
    public function latestData()
    {
        return $this->hasOne(HostData::class)->latestOfMany();
    }
}
