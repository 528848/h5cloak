<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    public function up(): void
    {
        Schema::create('api_keys', function (Blueprint $table) {
            $table->id();
            $table->foreignId('merchant_id')->constrained()->cascadeOnDelete();
            $table->string('name');
            $table->string('key_prefix', 32)->index();
            $table->string('secret_hash');
            $table->json('permissions')->nullable();
            $table->json('rules')->nullable();
            $table->string('remark')->nullable();
            $table->string('status', 32)->default('enabled')->index();
            $table->timestamp('last_called_at')->nullable();
            $table->timestamps();

            $table->index(['merchant_id', 'status']);
        });

        Schema::create('api_call_logs', function (Blueprint $table) {
            $table->id();
            $table->string('request_id')->index();
            $table->foreignId('merchant_id')->nullable()->constrained()->nullOnDelete();
            $table->foreignId('api_key_id')->nullable()->constrained('api_keys')->nullOnDelete();
            $table->string('api_name')->index();
            $table->string('method', 16);
            $table->string('path');
            $table->string('request_ip', 64)->nullable();
            $table->unsignedSmallInteger('status_code')->default(200)->index();
            $table->string('error_code')->nullable()->index();
            $table->unsignedInteger('latency_ms')->default(0);
            $table->timestamp('created_at')->nullable()->index();

            $table->index(['merchant_id', 'created_at']);
        });

        Schema::create('dashboard_metrics', function (Blueprint $table) {
            $table->id();
            $table->foreignId('merchant_id')->nullable()->constrained()->cascadeOnDelete();
            $table->date('metric_date')->index();
            $table->unsignedTinyInteger('metric_hour')->nullable();
            $table->unsignedInteger('visits')->default(0);
            $table->unsignedInteger('white_hits')->default(0);
            $table->unsignedInteger('black_hits')->default(0);
            $table->unsignedInteger('blocks')->default(0);
            $table->unsignedInteger('api_calls')->default(0);
            $table->timestamps();

            $table->unique(['merchant_id', 'metric_date', 'metric_hour']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('dashboard_metrics');
        Schema::dropIfExists('api_call_logs');
        Schema::dropIfExists('api_keys');
    }
};
