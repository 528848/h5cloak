<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

class OperationLog extends Model
{
    public const UPDATED_AT = null;

    protected $fillable = [
        'request_id',
        'actor_user_id',
        'actor_role',
        'merchant_id',
        'module',
        'action',
        'object_id',
        'before_snapshot',
        'after_snapshot',
        'ip',
        'created_at',
    ];

    protected $casts = [
        'before_snapshot' => 'array',
        'after_snapshot' => 'array',
    ];
}
