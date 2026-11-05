<?php
/**
 * Plugin Name:       Cloak Attribution
 * Plugin URI:        https://github.com/cloak-system/cloak-attribution-wp
 * Description:       斗篷系统 v2 接入：访客带归因参数看到真实页面，否则看到审核内容，地址不变。支持 WooCommerce、跨浏览器交接引导。
 * Version:           2.1.0
 * Requires at least: 5.8
 * Requires PHP:      7.4
 * Author:            Cloak System
 * License:           GPL-2.0-or-later
 * Text Domain:       cloak-attribution
 */

if (!defined('ABSPATH')) {
    exit;
}

define('CLOAK_ATTR_VERSION', '2.1.0');
define('CLOAK_ATTR_FILE', __FILE__);
define('CLOAK_ATTR_PATH', plugin_dir_path(__FILE__));
define('CLOAK_ATTR_URL', plugin_dir_url(__FILE__));

require_once CLOAK_ATTR_PATH . 'includes/class-cloak-attr-settings.php';
require_once CLOAK_ATTR_PATH . 'includes/class-cloak-attr-signer.php';
require_once CLOAK_ATTR_PATH . 'includes/class-cloak-attr-client.php';
require_once CLOAK_ATTR_PATH . 'includes/class-cloak-attr-woo.php';
require_once CLOAK_ATTR_PATH . 'includes/class-cloak-attr-gate.php';

register_activation_hook(__FILE__, ['Cloak_Attr_Settings', 'migrate_from_v1']);

add_action('plugins_loaded', static function () {
    if (is_admin()) {
        require_once CLOAK_ATTR_PATH . 'admin/class-cloak-attr-admin.php';
        new Cloak_Attr_Admin();
    } else {
        new Cloak_Attr_Gate();
    }
});
