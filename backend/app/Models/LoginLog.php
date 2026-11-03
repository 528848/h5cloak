<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

class LoginLog extends Model
{
    public const UPDATED_AT = null;

    protected $fillable = [
        'username',
        'ip',
        'location',
        'browser',
        'os',
        'status',
        'message',
        'created_at',
    ];
}
