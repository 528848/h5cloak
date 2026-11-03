<?php

use App\Http\Controllers\Cloak\EntryController;
use Illuminate\Support\Facades\Route;

Route::any('/{path?}', EntryController::class)->where('path', '.*');
