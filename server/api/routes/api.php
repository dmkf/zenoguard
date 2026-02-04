<?php

use App\Http\Controllers\AgentController;
use App\Http\Controllers\AuthController;
use App\Http\Controllers\HostController;
use App\Http\Controllers\LLMController;
use App\Http\Controllers\AlertController;
use Illuminate\Support\Facades\Route;

/*
|--------------------------------------------------------------------------
| API Routes
|--------------------------------------------------------------------------
|
| Here is where you can register API routes for your application. These
| routes are loaded by the RouteServiceProvider and all of them will
| be assigned to the "api" middleware group. Make something great!
|
*/

// Agent routes (Token authentication)
Route::middleware('token.auth')->group(function () {
    Route::post('/agent/report', [AgentController::class, 'report']);

    // Allow agent to query its own data
    Route::get('/agent/me', [AgentController::class, 'me']);
    Route::get('/agent/data', [AgentController::class, 'myData']);
});

// Admin authentication routes
Route::post('/auth/login', [AuthController::class, 'login']);

// Protected auth routes (require authentication)
Route::middleware('auth:sanctum')->group(function () {
    Route::get('/auth/me', [AuthController::class, 'me']);
    Route::post('/auth/logout', [AuthController::class, 'logout']);
    Route::post('/auth/change-password', [AuthController::class, 'changePassword']);
});

// Protected admin routes (Session/Sanctum authentication)
Route::middleware('auth:sanctum')->group(function () {
    // Dashboard
    Route::get('dashboard/stats', [HostController::class, 'dashboardStats']);
    Route::get('llm-analyses/recent', [HostController::class, 'recentLLAnalyses']);

    // Host management - specific routes must be before apiResource
    Route::get('hosts/{host}/data', [HostController::class, 'dataList']);
    Route::delete('hosts/{host}/data/clean', [HostController::class, 'cleanOldData']);
    Route::get('hosts/{host}/trend', [HostController::class, 'trendData']);
    Route::get('hosts/{host}/llm-summaries', [HostController::class, 'llmSummaries']);
    Route::post('hosts/{host}/trigger-analysis', [HostController::class, 'triggerAnalysis']);
    Route::post('hosts/{host}/regenerate-token', [HostController::class, 'regenerateToken']);
    Route::apiResource('hosts', HostController::class);

    // LLM configuration
    Route::get('llm', [LLMController::class, 'index']);
    Route::put('llm', [LLMController::class, 'update']);
    Route::post('llm/test', [LLMController::class, 'test']);

    // Alert configuration
    Route::get('alert', [AlertController::class, 'index']);
    Route::put('alert', [AlertController::class, 'update']);
    Route::post('alert/test', [AlertController::class, 'test']);
    Route::delete('alert/{id}', [AlertController::class, 'destroy']);
});
