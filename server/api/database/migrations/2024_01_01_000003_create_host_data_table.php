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
        Schema::create('host_data', function (Blueprint $table) {
            $table->id();
            $table->unsignedBigInteger('host_id');
            $table->timestamp('report_time');
            $table->json('ssh_logins')->nullable();
            $table->json('system_load')->nullable();
            $table->json('network_traffic')->nullable();
            $table->string('public_ip', 45)->nullable();
            $table->string('ip_location')->nullable();
            $table->text('llm_summary')->nullable();
            $table->boolean('is_alert')->default(false);
            $table->timestamps();

            $table->foreign('host_id')->references('id')->on('hosts')->onDelete('cascade');
            $table->index('host_id');
            $table->index('report_time');
            $table->index('is_alert');
            $table->index(['host_id', 'report_time', 'is_alert']);
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('host_data');
    }
};
