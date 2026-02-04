<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;
use Illuminate\Support\Facades\DB;

return new class extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        // Check if columns don't exist before adding them
        if (!Schema::hasColumn('llm_configs', 'system_prompt')) {
            Schema::table('llm_configs', function (Blueprint $table) {
                $table->text('system_prompt')->nullable()->after('is_active');
            });
        }

        if (!Schema::hasColumn('llm_configs', 'user_prompt')) {
            Schema::table('llm_configs', function (Blueprint $table) {
                $table->text('user_prompt')->nullable()->after('system_prompt');
            });
        }

        // Set default values for existing records
        $defaultSystemPrompt = '你是一个服务器安全分析专家。请根据服务器数据判断是否存在安全问题，并给出简明扼要的分析结果。只返回JSON格式：{"summary": "一句话总结", "is_alert": true/false}';
        $defaultUserPrompt = '当前服务器数据：
{DATA}

预警规则：{RULES}

请分析：
1. 用一句话总结服务器状态（不超过50字）
2. 判断是否触发预警（true/false）

只返回JSON格式：{"summary": "一句话总结", "is_alert": true/false}';

        DB::statement('UPDATE llm_configs SET system_prompt = ? WHERE system_prompt IS NULL', [$defaultSystemPrompt]);
        DB::statement('UPDATE llm_configs SET user_prompt = ? WHERE user_prompt IS NULL', [$defaultUserPrompt]);
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('llm_configs', function (Blueprint $table) {
            $table->dropColumn(['system_prompt', 'user_prompt']);
        });
    }
};
