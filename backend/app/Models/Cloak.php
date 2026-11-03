<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Cloak extends Model
{
    protected $fillable = [
        'merchant_id',
        'domain_id',
        'name',
        'status',
        'white_url',
        'black_url',
        'default_action',
        'rules',
        'remark',
    ];

    protected $casts = [
        'rules' => 'array',
    ];

    public function merchant(): BelongsTo
    {
        return $this->belongsTo(Merchant::class);
    }

    public function domain(): BelongsTo
    {
        return $this->belongsTo(Domain::class);
    }

    public function accessLists(): HasMany
    {
        return $this->hasMany(AccessList::class);
    }

    public function accessLogs(): HasMany
    {
        return $this->hasMany(CloakAccessLog::class);
    }
}
