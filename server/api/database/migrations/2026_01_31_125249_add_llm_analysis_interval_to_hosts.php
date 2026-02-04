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
        // Add llm_analysis_interval to hosts table
        Schema::table('hosts', function (Blueprint $table) {
            $table->unsignedInteger('llm_analysis_interval')->default(3600)->after('alert_rules');
            // 3600 seconds = 1 hour (default)
        });

        // Create llm_summaries table for historical analysis
        Schema::create('llm_summaries', function (Blueprint $table) {
            $table->id();
            $table->foreignId('host_id')->constrained()->onDelete('cascade');
            $table->text('summary')->comment('LLM分析总结');
            $table->boolean('is_alert')->default(false)->comment('是否触发预警');
            $table->timestamp('analysis_time')->comment('分析时间');
            $table->timestamps();

            $table->index(['host_id', 'analysis_time']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('llm_summaries');

        Schema::table('hosts', function (Blueprint $table) {
            $table->dropColumn('llm_analysis_interval');
        });
    }
};
