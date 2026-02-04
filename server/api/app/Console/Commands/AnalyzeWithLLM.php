<?php

namespace App\Console\Commands;

use App\Models\Host;
use App\Models\LLMSummary;
use App\Services\LLMService;
use Carbon\Carbon;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\Log;

class AnalyzeWithLLM extends Command
{
    /**
     * The name and signature of the console command.
     *
     * @var string
     */
    protected $signature = 'llm:analyze {--host= : Specific host ID to analyze}';

    /**
     * The console command description.
     *
     * @var string
     */
    protected $description = 'Perform LLM analysis for hosts based on their configured intervals';

    protected LLMService $llmService;

    public function __construct(LLMService $llmService)
    {
        parent::__construct();
        $this->llmService = $llmService;
    }

    /**
     * Execute the console command.
     */
    public function handle()
    {
        $hostId = $this->option('host');

        if ($hostId) {
            $this->analyzeHost($hostId);
        } else {
            $this->analyzeAllHosts();
        }

        return Command::SUCCESS;
    }

    /**
     * Analyze all active hosts that are due for analysis.
     */
    protected function analyzeAllHosts()
    {
        $this->info('Starting LLM analysis for all hosts...');

        $hosts = Host::active()->get();

        foreach ($hosts as $host) {
            if ($this->shouldAnalyze($host)) {
                $this->analyzeHost($host->id);
            } else {
                $this->line("Host {$host->hostname} (ID: {$host->id}) - Skipped (not due)");
            }
        }

        $this->info('LLM analysis completed.');
    }

    /**
     * Check if a host should be analyzed based on its interval.
     */
    protected function shouldAnalyze(Host $host): bool
    {
        // Get the last analysis time for this host
        $lastAnalysis = $host->latestLLMSummary;

        if (!$lastAnalysis) {
            // No previous analysis, should analyze
            return true;
        }

        // Calculate the next due time
        $nextDue = $lastAnalysis->analysis_time->addSeconds($host->llm_analysis_interval);

        return Carbon::now()->gte($nextDue);
    }

    /**
     * Analyze a specific host.
     */
    protected function analyzeHost(int $hostId)
    {
        $host = Host::find($hostId);

        if (!$host) {
            $this->error("Host ID {$hostId} not found.");
            return;
        }

        $this->line("Analyzing host: {$host->hostname} (ID: {$host->id})");

        try {
            // Get recent host data (last hour)
            $recentData = $host->hostData()
                ->where('report_time', '>=', Carbon::now()->subHour())
                ->orderBy('report_time', 'desc')
                ->get();

            if ($recentData->isEmpty()) {
                $this->warn("No recent data found for host {$host->hostname}");
                return;
            }

            // Prepare data for LLM analysis (aggregate all recent data)
            $this->info('Found ' . $recentData->count() . ' data records in the last hour');
            $data = $this->prepareAggregatedData($recentData);
            $this->info('Aggregated data: report_count=' . ($data['report_count'] ?? 0));
            $rules = $host->alert_rules ?? '分析服务器状态，判断是否存在安全问题';

            // Perform LLM analysis
            $this->line('Calling LLM API...');
            $result = $this->llmService->analyze($data, $rules);

            // Save the analysis result
            LLMSummary::create([
                'host_id' => $host->id,
                'summary' => $result['summary'],
                'is_alert' => $result['is_alert'],
                'analysis_time' => Carbon::now(),
            ]);

            $status = $result['is_alert'] ? 'ALERT' : 'OK';
            $this->info("Analysis complete: [{$status}] {$result['summary']}");

            // Send DingTalk notification if alert
            if ($result['is_alert']) {
                $this->line("Sending DingTalk notification...");
                $alertSent = $this->llmService->getAlertService()->sendLLMAlert($host, $result['summary'], $data);

                if ($alertSent) {
                    $this->info("DingTalk notification sent successfully");
                    Log::info("LLM Alert notification sent for host {$host->hostname}: {$result['summary']}");
                } else {
                    $this->warn("Failed to send DingTalk notification");
                    Log::warning("Failed to send LLM Alert notification for host {$host->hostname}");
                }
            }

        } catch (\Exception $e) {
            $this->error("Analysis failed for host {$host->hostname}: " . $e->getMessage());
            Log::error("LLM analysis error for host {$host->id}", [
                'error' => $e->getMessage(),
                'trace' => $e->getTraceAsString(),
            ]);
        }
    }

    /**
     * Prepare aggregated host data for LLM analysis.
     *
     * @param \Illuminate\Database\Eloquent\Collection $recentData
     * @return array
     */
    protected function prepareAggregatedData($recentData): array
    {
        // Use the latest record for basic info
        $latest = $recentData->first();

        // Aggregate SSH logins
        $aggregatedSSH = $this->aggregateSSHLogins($recentData);
        Log::info('Aggregated SSH data', [
            'total_unique_users' => $aggregatedSSH['total_unique_users'] ?? 0,
            'summary_count' => count($aggregatedSSH['summary'] ?? []),
            'failed_attempts_count' => count($aggregatedSSH['failed_attempts'] ?? []),
            'sample_summary' => $aggregatedSSH['summary'][0] ?? null,
        ]);

        // Calculate average system load
        $avgLoad = $this->calculateAverageLoad($recentData);

        // Aggregate network traffic
        $totalTraffic = $this->aggregateTraffic($recentData);

        return [
            'hostname' => optional($latest->host)->hostname,
            'report_time' => $latest->report_time,
            'report_count' => $recentData->count(), // Number of reports aggregated
            'time_range' => [
                'from' => $recentData->last()->report_time,
                'to' => $latest->report_time,
            ],
            'system_load' => $avgLoad,
            'network_traffic' => $totalTraffic,
            'ssh_logins' => $aggregatedSSH,
            'public_ip' => $latest->public_ip,
            'ip_location' => $latest->ip_location,
        ];
    }

    /**
     * Aggregate SSH login data from multiple reports.
     *
     * @param \Illuminate\Database\Eloquent\Collection $recentData
     * @return array
     */
    protected function aggregateSSHLogins($recentData): array
    {
        $loginStats = [];
        $failedLogins = [];

        foreach ($recentData as $data) {
            $sshLogins = $data->ssh_logins ?? [];

            foreach ($sshLogins as $login) {
                $key = $login['user'] . '@' . $login['ip'];

                if (!isset($loginStats[$key])) {
                    $loginStats[$key] = [
                        'user' => $login['user'],
                        'ip' => $login['ip'],
                        'success_count' => 0,
                        'failed_count' => 0,
                        'last_seen' => $login['time'],
                        'method' => $login['method'] ?? 'unknown',
                    ];
                }

                if ($login['success']) {
                    $loginStats[$key]['success_count']++;
                } else {
                    $loginStats[$key]['failed_count']++;
                    // Track recent failed logins (last 5)
                    if (count($failedLogins) < 10) {
                        $failedLogins[] = [
                            'user' => $login['user'],
                            'ip' => $login['ip'],
                            'time' => $login['time'],
                            'method' => $login['method'] ?? 'unknown',
                        ];
                    }
                }

                // Update last seen time
                if ($login['time'] > $loginStats[$key]['last_seen']) {
                    $loginStats[$key]['last_seen'] = $login['time'];
                }
            }
        }

        // Build summary for LLM
        $summary = [
            'total_unique_users' => count($loginStats),
            'summary' => [],
            'failed_attempts' => $failedLogins,
        ];

        // Only include users with suspicious activity
        foreach ($loginStats as $stat) {
            $totalAttempts = $stat['success_count'] + $stat['failed_count'];

            // Include if: has failures, high activity (>5 attempts), or successful login
            if ($stat['failed_count'] > 0 || $totalAttempts > 5 || $stat['success_count'] > 0) {
                $summary['summary'][] = [
                    'user' => $stat['user'],
                    'ip' => $stat['ip'],
                    'success_count' => $stat['success_count'],
                    'failed_count' => $stat['failed_count'],
                    'total_attempts' => $totalAttempts,
                    'last_seen' => $stat['last_seen'],
                    'method' => $stat['method'],
                ];
            }
        }

        return $summary;
    }

    /**
     * Calculate average system load from multiple reports.
     *
     * @param \Illuminate\Database\Eloquent\Collection $recentData
     * @return array|null
     */
    protected function calculateAverageLoad($recentData): ?array
    {
        $loads = [];

        foreach ($recentData as $data) {
            if ($data->system_load) {
                $loads[] = $data->system_load;
            }
        }

        if (empty($loads)) {
            return null;
        }

        $count = count($loads);
        $avgLoad1 = array_sum(array_column($loads, 'load1')) / $count;
        $avgLoad5 = array_sum(array_column($loads, 'load5')) / $count;
        $avgLoad15 = array_sum(array_column($loads, 'load15')) / $count;

        return [
            'load1' => round($avgLoad1, 2),
            'load5' => round($avgLoad5, 2),
            'load15' => round($avgLoad15, 2),
            'sample_count' => $count,
        ];
    }

    /**
     * Aggregate network traffic from multiple reports.
     *
     * @param \Illuminate\Database\Eloquent\Collection $recentData
     * @return array|null
     */
    protected function aggregateTraffic($recentData): ?array
    {
        $totalIn = 0;
        $totalOut = 0;
        $count = 0;

        foreach ($recentData as $data) {
            if ($data->network_traffic) {
                $totalIn += $data->network_traffic['in_bytes'] ?? 0;
                $totalOut += $data->network_traffic['out_bytes'] ?? 0;
                $count++;
            }
        }

        if ($count === 0) {
            return null;
        }

        return [
            'total_in_bytes' => $totalIn,
            'total_out_bytes' => $totalOut,
            'avg_in_bytes' => round($totalIn / $count),
            'avg_out_bytes' => round($totalOut / $count),
            'sample_count' => $count,
        ];
    }
}
