<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    public function up(): void
    {
        Schema::create('merchants', function (Blueprint $table) {
            $table->id();
            $table->string('name');
            $table->string('contact_name')->nullable();
            $table->string('contact_email')->nullable();
            $table->string('contact_phone')->nullable();
            $table->string('status', 32)->default('enabled')->index();
            $table->string('plan_type', 64)->nullable();
            $table->timestamp('expire_at')->nullable();
            $table->unsignedInteger('domain_quota')->default(0);
            $table->unsignedInteger('cloak_quota')->default(0);
            $table->unsignedInteger('api_quota')->default(0);
            $table->string('remark')->nullable();
            $table->timestamps();
        });

        Schema::create('users', function (Blueprint $table) {
            $table->id();
            $table->foreignId('merchant_id')->nullable()->constrained()->nullOnDelete();
            $table->string('username')->unique();
            $table->string('nickname')->nullable();
            $table->string('password');
            $table->string('avatar')->nullable();
            $table->string('email')->nullable();
            $table->string('phone')->nullable();
            $table->string('status', 32)->default('enabled')->index();
            $table->timestamp('last_login_at')->nullable();
            $table->rememberToken();
            $table->timestamps();
        });

        Schema::create('roles', function (Blueprint $table) {
            $table->id();
            $table->string('name');
            $table->string('role_key')->unique();
            $table->string('scope', 32)->default('merchant')->index();
            $table->string('status', 32)->default('enabled')->index();
            $table->unsignedInteger('sort')->default(0);
            $table->string('remark')->nullable();
            $table->timestamps();
        });

        Schema::create('menus', function (Blueprint $table) {
            $table->id();
            $table->foreignId('parent_id')->nullable()->constrained('menus')->nullOnDelete();
            $table->string('name');
            $table->string('path')->nullable();
            $table->string('component')->nullable();
            $table->string('query')->nullable();
            $table->string('menu_type', 8)->default('M')->index();
            $table->boolean('visible')->default(true);
            $table->string('status', 32)->default('enabled')->index();
            $table->string('perms')->nullable()->index();
            $table->string('icon')->nullable();
            $table->unsignedInteger('sort')->default(0);
            $table->timestamps();
        });

        Schema::create('user_roles', function (Blueprint $table) {
            $table->foreignId('user_id')->constrained()->cascadeOnDelete();
            $table->foreignId('role_id')->constrained()->cascadeOnDelete();
            $table->primary(['user_id', 'role_id']);
        });

        Schema::create('role_menus', function (Blueprint $table) {
            $table->foreignId('role_id')->constrained()->cascadeOnDelete();
            $table->foreignId('menu_id')->constrained()->cascadeOnDelete();
            $table->primary(['role_id', 'menu_id']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('role_menus');
        Schema::dropIfExists('user_roles');
        Schema::dropIfExists('menus');
        Schema::dropIfExists('roles');
        Schema::dropIfExists('users');
        Schema::dropIfExists('merchants');
    }
};
