<?php

namespace App\Http\Controllers;

use App\Models\Host;
use App\Models\HostData;
use App\Models\LLMSummary;
use App\Services\LLMService;
use App\Utils\TokenGenerator;
use Carbon\Carbon;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Hash;
use Illuminate\Support\Facades\Validator;
use Illuminate\Support\Facades\Log;
use Illuminate\Validation\Rule;

class HostController extends Controller
{
    protected LLMService $llmService;

    public function __construct(LLMService $llmService)
    {
        $this->llmService = $llmService;
    }
    /**
     * List all hosts.
     */
    public function index(Request $request)
    {
        $query = Host::query();

        // Filter by active status
        if ($request->has('is_active')) {
            $query->where('is_active', $request->boolean('is_active'));
        }

        // Search by hostname
        if ($request->has('search')) {
            $search = $request->input('search');
            $query->where('hostname', 'like', "%{$search}%")
                ->orWhere('remark', 'like', "%{$search}%");
        }

        $hosts = $query->with('latestData')->orderBy('created_at', 'desc')->paginate(20);

        return response()->json($hosts);
    }

    /**
     * Get a specific host.
     */
    public function show(Host $host)
    {
        $host->load('latestData', 'latestLLMSummary');

        return response()->json($host);
    }

    /**
     * Create a new host.
     */
    public function store(Request $request)
    {
        $validator = Validator::make($request->all(), [
            'hostname' => 'required|string|max:255',
            'remark' => 'nullable|string',
            'report_interval' => 'nullable|integer|min:300|max:86400',
            'alert_rules' => 'nullable|string|max:2000',
            'is_active' => 'nullable|boolean',
        ]);

        if ($validator->fails()) {
            return response()->json(['error' => $validator->errors()], 400);
        }

        $host = Host::create([
            'hostname' => $request->input('hostname'),
            'token' => TokenGenerator::generate(),
            'remark' => $request->input('remark'),
            'report_interval' => $request->input('report_interval', 3600),
            'alert_rules' => $request->input('alert_rules'),
            'is_active' => $request->input('is_active', true),
        ]);

        return response()->json($host, 201);
    }

    /**
     * Update a host.
     */
    public function update(Request $request, Host $host)
    {
        $validator = Validator::make($request->all(), [
            'hostname' => 'sometimes|string|max:255',
            'remark' => 'nullable|string',
            'report_interval' => 'nullable|integer|min:300|max:86400',
            'alert_rules' => 'nullable|string|max:2000',
            'is_active' => 'nullable|boolean',
        ]);

        if ($validator->fails()) {
            return response()->json(['error' => $validator->errors()], 400);
        }

        $host->update($request->only([
            'hostname',
            'remark',
            'report_interval',
            'alert_rules',
            'is_active',
        ]));

        return response()->json($host);
    }

    /**
     * Delete a host.
     */
    public function destroy(Request $request, Host $host)
    {
        // Validate password confirmation
        $request->validate([
            'password' => 'required|string',
        ]);

        // Verify admin password
        if (!Hash::check($request->password, $request->user()->password)) {
            return response()->json([
                'error' => 'Incorrect password'
            ], 403);
        }

        $host->delete();

        return response()->json(['message' => 'Host deleted successfully']);
    }

    /**
     * Get host data list.
     */
    public function dataList(Request $request, Host $host)
    {
        $query = $host->hostData();

        // Filter by date range
        if ($request->has('start_date') && $request->has('end_date')) {
            $query->whereBetween('report_time', [
                $request->input('start_date'),
                $request->input('end_date')
            ]);
        }

        // Filter by alert status
        if ($request->has('is_alert')) {
            $query->where('is_alert', $request->boolean('is_alert'));
        }

        // Search in summary
        if ($request->has('search')) {
            $search = $request->input('search');
            $query->where('llm_summary', 'like', "%{$search}%");
        }

        $data = $query->orderBy('report_time', 'desc')->paginate(50);

        return response()->json($data);
    }

    /**
     * Clean old host data.
     */
    public function cleanOldData(Request $request, Host $host)
    {
        $period = $request->input('period', '1m');

        // Calculate cutoff date
        $cutoffDate = match($period) {
            '3d' => now()->subDays(3),
            '1w' => now()->subWeek(),
            '1m' => now()->subMonth(),
            default => now()->subMonth(),
        };

        // Delete old data
        $deletedCount = $host->hostData()
            ->where('report_time', '<', $cutoffDate)
            ->delete();

        Log::info("Cleaned old host data", [
            'host_id' => $host->id,
            'hostname' => $host->hostname,
            'period' => $period,
            'cutoff_date' => $cutoffDate->toDateTimeString(),
            'deleted_count' => $deletedCount,
        ]);

        return response()->json([
            'message' => 'Data cleaned successfully',
            'deleted_count' => $deletedCount,
            'cutoff_date' => $cutoffDate->toDateTimeString(),
        ]);
    }

    /**
     * Regenerate token for a host.
     */
    public function regenerateToken(Host $host)
    {
        $host->update(['token' => TokenGenerator::generate()]);

        return response()->json([
            'token' => $host->token
        ]);
    }

    /**
     * Get trend data for charts.
     */
    public function trendData(Request $request, Host $host)
    {
        $validator = Validator::make($request->all(), [
            'type' => 'required|in:load,network',
            'start_date' => 'required|date',
            'end_date' => 'required|date|after:start_date',
        ]);

        if ($validator->fails()) {
            return response()->json(['error' => $validator->errors()], 400);
        }

        $type = $request->input('type');
        $startDate = Carbon::parse($request->input('start_date'));
        $endDate = Carbon::parse($request->input('end_date'));
        $hoursDiff = $endDate->diffInHours($startDate);

        // Determine aggregation level based on time range
        if ($hoursDiff <= 24) {
            // Less than 1 day: no aggregation (raw data)
            $aggregation = null;
        } elseif ($hoursDiff <= 168) { // 7 days
            // 1-7 days: aggregate by hour
            $aggregation = 'hour';
        } elseif ($hoursDiff <= 720) { // 30 days
            // 7-30 days: aggregate by 6 hours
            $aggregation = '6hours';
        } else {
            // More than 30 days: aggregate by day
            $aggregation = 'day';
        }

        // Query host data in the specified range
        $query = $host->hostData()
            ->whereBetween('report_time', [$startDate, $endDate]);

        // For load trend
        if ($type === 'load') {
            if ($aggregation === null) {
                // No aggregation - return raw data
                $data = $query
                    ->select('id', 'report_time', 'system_load')
                    ->whereNotNull('system_load')
                    ->orderBy('report_time', 'asc')
                    ->get();

                return response()->json([
                    'data' => $data->map(function ($item) {
                        $systemLoad = $item->system_load;
                        return [
                            'report_time' => $item->report_time,
                            'load1' => round($systemLoad['load1'] ?? 0, 2),
                            'load5' => round($systemLoad['load5'] ?? 0, 2),
                            'load15' => round($systemLoad['load15'] ?? 0, 2),
                        ];
                    })
                ]);
            } elseif ($aggregation === 'hour') {
                // Aggregate by hour
                $data = $query
                    ->selectRaw("
                        HOUR(report_time) as hour,
                        DATE(report_time) as date,
                        AVG(JSON_UNQUOTE(JSON_EXTRACT(system_load, '$.load1'))) as avg_load1,
                        AVG(JSON_UNQUOTE(JSON_EXTRACT(system_load, '$.load5'))) as avg_load5,
                        AVG(JSON_UNQUOTE(JSON_EXTRACT(system_load, '$.load15'))) as avg_load15,
                        MIN(report_time) as min_time
                    ")
                    ->whereNotNull('system_load')
                    ->groupBy('date', 'hour')
                    ->orderBy('date', 'asc')
                    ->orderBy('hour', 'asc')
                    ->get();

                return response()->json([
                    'data' => $data->map(function ($item) {
                        return [
                            'report_time' => $item->min_time,
                            'load1' => round($item->avg_load1, 2),
                            'load5' => round($item->avg_load5, 2),
                            'load15' => round($item->avg_load15, 2),
                        ];
                    })
                ]);
            } elseif ($aggregation === '6hours') {
                // Aggregate by 6 hours
                $data = $query
                    ->selectRaw("
                        DATE(report_time) as date,
                        FLOOR(HOUR(report_time) / 6) as period,
                        AVG(JSON_UNQUOTE(JSON_EXTRACT(system_load, '$.load1'))) as avg_load1,
                        AVG(JSON_UNQUOTE(JSON_EXTRACT(system_load, '$.load5'))) as avg_load5,
                        AVG(JSON_UNQUOTE(JSON_EXTRACT(system_load, '$.load15'))) as avg_load15,
                        MIN(report_time) as min_time
                    ")
                    ->whereNotNull('system_load')
                    ->groupBy('date', 'period')
                    ->orderBy('date', 'asc')
                    ->orderBy('period', 'asc')
                    ->get();

                return response()->json([
                    'data' => $data->map(function ($item) {
                        return [
                            'report_time' => $item->min_time,
                            'load1' => round($item->avg_load1, 2),
                            'load5' => round($item->avg_load5, 2),
                            'load15' => round($item->avg_load15, 2),
                        ];
                    })
                ]);
            } else {
                // Aggregate by day
                $data = $query
                    ->selectRaw("
                        DATE(report_time) as date,
                        AVG(JSON_UNQUOTE(JSON_EXTRACT(system_load, '$.load1'))) as avg_load1,
                        AVG(JSON_UNQUOTE(JSON_EXTRACT(system_load, '$.load5'))) as avg_load5,
                        AVG(JSON_UNQUOTE(JSON_EXTRACT(system_load, '$.load15'))) as avg_load15,
                        MIN(report_time) as min_time
                    ")
                    ->whereNotNull('system_load')
                    ->groupBy('date')
                    ->orderBy('date', 'asc')
                    ->get();

                return response()->json([
                    'data' => $data->map(function ($item) {
                        return [
                            'report_time' => $item->min_time,
                            'load1' => round($item->avg_load1, 2),
                            'load5' => round($item->avg_load5, 2),
                            'load15' => round($item->avg_load15, 2),
                        ];
                    })
                ]);
            }
        }

        // For network trend
        if ($type === 'network') {
            if ($aggregation === null) {
                // No aggregation - return raw data
                $data = $query
                    ->select('id', 'report_time', 'network_traffic')
                    ->whereNotNull('network_traffic')
                    ->orderBy('report_time', 'asc')
                    ->get();

                return response()->json([
                    'data' => $data->map(function ($item) {
                        $networkTraffic = $item->network_traffic;
                        return [
                            'report_time' => $item->report_time,
                            'in_rate' => round($networkTraffic['total_in_bytes'] ?? 0),
                            'out_rate' => round($networkTraffic['total_out_bytes'] ?? 0),
                        ];
                    })
                ]);
            } elseif ($aggregation === 'hour') {
                // Aggregate by hour
                $data = $query
                    ->selectRaw("
                        HOUR(report_time) as hour,
                        DATE(report_time) as date,
                        AVG(JSON_UNQUOTE(JSON_EXTRACT(network_traffic, '$.total_in_bytes'))) as avg_in,
                        AVG(JSON_UNQUOTE(JSON_EXTRACT(network_traffic, '$.total_out_bytes'))) as avg_out,
                        MIN(report_time) as min_time
                    ")
                    ->whereNotNull('network_traffic')
                    ->groupBy('date', 'hour')
                    ->orderBy('date', 'asc')
                    ->orderBy('hour', 'asc')
                    ->get();

                return response()->json([
                    'data' => $data->map(function ($item) {
                        return [
                            'report_time' => $item->min_time,
                            'in_rate' => round($item->avg_in),
                            'out_rate' => round($item->avg_out),
                        ];
                    })
                ]);
            } elseif ($aggregation === '6hours') {
                // Aggregate by 6 hours
                $data = $query
                    ->selectRaw("
                        DATE(report_time) as date,
                        FLOOR(HOUR(report_time) / 6) as period,
                        AVG(JSON_UNQUOTE(JSON_EXTRACT(network_traffic, '$.total_in_bytes'))) as avg_in,
                        AVG(JSON_UNQUOTE(JSON_EXTRACT(network_traffic, '$.total_out_bytes'))) as avg_out,
                        MIN(report_time) as min_time
                    ")
                    ->whereNotNull('network_traffic')
                    ->groupBy('date', 'period')
                    ->orderBy('date', 'asc')
                    ->orderBy('period', 'asc')
                    ->get();

                return response()->json([
                    'data' => $data->map(function ($item) {
                        return [
                            'report_time' => $item->min_time,
                            'in_rate' => round($item->avg_in),
                            'out_rate' => round($item->avg_out),
                        ];
                    })
                ]);
            } else {
                // Aggregate by day
                $data = $query
                    ->selectRaw("
                        DATE(report_time) as date,
                        AVG(JSON_UNQUOTE(JSON_EXTRACT(network_traffic, '$.total_in_bytes'))) as avg_in,
                        AVG(JSON_UNQUOTE(JSON_EXTRACT(network_traffic, '$.total_out_bytes'))) as avg_out,
                        MIN(report_time) as min_time
                    ")
                    ->whereNotNull('network_traffic')
                    ->groupBy('date')
                    ->orderBy('date', 'asc')
                    ->get();

                return response()->json([
                    'data' => $data->map(function ($item) {
                        return [
                            'report_time' => $item->min_time,
                            'in_rate' => round($item->avg_in),
                            'out_rate' => round($item->avg_out),
                        ];
                    })
                ]);
            }
        }

        return response()->json(['data' => []]);
    }

    /**
     * Get LLM summaries for a host.
     */
    public function llmSummaries(Request $request, Host $host)
    {
        $query = $host->llmSummaries();

        // Filter by alert status
        if ($request->has('is_alert')) {
            $query->where('is_alert', $request->boolean('is_alert'));
        }

        // Filter by date range
        if ($request->has('start_date') && $request->has('end_date')) {
            $query->whereBetween('analysis_time', [
                $request->input('start_date'),
                $request->input('end_date')
            ]);
        }

        $summaries = $query->orderBy('analysis_time', 'desc')->paginate(20);

        return response()->json($summaries);
    }

    /**
     * Manually trigger LLM analysis for a host.
     */
    public function triggerAnalysis(Host $host)
    {
        Log::info("triggerAnalysis called for host: {$host->id} ({$host->hostname})");

        try {
            // Get recent host data (last 24 hours for manual trigger)
            $recentData = $host->hostData()
                ->where('report_time', '>=', Carbon::now()->subHours(24))
                ->orderBy('report_time', 'desc')
                ->first();

            Log::info("Recent data query result", [
                'found' => $recentData ? 'yes' : 'no',
                'time' => $recentData ? $recentData->report_time : 'N/A',
                'now_sub_24h' => Carbon::now()->subHours(24)
            ]);

            if (!$recentData) {
                // Check if there's ANY data for this host
                $anyData = $host->hostData()->orderBy('report_time', 'desc')->first();
                if ($anyData) {
                    Log::warning("Recent data not found, but host has data", [
                        'latest_time' => $anyData->report_time,
                        'hours_ago' => Carbon::now()->diffInHours($anyData->report_time)
                    ]);
                } else {
                    Log::warning("No data found at all for host {$host->id}");
                }

                return response()->json([
                    'error' => 'No recent data found for analysis. Please wait for agent to report data.'
                ], 404);
            }

            // Prepare data for LLM analysis
            $data = [
                'hostname' => $host->hostname,
                'report_time' => $recentData->report_time,
                'system_load' => $recentData->system_load,
                'network_traffic' => $recentData->network_traffic,
                'ssh_logins' => $recentData->ssh_logins,
                'public_ip' => $recentData->public_ip,
                'ip_location' => $recentData->ip_location,
            ];

            $rules = $host->alert_rules ?? '分析服务器状态，判断是否存在安全问题';

            // Perform LLM analysis
            $result = $this->llmService->analyze($data, $rules);

            // Save the analysis result
            $summary = LLMSummary::create([
                'host_id' => $host->id,
                'summary' => $result['summary'],
                'is_alert' => $result['is_alert'],
                'analysis_time' => Carbon::now(),
            ]);

            // Log if alert
            if ($result['is_alert']) {
                Log::warning("Manual LLM analysis triggered alert for host {$host->hostname}: {$result['summary']}");
            }

            return response()->json([
                'message' => 'Analysis completed successfully',
                'summary' => $summary
            ]);

        } catch (\Exception $e) {
            Log::error("Manual LLM analysis error for host {$host->id}", [
                'error' => $e->getMessage(),
                'trace' => $e->getTraceAsString(),
            ]);

            return response()->json([
                'error' => 'Analysis failed: ' . $e->getMessage()
            ], 500);
        }
    }

    /**
     * Get recent LLM analyses across all hosts for dashboard.
     */
    public function recentLLAnalyses(Request $request)
    {
        $limit = $request->input('limit', 10);

        $summaries = LLMSummary::with('host')
            ->orderBy('analysis_time', 'desc')
            ->limit($limit)
            ->get();

        return response()->json([
            'data' => $summaries->map(function ($summary) {
                return [
                    'id' => $summary->id,
                    'hostname' => $summary->host->hostname,
                    'analysis_time' => $summary->analysis_time,
                    'summary' => $summary->summary,
                    'is_alert' => $summary->is_alert,
                ];
            })
        ]);
    }

    /**
     * Get dashboard statistics.
     */
    public function dashboardStats()
    {
        $totalHosts = Host::count();
        $activeHosts = Host::where('is_active', true)->count();

        // Count alerts from LLM analyses today
        $alertsToday = LLMSummary::whereDate('analysis_time', Carbon::today())
            ->where('is_alert', true)
            ->count();

        // Total reports count
        $totalReports = HostData::count();

        return response()->json([
            'total_hosts' => $totalHosts,
            'active_hosts' => $activeHosts,
            'alerts_today' => $alertsToday,
            'total_reports' => $totalReports,
        ]);
    }
}
