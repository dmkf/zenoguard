<?php

namespace App\Http\Middleware;

use App\Models\Host;
use Closure;
use Illuminate\Http\Request;

class TokenAuth
{
    /**
     * Handle an incoming request.
     */
    public function handle(Request $request, Closure $next)
    {
        $token = $request->bearerToken();

        if (!$token) {
            return response()->json(['error' => 'Token required'], 401);
        }

        $host = Host::where('token', $token)->first();

        if (!$host) {
            return response()->json(['error' => 'Invalid token'], 401);
        }

        if (!$host->is_active) {
            return response()->json(['error' => 'Host is inactive'], 403);
        }

        // Attach host to request for later use
        $request->attributes->set('host', $host);

        return $next($request);
    }
}
