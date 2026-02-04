<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;

class LLMConfig extends Model
{
    use HasFactory;

    /**
     * The table associated with the model.
     */
    protected $table = 'llm_configs';

    /**
     * The attributes that are mass assignable.
     *
     * @var array<int, string>
     */
    protected $fillable = [
        'model_name',
        'api_url',
        'api_key',
        'is_active',
        'system_prompt',
        'user_prompt',
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
     * Get the active configuration.
     */
    public static function getActive()
    {
        return static::active()->first();
    }
}
