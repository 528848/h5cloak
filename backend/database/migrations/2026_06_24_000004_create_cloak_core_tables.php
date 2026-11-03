<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    public function up(): void
    {
        Schema::create('cloaks', function (Blueprint $table) {
            $table->id();
            $table->foreignId('merchant_id')->constrained()->cascadeOnDelete();
            $table->foreignId('domain_id')->constrained()->cascadeOnDelete();
            $table->string('name');
            $table->string('status', 32)->default('disabled')->index();
            $table->string('white_url');
            $table->string('black_url');
            $table->string('default_action', 32)->default('block');
            $table->json('rules')->nullable();
            $table->string('remark')->nullable();
            $table->timestamps();

            $table->index(['merchant_id', 'status']);
            $table->index(['domain_id', 'status']);
        });

        Schema::create('access_lists', function (Blueprint $table) {
            $table->id();
            $table->foreignId('merchant_id')->constrained()->cascadeOnDelete();
            $table->foreignId('cloak_id')->nullable()->constrained()->cascadeOnDelete();
            $table->string('list_type', 16)->index();
            $table->string('target_type', 32)->index();
            $table->string('target_value');
            $table->string('scope', 32)->default('merchant')->index();
            $table->string('status', 32)->default('enabled')->index();
            $table->string('remark')->nullable();
            $table->timestamps();

            $table->index(['merchant_id', 'cloak_id', 'status']);
        });

        Schema::create('cloak_access_logs', function (Blueprint $table) {
            $table->id();
            $table->string('request_id')->index();
            $table->foreignId('merchant_id')->nullable()->constrained()->nullOnDelete();
            $table->foreignId('domain_id')->nullable()->constrained()->nullOnDelete();
            $table->foreignId('cloak_id')->nullable()->constrained()->nullOnDelete();
            $table->string('ip_hash')->nullable()->index();
            $table->string('country')->nullable();
            $table->string('region')->nullable();
            $table->string('device_type')->nullable();
            $table->string('user_agent_summary')->nullable();
            $table->string('referer_summary')->nullable();
            $table->string('hit_rule')->nullable();
            $table->string('final_action', 32)->index();
            $table->unsignedInteger('latency_ms')->default(0);
            $table->timestamp('created_at')->nullable()->index();

            $table->index(['merchant_id', 'created_at']);
            $table->index(['domain_id', 'created_at']);
            $table->index(['cloak_id', 'created_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('cloak_access_logs');
        Schema::dropIfExists('access_lists');
        Schema::dropIfExists('cloaks');
    }
};
