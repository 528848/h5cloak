<?php

use App\Http\Controllers\Admin\CloakController as AdminCloakController;
use App\Http\Controllers\Admin\ApiLogController as AdminApiLogController;
use App\Http\Controllers\Admin\DashboardController as AdminDashboardController;
use App\Http\Controllers\Admin\DomainController as AdminDomainController;
use App\Http\Controllers\Admin\MerchantController;
use App\Http\Controllers\Auth\AuthController;
use App\Http\Controllers\Auth\MenuController;
use App\Http\Controllers\System\DictDataController;
use App\Http\Controllers\System\UserProfileController;
use App\Http\Controllers\Merchant\AccessListController;
use App\Http\Controllers\Merchant\ApiKeyController;
use App\Http\Controllers\Merchant\ApiLogController as MerchantApiLogController;
use App\Http\Controllers\Merchant\CloakController as MerchantCloakController;
use App\Http\Controllers\Merchant\DashboardController as MerchantDashboardController;
use App\Http\Controllers\Merchant\DomainController as MerchantDomainController;
use App\Http\Controllers\Open\CloakController as OpenCloakController;
use App\Http\Controllers\Open\DecisionController as OpenDecisionController;
use Illuminate\Support\Facades\Route;

Route::post('/login', [AuthController::class, 'login']);
Route::middleware('auth:sanctum')->group(function () {
    Route::post('/logout', [AuthController::class, 'logout']);
    Route::get('/getInfo', [AuthController::class, 'getInfo']);
    Route::get('/getRouters', [MenuController::class, 'routers']);

    Route::prefix('system')->group(function () {
        Route::get('/user/profile', [UserProfileController::class, 'show']);
        Route::put('/user/profile', [UserProfileController::class, 'update']);
        Route::post('/user/profile/avatar', [UserProfileController::class, 'updateAvatar']);
        Route::put('/user/profile/updatePwd', [UserProfileController::class, 'updatePassword']);
        Route::get('/dict/data/type/{type}', [DictDataController::class, 'type']);
    });
});

Route::prefix('admin')->group(function () {
    Route::get('/dashboard', [AdminDashboardController::class, 'index']);
    Route::get('/merchant/list', [MerchantController::class, 'index']);
    Route::post('/merchant', [MerchantController::class, 'store']);
    Route::put('/merchant/changeStatus', [MerchantController::class, 'changeStatus']);
    Route::get('/domain/list', [AdminDomainController::class, 'index']);
    Route::put('/domain/changeStatus', [AdminDomainController::class, 'changeStatus']);
    Route::get('/cloak/list', [AdminCloakController::class, 'index']);
    Route::put('/cloak/changeStatus', [AdminCloakController::class, 'changeStatus']);
    Route::get('/cloak-log/list', [AdminCloakController::class, 'logs']);
    Route::get('/api-log/list', [AdminApiLogController::class, 'index']);
});

Route::prefix('merchant')->middleware(['auth:sanctum', 'merchant.enabled'])->group(function () {
    Route::get('/dashboard', [MerchantDashboardController::class, 'index']);
    Route::get('/domain/list', [MerchantDomainController::class, 'index']);
    Route::get('/domain/dns-config', [MerchantDomainController::class, 'dnsConfig']);
    Route::post('/domain', [MerchantDomainController::class, 'store']);
    Route::post('/domain/{domainId}/verify', [MerchantDomainController::class, 'verify']);
    Route::get('/cloak/list', [MerchantCloakController::class, 'index']);
    Route::post('/cloak', [MerchantCloakController::class, 'store']);
    Route::put('/cloak/changeStatus', [MerchantCloakController::class, 'changeStatus']);
    Route::put('/cloak/{cloakId}', [MerchantCloakController::class, 'update']);
    Route::post('/cloak/{cloakId}/copy', [MerchantCloakController::class, 'copy']);
    Route::delete('/cloak/{cloakId}', [MerchantCloakController::class, 'destroy']);
    Route::get('/cloak-log/list', [MerchantCloakController::class, 'logs']);
    Route::get('/access-list/list', [AccessListController::class, 'index']);
    Route::post('/access-list', [AccessListController::class, 'store']);
    Route::put('/access-list/changeStatus', [AccessListController::class, 'changeStatus']);
    Route::put('/access-list/{listId}', [AccessListController::class, 'update']);
    Route::delete('/access-list/{listId}', [AccessListController::class, 'destroy']);
    Route::get('/api-key/list', [ApiKeyController::class, 'index']);
    Route::post('/api-key', [ApiKeyController::class, 'store']);
    Route::put('/api-key/changeStatus', [ApiKeyController::class, 'changeStatus']);
    Route::post('/api-key/{apiKeyId}/reset', [ApiKeyController::class, 'reset']);
    Route::delete('/api-key/{apiKeyId}', [ApiKeyController::class, 'destroy']);
    Route::get('/api-log/list', [MerchantApiLogController::class, 'index']);
});

Route::prefix('open')->group(function () {
    Route::post('/decision', [OpenDecisionController::class, 'decide']);
    Route::get('/cloaks', [OpenCloakController::class, 'index']);
    Route::post('/cloaks', [OpenCloakController::class, 'store']);
    Route::get('/cloaks/{cloakId}', [OpenCloakController::class, 'show']);
    Route::put('/cloaks/{cloakId}', [OpenCloakController::class, 'update']);
    Route::put('/cloaks/{cloakId}/status', [OpenCloakController::class, 'changeStatus']);
});
