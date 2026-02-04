<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('alert_configs', function (Blueprint $table) {
            $table->id();
            $table->string('platform', 50); // dingtalk, etc.
            $table->string('webhook_url', 512);
            $table->string('secret')->nullable();
            $table->boolean('is_active')->default(true);
            $table->timestamps();

            $table->index('platform');
            $table->index('is_active');
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('alert_configs');
    }
};
