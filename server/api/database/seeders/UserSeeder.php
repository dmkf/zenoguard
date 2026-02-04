<?php

namespace Database\Seeders;

use App\Models\User;
use Illuminate\Database\Seeder;
use Illuminate\Support\Facades\Hash;

class UserSeeder extends Seeder
{
    /**
     * Run the database seeds.
     */
    public function run(): void
    {
        // Create default admin user
        User::updateOrCreate(
            ['username' => 'xiong'],
            [
                'password' => Hash::make('6633669'),
            ]
        );

        $this->command->info('Default admin user created (username: xiong, password: 6633669)');
    }
}
