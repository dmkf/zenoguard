<?php

namespace App\Services;

use App\Models\AlertConfig;
use App\Models\Host;
use App\Models\HostData;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;

class AlertService
{
    /**
     * Send an alert for a host.
     */
    public function sendAlert(Host $host, HostData $data): bool
    {
        $configs = AlertConfig::active()->get();

        if ($configs->isEmpty()) {
            Log::warning('No active alert configuration found');
            return false;
        }

        $success = true;

        foreach ($configs as $config) {
            if ($config->platform === 'dingtalk') {
                if (!$this->sendDingTalk($config, $host, $data)) {
                    $success = false;
                }
            }
            // Add more platforms here
        }

        return $success;
    }

    /**
     * Send a DingTalk notification.
     */
    protected function sendDingTalk(AlertConfig $config, Host $host, HostData $data): bool
    {
        try {
            $message = $this->buildDingTalkMessage($host, $data);

            $webhook = $config->webhook_url;

            $payload = [
                'msgtype' => 'text',
                'text' => [
                    'content' => $message
                ]
            ];

            // Add signature if secret is provided
            if (!empty($config->secret)) {
                $timestamp = time() * 1000;
                $sign = $this->generateDingTalkSign($config->secret, $timestamp);

                $webhook .= "&timestamp={$timestamp}&sign={$sign}";
            }

            $response = Http::timeout(10)->post($webhook, $payload);

            if ($response->successful()) {
                $result = $response->json();

                if (($result['errcode'] ?? 1) === 0) {
                    Log::info('DingTalk alert sent successfully');
                    return true;
                } else {
                    Log::error('DingTalk alert failed: ' . json_encode($result));
                    return false;
                }
            } else {
                Log::error('DingTalk HTTP error: ' . $response->body());
                return false;
            }
        } catch (\Exception $e) {
            Log::error('DingTalk alert exception: ' . $e->getMessage());
            return false;
        }
    }

    /**
     * Build the DingTalk message.
     */
    protected function buildDingTalkMessage(Host $host, HostData $data): string
    {
        $message = "【智巡Guard预警】\n";
        $message .= "主机: {$host->hostname}\n";
        $message .= "时间: {$data->report_time}\n";
        $message .= "公网IP: {$data->public_ip}\n";
        $message .= "位置: {$data->ip_location}\n";

        // Add SSH login alerts if any
        $sshLogins = $data->ssh_logins ?? [];
        if (!empty($sshLogins)) {
            $message .= "\nSSH登录记录:\n";
            foreach ($sshLogins as $login) {
                $status = $login['success'] ? '成功' : '失败';
                $message .= "  - {$login['user']}@{$login['ip']} ({$status})\n";
            }
        }

        // Add system load
        if ($data->system_load) {
            $load = $data->system_load;
            $message .= "\n系统负载: {$load['load1']}, {$load['load5']}, {$load['load15']}\n";
        }

        // Add LLM summary if available
        if (!empty($data->llm_summary)) {
            $message .= "\n分析: {$data->llm_summary}\n";
        }

        return $message;
    }

    /**
     * Generate DingTalk signature.
     */
    protected function generateDingTalkSign(string $secret, int $timestamp): string
    {
        $stringToSign = "{$timestamp}\n{$secret}";
        $sign = base64_encode(hash_hmac('sha256', $stringToSign, $secret, true));
        return rawurlencode($sign);
    }

    /**
     * Send an LLM analysis alert.
     */
    public function sendLLMAlert(Host $host, string $summary, array $latestData = null): bool
    {
        $configs = AlertConfig::active()->get();

        if ($configs->isEmpty()) {
            Log::warning('No active alert configuration found for LLM alert');
            return false;
        }

        $success = true;

        foreach ($configs as $config) {
            if ($config->platform === 'dingtalk') {
                if (!$this->sendDingTalkLLM($config, $host, $summary, $latestData)) {
                    $success = false;
                }
            }
            // Add more platforms here
        }

        return $success;
    }

    /**
     * Send a DingTalk notification for LLM analysis.
     */
    protected function sendDingTalkLLM(AlertConfig $config, Host $host, string $summary, array $latestData = null): bool
    {
        try {
            $message = $this->buildDingTalkLLMMessage($host, $summary, $latestData);

            $webhook = $config->webhook_url;

            $payload = [
                'msgtype' => 'text',
                'text' => [
                    'content' => $message
                ]
            ];

            // Add signature if secret is provided
            if (!empty($config->secret)) {
                $timestamp = time() * 1000;
                $sign = $this->generateDingTalkSign($config->secret, $timestamp);

                $webhook .= "&timestamp={$timestamp}&sign={$sign}";
            }

            $response = Http::timeout(10)->post($webhook, $payload);

            if ($response->successful()) {
                $result = $response->json();

                if (($result['errcode'] ?? 1) === 0) {
                    Log::info('DingTalk LLM alert sent successfully', [
                        'host' => $host->hostname,
                        'summary' => $summary
                    ]);
                    return true;
                } else {
                    Log::error('DingTalk LLM alert failed: ' . json_encode($result));
                    return false;
                }
            } else {
                Log::error('DingTalk LLM HTTP error: ' . $response->body());
                return false;
            }
        } catch (\Exception $e) {
            Log::error('DingTalk LLM alert exception: ' . $e->getMessage());
            return false;
        }
    }

    /**
     * Build the DingTalk message for LLM analysis.
     */
    protected function buildDingTalkLLMMessage(Host $host, string $summary, array $latestData = null): string
    {
        $message = "【智巡Guard LLM分析预警】\n";
        $message .= "主机: {$host->hostname}\n";
        $message .= "时间: " . now()->format('Y-m-d H:i:s') . "\n";

        if ($latestData) {
            $message .= "公网IP: {$latestData['public_ip']}\n";
            $message .= "位置: {$latestData['ip_location']}\n";
        }

        $message .= "\n分析结果: {$summary}";

        return $message;
    }

    /**
     * Test the DingTalk connection.
     */
    public function testDingTalk(): array
    {
        $config = AlertConfig::getActiveByPlatform('dingtalk');

        if (!$config) {
            return [
                'success' => false,
                'message' => 'No active DingTalk configuration',
            ];
        }

        try {
            $webhook = $config->webhook_url;

            if (!empty($config->secret)) {
                $timestamp = time() * 1000;
                $sign = $this->generateDingTalkSign($config->secret, $timestamp);
                $webhook .= "&timestamp={$timestamp}&sign={$sign}";
            }

            $response = Http::timeout(10)->post($webhook, [
                'msgtype' => 'text',
                'text' => [
                    'content' => '【智巡Guard测试】这是一条测试消息'
                ]
            ]);

            if ($response->successful()) {
                $result = $response->json();

                if (($result['errcode'] ?? 1) === 0) {
                    return [
                        'success' => true,
                        'message' => 'Test message sent successfully',
                    ];
                } else {
                    return [
                        'success' => false,
                        'message' => 'API error: ' . json_encode($result),
                    ];
                }
            } else {
                return [
                    'success' => false,
                    'message' => 'HTTP error: ' . $response->body(),
                ];
            }
        } catch (\Exception $e) {
            return [
                'success' => false,
                'message' => 'Test failed: ' . $e->getMessage(),
            ];
        }
    }
}
