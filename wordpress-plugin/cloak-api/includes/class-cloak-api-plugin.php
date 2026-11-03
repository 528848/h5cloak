<?php

if (!defined('ABSPATH')) {
    exit;
}

final class Cloak_Api_Plugin
{
    private static ?self $instance = null;

    public static function instance(): self
    {
        if (self::$instance === null) {
            self::$instance = new self();
        }

        return self::$instance;
    }

    private function __construct()
    {
        add_action('plugins_loaded', [$this, 'init']);
    }

    public function init(): void
    {
        if (is_admin()) {
            require_once CLOAK_API_PATH . 'admin/class-cloak-admin.php';
            new Cloak_Admin();
        }

        if (!is_admin()) {
            new Cloak_Router();
        }
    }
}
