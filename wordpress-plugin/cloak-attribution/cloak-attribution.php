<?php
/**
 * Plugin Name:       Cloak Attribution
 * Plugin URI:        https://github.com/cloak-system/cloak-attribution-wp
 * Description:       斗篷系统 v2 接入：访客带归因参数看到真实页面，否则看到审核内容，地址不变。支持服务端 SDK 深度代理、WooCommerce、跨浏览器交接引导。
 * Version:           2.3.8
 * Requires at least: 5.6
 * Requires PHP:      7.4
 * Tested up to:      6.8
 * Author:            Cloak System
 * License:           GPL-2.0-or-later
 * Text Domain:       cloak-attribution
 */

if (!defined('ABSPATH')) {
    exit;
}

define('CLOAK_ATTR_VERSION', '2.3.8');
define('CLOAK_ATTR_FILE', __FILE__);
define('CLOAK_ATTR_PATH', plugin_dir_path(__FILE__));
define('CLOAK_ATTR_URL', plugin_dir_url(__FILE__));

// 服务端 SDK（与 Go/网关一致的深度代理改写与签名），由插件继承复用。
require_once CLOAK_ATTR_PATH . 'includes/sdk/Rewriter.php';
require_once CLOAK_ATTR_PATH . 'includes/sdk/Signer.php';
require_once CLOAK_ATTR_PATH . 'includes/sdk/ResponseCache.php';

require_once CLOAK_ATTR_PATH . 'includes/class-cloak-attr-settings.php';
require_once CLOAK_ATTR_PATH . 'includes/class-cloak-attr-signer.php';
require_once CLOAK_ATTR_PATH . 'includes/class-cloak-attr-client.php';
require_once CLOAK_ATTR_PATH . 'includes/class-cloak-attr-deep-proxy.php';
require_once CLOAK_ATTR_PATH . 'includes/class-cloak-attr-woo.php';
require_once CLOAK_ATTR_PATH . 'includes/class-cloak-attr-gate.php';

register_activation_hook(__FILE__, ['Cloak_Attr_Settings', 'migrate_from_v1']);

add_action('plugins_loaded', static function () {
    // 任何加载异常都不能影响站点前台。
    try {
        if (is_admin()) {
            require_once CLOAK_ATTR_PATH . 'admin/class-cloak-attr-admin.php';
            new Cloak_Attr_Admin();
        } else {
            new Cloak_Attr_Gate();
        }
    } catch (Throwable $e) {
        if (function_exists('error_log')) {
            error_log('cloak-attribution boot: ' . $e->getMessage());
        }
    }
});
