<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Domain extends Model
{
    protected $fillable = [
        'merchant_id',
        'domain',
        'verify_type',
        'verify_value',
        'status',
        'last_checked_at',
        'remark',
    ];

    protected $casts = [
        'last_checked_at' => 'datetime',
    ];

    public function merchant(): BelongsTo
    {
        return $this->belongsTo(Merchant::class);
    }

    public function cloaks(): HasMany
    {
        return $this->hasMany(Cloak::class);
    }
}
