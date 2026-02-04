<?php

namespace App\Console;

use Illuminate\Foundation\Console\Kernel as ConsoleKernel;

class Kernel extends ConsoleKernel
{
    /**
     * The Artisan commands provided by your application.
     */
    protected $commands = [
        \App\Console\Commands\AnalyzeWithLLM::class,
    ];

    /**
     * Define the application's command schedule.
     */
    protected function schedule(\Illuminate\Console\Scheduling\Schedule $schedule): void
    {
        // Run LLM analysis every 5 minutes
        // Each host will check its own llm_analysis_interval to determine if analysis is due
        $schedule->command('llm:analyze')
            ->everyFiveMinutes()
            ->withoutOverlapping()
            ->runInBackground()
            ->appendOutputTo(storage_path('logs/llm-analysis.log'));
    }

    /**
     * Register the commands for the application.
     */
    protected function commands(): void
    {
        $this->load(__DIR__.'/Routes');

        require base_path('routes/console.php');
    }
}
