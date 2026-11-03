<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;

class DashboardMetric extends Model
{
    protected $fillable = [
        'merchant_id',
        'metric_date',
        'metric_hour',
        'visits',
        'white_hits',
        'black_hits',
        'blocks',
        'api_calls',
    ];

    protected $casts = [
        'metric_date' => 'date',
    ];
}
