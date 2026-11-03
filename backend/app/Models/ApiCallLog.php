<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class ApiCallLog extends Model
{
    public const UPDATED_AT = null;

    protected $fillable = [
        'request_id',
        'merchant_id',
        'api_key_id',
        'api_name',
        'method',
        'path',
        'request_ip',
        'status_code',
        'error_code',
        'latency_ms',
        'created_at',
    ];

    public function merchant(): BelongsTo
    {
        return $this->belongsTo(Merchant::class);
    }

    public function apiKey(): BelongsTo
    {
        return $this->belongsTo(ApiKey::class);
    }
}
