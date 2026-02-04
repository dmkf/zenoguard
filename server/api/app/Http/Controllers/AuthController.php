<?php

namespace App\Http\Controllers;

use App\Models\User;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\Hash;
use Illuminate\Validation\ValidationException;

class AuthController extends Controller
{
    /**
     * Login and generate token.
     */
    public function login(Request $request)
    {
        $request->validate([
            'username' => 'required|string',
            'password' => 'required|string',
        ]);

        $user = User::where('username', $request->username)->first();

        if (!$user || !Hash::check($request->password, $user->password)) {
            throw ValidationException::withMessages([
                'username' => ['The provided credentials are incorrect.'],
            ]);
        }

        // Delete old tokens
        $user->tokens()->delete();

        // Create new token
        $token = $user->createToken('auth-token')->plainTextToken;

        return response()->json([
            'user' => [
                'id' => $user->id,
                'username' => $user->username,
            ],
            'token' => $token,
        ]);
    }

    /**
     * Logout and revoke token.
     */
    public function logout(Request $request)
    {
        $user = $request->user();

        if (!$user) {
            return response()->json([
                'error' => 'Not authenticated'
            ], 401);
        }

        // Delete current access token
        $user->currentAccessToken()->delete();

        return response()->json([
            'message' => 'Logged out successfully'
        ]);
    }

    /**
     * Get current user info.
     */
    public function me(Request $request)
    {
        return response()->json([
            'user' => [
                'id' => $request->user()->id,
                'username' => $request->user()->username,
            ]
        ]);
    }

    /**
     * Change password.
     */
    public function changePassword(Request $request)
    {
        $request->validate([
            'current_password' => 'required|string',
            'new_password' => 'required|string|min:6|confirmed',
        ]);

        $user = $request->user();

        // Verify current password
        if (!Hash::check($request->current_password, $user->password)) {
            return response()->json([
                'error' => 'Current password is incorrect'
            ], 400);
        }

        // Update password
        $user->password = Hash::make($request->new_password);
        $user->save();

        return response()->json([
            'message' => 'Password changed successfully'
        ]);
    }
}
