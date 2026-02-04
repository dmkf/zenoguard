<?php

namespace App\Http\Controllers;

use App\Models\LLMConfig;
use App\Services\LLMService;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Validator;

class LLMController extends Controller
{
    protected LLMService $llmService;

    public function __construct(LLMService $llmService)
    {
        $this->llmService = $llmService;
    }

    /**
     * Get current LLM configuration.
     */
    public function index()
    {
        $config = LLMConfig::getActive();

        if (!$config) {
            return response()->json(['data' => null]);
        }

        // Hide API key in response
        $config->makeHidden(['api_key']);

        return response()->json(['data' => $config]);
    }

    /**
     * Update LLM configuration.
     */
    public function update(Request $request)
    {
        $validator = Validator::make($request->all(), [
            'model_name' => 'required|string|max:255',
            'api_url' => 'required|string|max:512',
            'api_key' => 'required|string|max:255',
            'is_active' => 'nullable|boolean',
            'system_prompt' => 'nullable|string|max:5000',
            'user_prompt' => 'nullable|string|max:10000',
        ]);

        if ($validator->fails()) {
            return response()->json(['error' => $validator->errors()], 400);
        }

        // Deactivate all existing configs
        LLMConfig::query()->update(['is_active' => false]);

        // Create or update config
        $config = LLMConfig::updateOrCreate(
            ['api_url' => $request->input('api_url')],
            [
                'model_name' => $request->input('model_name'),
                'api_key' => $request->input('api_key'),
                'is_active' => $request->input('is_active', true),
                'system_prompt' => $request->input('system_prompt'),
                'user_prompt' => $request->input('user_prompt'),
            ]
        );

        // Update service config
        $this->llmService->setConfig($config);

        return response()->json([
            'message' => 'LLM configuration updated successfully',
            'data' => $config->makeHidden(['api_key'])
        ]);
    }

    /**
     * Test LLM connection.
     */
    public function test()
    {
        $result = $this->llmService->test();

        return response()->json($result);
    }
}
