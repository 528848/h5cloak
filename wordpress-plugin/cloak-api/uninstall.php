<?php

if (!defined('WP_UNINSTALL_PLUGIN')) {
    exit;
}

delete_option('cloak_api_settings');
