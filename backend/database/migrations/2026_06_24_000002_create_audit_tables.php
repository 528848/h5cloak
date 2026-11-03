<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    public function up(): void
    {
        Schema::create('login_logs', function (Blueprint $table) {
            $table->id();
            $table->string('username')->nullable()->index();
            $table->string('ip', 64)->nullable();
            $table->string('location')->nullable();
            $table->string('browser')->nullable();
            $table->string('os')->nullable();
            $table->string('status', 32)->index();
            $table->string('message')->nullable();
            $table->timestamp('created_at')->nullable()->index();
        });

        Schema::create('operation_logs', function (Blueprint $table) {
            $table->id();
            $table->string('request_id')->nullable()->index();
            $table->foreignId('actor_user_id')->nullable()->constrained('users')->nullOnDelete();
            $table->string('actor_role')->nullable();
            $table->foreignId('merchant_id')->nullable()->constrained()->nullOnDelete();
            $table->string('module')->index();
            $table->string('action')->index();
            $table->string('object_id')->nullable();
            $table->json('before_snapshot')->nullable();
            $table->json('after_snapshot')->nullable();
            $table->string('ip', 64)->nullable();
            $table->timestamp('created_at')->nullable()->index();
        });

        Schema::create('personal_access_tokens', function (Blueprint $table) {
            $table->id();
            $table->morphs('tokenable');
            $table->string('name');
            $table->string('token', 64)->unique();
            $table->text('abilities')->nullable();
            $table->timestamp('last_used_at')->nullable();
            $table->timestamp('expires_at')->nullable();
            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('personal_access_tokens');
        Schema::dropIfExists('operation_logs');
        Schema::dropIfExists('login_logs');
    }
};
