<?php

namespace App\Support;

use Illuminate\Http\JsonResponse;

class ApiResponse
{
    public static function success(mixed $data = null, string $msg = '操作成功'): JsonResponse
    {
        return response()->json([
            'code' => 200,
            'msg' => $msg,
            'data' => $data,
        ]);
    }

    public static function page(array $rows, int $total, string $msg = '查询成功'): JsonResponse
    {
        return response()->json([
            'code' => 200,
            'msg' => $msg,
            'rows' => $rows,
            'total' => $total,
        ]);
    }

    public static function error(string $msg = '操作失败', int $code = 500): JsonResponse
    {
        return response()->json([
            'code' => $code,
            'msg' => $msg,
        ]);
    }
}
