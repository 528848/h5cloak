<?php

namespace App\Http\Controllers\System;

use App\Http\Controllers\Controller;
use App\Support\ApiResponse;

class DictDataController extends Controller
{
    private const DICTS = [
        'sys_user_sex' => [
            ['dictLabel' => '男', 'dictValue' => '0', 'listClass' => 'default', 'status' => '0'],
            ['dictLabel' => '女', 'dictValue' => '1', 'listClass' => 'default', 'status' => '0'],
            ['dictLabel' => '未知', 'dictValue' => '2', 'listClass' => 'default', 'status' => '0'],
        ],
        'sys_normal_disable' => [
            ['dictLabel' => '正常', 'dictValue' => '0', 'listClass' => 'primary', 'status' => '0'],
            ['dictLabel' => '停用', 'dictValue' => '1', 'listClass' => 'danger', 'status' => '0'],
        ],
    ];

    public function type(string $type)
    {
        $data = self::DICTS[$type] ?? [];

        return ApiResponse::success($data);
    }
}
