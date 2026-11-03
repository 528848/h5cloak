<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class AccessList extends Model
{
    protected $fillable = [
        'merchant_id',
        'cloak_id',
        'list_type',
        'target_type',
        'target_value',
        'scope',
        'status',
        'remark',
    ];

    public function merchant(): BelongsTo
    {
        return $this->belongsTo(Merchant::class);
    }

    public function cloak(): BelongsTo
    {
        return $this->belongsTo(Cloak::class);
    }
}
