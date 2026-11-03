<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    public function up(): void
    {
        Schema::table('cloaks', function (Blueprint $table) {
            $table->text('white_url')->change();
            $table->text('black_url')->change();
            $table->string('remark', 200)->nullable()->change();
        });
    }

    public function down(): void
    {
        Schema::table('cloaks', function (Blueprint $table) {
            $table->string('white_url')->change();
            $table->string('black_url')->change();
            $table->string('remark')->nullable()->change();
        });
    }
};
