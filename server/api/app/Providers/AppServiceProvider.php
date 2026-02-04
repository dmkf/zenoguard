<?php

namespace App\Providers;

use Illuminate\Support\ServiceProvider;

class AppServiceProvider extends ServiceProvider
{
    /**
     * Register any application services.
     */
    public function register(): void
    {
        // Ensure view paths are set before ViewServiceProvider registers
        $this->app->config->set('view.paths', [base_path('resources/views')]);
        $this->app->config->set('view.compiled', base_path('storage/framework/views'));
    }

    /**
     * Bootstrap any application services.
     */
    public function boot(): void
    {
        //
    }
}
