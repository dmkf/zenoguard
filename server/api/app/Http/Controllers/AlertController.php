<?php

namespace App\Http\Controllers;

use App\Models\AlertConfig;
use App\Services\AlertService;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Validator;

class AlertController extends Controller
{
    protected AlertService $alertService;

    public function __construct(AlertService $alertService)
    {
        $this->alertService = $alertService;
    }

    /**
     * Get all alert configurations.
     */
    public function index()
    {
        $configs = AlertConfig::all();

        // Hide secrets in response
        $configs->each->makeHidden(['secret']);

        return response()->json(['data' => $configs]);
    }

    /**
     * Update or create alert configuration.
     */
    public function update(Request $request)
    {
        $validator = Validator::make($request->all(), [
            'platform' => 'required|string|max:50',
            'webhook_url' => 'required|string|max:512',
            'secret' => 'nullable|string|max:255',
            'is_active' => 'nullable|boolean',
        ]);

        if ($validator->fails()) {
            return response()->json(['error' => $validator->errors()], 400);
        }

        // Create or update config
        $config = AlertConfig::updateOrCreate(
            ['platform' => $request->input('platform')],
            [
                'webhook_url' => $request->input('webhook_url'),
                'secret' => $request->input('secret'),
                'is_active' => $request->input('is_active', true),
            ]
        );

        return response()->json([
            'message' => 'Alert configuration updated successfully',
            'data' => $config->makeHidden(['secret'])
        ]);
    }

    /**
     * Test alert configuration.
     */
    public function test(Request $request)
    {
        $platform = $request->input('platform', 'dingtalk');

        if ($platform === 'dingtalk') {
            $result = $this->alertService->testDingTalk();
        } else {
            return response()->json([
                'success' => false,
                'message' => "Platform '{$platform}' not supported"
            ], 400);
        }

        return response()->json($result);
    }

    /**
     * Delete alert configuration.
     */
    public function destroy(AlertConfig $config)
    {
        $config->delete();

        return response()->json(['message' => 'Alert configuration deleted']);
    }
}
