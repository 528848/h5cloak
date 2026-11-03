<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    public function up(): void
    {
        Schema::create('domains', function (Blueprint $table) {
            $table->id();
            $table->foreignId('merchant_id')->constrained()->cascadeOnDelete();
            $table->string('domain')->unique();
            $table->string('verify_type', 32)->default('CNAME');
            $table->string('verify_value')->nullable();
            $table->string('status', 32)->default('pending')->index();
            $table->timestamp('last_checked_at')->nullable();
            $table->string('remark')->nullable();
            $table->timestamps();

            $table->index(['merchant_id', 'status']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('domains');
    }
};
