<?php

namespace App\Services;

use App\Models\LLMConfig;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;

class LLMService
{
    protected ?LLMConfig $config;
    protected IPService $ipService;
    protected AlertService $alertService;

    public function __construct(IPService $ipService, AlertService $alertService)
    {
        $this->config = LLMConfig::getActive();
        $this->ipService = $ipService;
        $this->alertService = $alertService;
    }

    /**
     * Analyze host data and determine if an alert should be triggered.
     */
    public function analyze(array $data, string $rules): array
    {
        if (!$this->config) {
            Log::warning('No active LLM configuration found');
            return [
                'summary' => 'LLM not configured',
                'is_alert' => false,
            ];
        }

        try {
            $systemPrompt = $this->getSystemPrompt();
            $userPrompt = $this->buildPrompt($data, $rules);

            // Log the prompts being sent
            Log::info('LLM Analysis - System Prompt', [
                'prompt' => $systemPrompt
            ]);
            Log::info('LLM Analysis - User Prompt', [
                'prompt' => $userPrompt,
                'prompt_length' => strlen($userPrompt),
                'rules_section' => substr($userPrompt, strpos($userPrompt, '预警规则') ?: 0, 200)
            ]);

            $requestData = [
                'model' => $this->config->model_name,
                'messages' => [
                    [
                        'role' => 'system',
                        'content' => $systemPrompt
                    ],
                    [
                        'role' => 'user',
                        'content' => $userPrompt
                    ]
                ],
                'temperature' => 0.3,
                'max_tokens' => 500,
            ];

            $response = Http::timeout(30)
                ->withToken($this->config->api_key)
                ->post($this->config->api_url, $requestData);

            if ($response->successful()) {
                $result = $response->json();

                // Log the response and token usage
                $usage = $result['usage'] ?? [];
                Log::info('LLM Analysis - API Response', [
                    'model' => $result['model'] ?? 'unknown',
                    'prompt_tokens' => $usage['prompt_tokens'] ?? 'N/A',
                    'completion_tokens' => $usage['completion_tokens'] ?? 'N/A',
                    'total_tokens' => $usage['total_tokens'] ?? 'N/A',
                    'raw_response' => $result
                ]);

                // Parse the response
                $content = $result['choices'][0]['message']['content'] ?? '';

                // Log parsed content
                Log::info('LLM Analysis - Parsed Content', [
                    'content' => $content
                ]);

                // Try to parse JSON from content
                $parsed = json_decode($content, true);

                if (json_last_error() === JSON_ERROR_NONE && is_array($parsed)) {
                    $summary = $parsed['summary'] ?? 'Analysis completed';
                    $isAlert = $parsed['is_alert'] ?? false;

                    Log::info('LLM Analysis - Result', [
                        'summary' => $summary,
                        'is_alert' => $isAlert ? 'true' : 'false'
                    ]);

                    return [
                        'summary' => $summary,
                        'is_alert' => $isAlert,
                    ];
                }

                // Fallback: try to extract from content
                Log::warning('LLM Analysis - JSON parse failed, using content as fallback', [
                    'json_error' => json_last_error_msg(),
                    'content' => substr($content, 0, 200)
                ]);

                return [
                    'summary' => $content ?: 'Analysis completed',
                    'is_alert' => false,
                ];
            } else {
                Log::error('LLM API error', [
                    'status' => $response->status(),
                    'body' => $response->body()
                ]);
                return [
                    'summary' => 'LLM analysis failed',
                    'is_alert' => false,
                ];
            }
        } catch (\Exception $e) {
            Log::error('LLM service error', [
                'message' => $e->getMessage(),
                'trace' => $e->getTraceAsString()
            ]);
            return [
                'summary' => 'LLM error: ' . $e->getMessage(),
                'is_alert' => false,
            ];
        }
    }

    /**
     * Build the prompt for LLM analysis.
     */
    protected function buildPrompt(array $data, string $rules): string
    {
        $template = $this->getUserPrompt();

        // Replace placeholders
        $dataJson = json_encode($data, JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);
        $prompt = str_replace(
            ['{DATA}', '{RULES}'],
            [$dataJson, $rules],
            $template
        );

        return $prompt;
    }

    /**
     * Get the system prompt from configuration.
     */
    protected function getSystemPrompt(): string
    {
        $defaultPrompt = '你是一个服务器安全分析专家。请根据服务器数据判断是否存在安全问题，并给出简明扼要的分析结果。只返回JSON格式：{"summary": "一句话总结", "is_alert": true/false}';

        return $this->config->system_prompt ?? $defaultPrompt;
    }

    /**
     * Get the user prompt template from configuration.
     */
    protected function getUserPrompt(): string
    {
        $defaultPrompt = '当前服务器数据：
{DATA}

预警规则：{RULES}

请分析：
1. 用一句话总结服务器状态（不超过50字）
2. 判断是否触发预警（true/false）

只返回JSON格式：{"summary": "一句话总结", "is_alert": true/false}';

        return $this->config->user_prompt ?? $defaultPrompt;
    }

    /**
     * Test the LLM connection.
     */
    public function test(): array
    {
        if (!$this->config) {
            return [
                'success' => false,
                'message' => 'No active LLM configuration',
            ];
        }

        try {
            $response = Http::timeout(10)
                ->withToken($this->config->api_key)
                ->post($this->config->api_url, [
                    'model' => $this->config->model_name,
                    'messages' => [
                        [
                            'role' => 'user',
                            'content' => 'Hello, this is a test message.'
                        ]
                    ],
                    'max_tokens' => 10,
                ]);

            if ($response->successful()) {
                return [
                    'success' => true,
                    'message' => 'Connection successful',
                ];
            } else {
                return [
                    'success' => false,
                    'message' => 'API error: ' . $response->body(),
                ];
            }
        } catch (\Exception $e) {
            return [
                'success' => false,
                'message' => 'Connection failed: ' . $e->getMessage(),
            ];
        }
    }

    /**
     * Update the active configuration.
     */
    public function setConfig(LLMConfig $config): void
    {
        $this->config = $config;
    }

    /**
     * Get the alert service instance.
     */
    public function getAlertService(): AlertService
    {
        return $this->alertService;
    }
}
