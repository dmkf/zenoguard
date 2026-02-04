<?php

namespace App\Http\Controllers;

use App\Models\Host;
use App\Models\HostData;
use App\Services\IPService;
use App\Services\AlertService;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Facades\Validator;

class AgentController extends Controller
{
    protected IPService $ipService;
    protected AlertService $alertService;

    public function __construct(IPService $ipService, AlertService $alertService)
    {
        $this->ipService = $ipService;
        $this->alertService = $alertService;
    }

    /**
     * Handle agent data report.
     */
    public function report(Request $request)
    {
        // Get token from bearer auth
        $token = $request->bearerToken();

        if (!$token) {
            return response()->json(['error' => 'Token required'], 401);
        }

        // Find host by token
        $host = Host::where('token', $token)->first();

        if (!$host) {
            return response()->json(['error' => 'Invalid token'], 401);
        }

        if (!$host->is_active) {
            return response()->json(['error' => 'Host is inactive'], 403);
        }

        // Validate request data
        $validator = Validator::make($request->all(), [
            'hostname' => 'required|string|max:255',
            'ssh_logins' => 'array',
            'system_load' => 'array',
            'network_traffic' => 'array',
            'public_ip' => 'nullable|ip',
        ]);

        if ($validator->fails()) {
            return response()->json(['error' => $validator->errors()], 400);
        }

        try {
            // Get IP location for host's public IP
            $publicIP = $request->input('public_ip');
            $ipLocation = $publicIP ? $this->ipService->getLocation($publicIP) : null;

            // Process SSH logins - add IP location for each
            $sshLogins = $request->input('ssh_logins', []);
            if (is_array($sshLogins)) {
                // First pass: skip private IPs and build unique IP list
                $uniqueIPs = [];
                foreach ($sshLogins as &$login) {
                    if (isset($login['ip']) && !empty($login['ip'])) {
                        // Skip private IPs
                        if ($this->ipService->isPrivateIP($login['ip'])) {
                            $login['ip_location'] = 'Private';
                        } else {
                            $uniqueIPs[$login['ip']] = true;
                        }
                    }
                }
                unset($login);

                // Second pass: lookup unique IPs only
                $ipLocations = [];
                foreach (array_keys($uniqueIPs) as $ip) {
                    $ipLocations[$ip] = $this->ipService->getLocation($ip);
                }

                // Third pass: assign locations
                foreach ($sshLogins as &$login) {
                    if (isset($login['ip']) && isset($ipLocations[$login['ip']])) {
                        $login['ip_location'] = $ipLocations[$login['ip']];
                    }
                }
                unset($login);
            }

            // Create host data record (without LLM analysis)
            $hostData = HostData::create([
                'host_id' => $host->id,
                'report_time' => now(),
                'ssh_logins' => $sshLogins,
                'system_load' => $request->input('system_load'),
                'network_traffic' => $request->input('network_traffic'),
                'public_ip' => $publicIP,
                'ip_location' => $ipLocation,
                'llm_summary' => null,  // Will be populated by scheduled LLM analysis
                'is_alert' => false,     // Will be updated by scheduled LLM analysis
            ]);

            Log::info("Agent report received from {$host->hostname}");

            return response()->json([
                'success' => true,
                'report_interval' => $host->report_interval,
            ]);

        } catch (\Exception $e) {
            Log::error("Agent report error: " . $e->getMessage());
            return response()->json(['error' => 'Internal server error'], 500);
        }
    }

    /**
     * Get current agent info.
     */
    public function me(Request $request)
    {
        $token = $request->bearerToken();
        $host = Host::where('token', $token)->first();

        if (!$host) {
            return response()->json(['error' => 'Invalid token'], 401);
        }

        return response()->json([
            'id' => $host->id,
            'hostname' => $host->hostname,
            'remark' => $host->remark,
            'report_interval' => $host->report_interval,
            'is_active' => $host->is_active,
            'created_at' => $host->created_at,
        ]);
    }

    /**
     * Get current agent's data.
     */
    public function myData(Request $request)
    {
        $token = $request->bearerToken();
        $host = Host::where('token', $token)->first();

        if (!$host) {
            return response()->json(['error' => 'Invalid token'], 401);
        }

        $query = $host->hostData()->orderBy('report_time', 'desc');

        // Paginate
        $data = $query->paginate(50);

        return response()->json($data);
    }
}
