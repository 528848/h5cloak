<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\HasMany;

class Merchant extends Model
{
    protected $fillable = [
        'name',
        'contact_name',
        'contact_email',
        'contact_phone',
        'status',
        'plan_type',
        'expire_at',
        'domain_quota',
        'cloak_quota',
        'api_quota',
        'remark',
    ];

    protected $casts = [
        'expire_at' => 'datetime',
    ];

    public function users(): HasMany
    {
        return $this->hasMany(User::class);
    }

    public function domains(): HasMany
    {
        return $this->hasMany(Domain::class);
    }

    public function cloaks(): HasMany
    {
        return $this->hasMany(Cloak::class);
    }
}
