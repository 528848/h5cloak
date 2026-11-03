<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class ApiKey extends Model
{
    protected $fillable = [
        'merchant_id',
        'name',
        'key_prefix',
        'secret_hash',
        'permissions',
        'rules',
        'remark',
        'status',
        'last_called_at',
    ];

    protected $casts = [
        'permissions' => 'array',
        'rules' => 'array',
        'last_called_at' => 'datetime',
    ];

    public function merchant(): BelongsTo
    {
        return $this->belongsTo(Merchant::class);
    }
}
