<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class CloakAccessLog extends Model
{
    public const UPDATED_AT = null;

    protected $fillable = [
        'request_id',
        'merchant_id',
        'domain_id',
        'cloak_id',
        'ip_hash',
        'country',
        'region',
        'device_type',
        'user_agent_summary',
        'referer_summary',
        'hit_rule',
        'final_action',
        'latency_ms',
        'created_at',
    ];

    public function merchant(): BelongsTo
    {
        return $this->belongsTo(Merchant::class);
    }

    public function domain(): BelongsTo
    {
        return $this->belongsTo(Domain::class);
    }

    public function cloak(): BelongsTo
    {
        return $this->belongsTo(Cloak::class);
    }
}
