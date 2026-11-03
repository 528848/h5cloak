<?php
/**
 * Plugin Name:       Cloak API
 * Plugin URI:        https://github.com/cloak-system/cloak-api-wp
 * Description:       接入斗篷系统 API 判定接口，支持站内页面、外部链接、服务器 HTML 等多种审核页展示方式。
 * Version:           1.1.0
 * Requires at least: 5.8
 * Requires PHP:      7.4
 * Author:            Cloak System
 * License:           GPL-2.0-or-later
 * Text Domain:       cloak-api
 */

if (!defined('ABSPATH')) {
    exit;
}

define('CLOAK_API_VERSION', '1.1.0');
define('CLOAK_API_FILE', __FILE__);
define('CLOAK_API_PATH', plugin_dir_path(__FILE__));
define('CLOAK_API_URL', plugin_dir_url(__FILE__));

require_once CLOAK_API_PATH . 'includes/class-cloak-settings.php';
require_once CLOAK_API_PATH . 'includes/class-cloak-api-client.php';
require_once CLOAK_API_PATH . 'includes/class-cloak-router.php';
require_once CLOAK_API_PATH . 'includes/class-cloak-api-plugin.php';

Cloak_Api_Plugin::instance();
