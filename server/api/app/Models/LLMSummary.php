<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;

class LLMSummary extends Model
{
    use HasFactory;

    protected $table = 'llm_summaries';

    protected $fillable = [
        'host_id',
        'summary',
        'is_alert',
        'analysis_time',
    ];

    protected $casts = [
        'is_alert' => 'boolean',
        'analysis_time' => 'datetime',
    ];

    /**
     * Get the host that owns the summary.
     */
    public function host()
    {
        return $this->belongsTo(Host::class);
    }
}
