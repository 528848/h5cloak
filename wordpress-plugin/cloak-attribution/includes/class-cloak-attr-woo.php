<?php

if (!defined('ABSPATH')) {
    exit;
}

/** WooCommerce 范围判断：当前请求是否属于需要判定的页面。 */
class Cloak_Attr_Woo
{
    public static function active(): bool
    {
        return class_exists('WooCommerce');
    }

    /**
     * 返回 true 表示 Woo 规则允许判定；false 表示 Woo 规则排除本页。
     * 非 Woo 页面（普通文章/页面）始终返回 true，由全站逻辑处理。
     */
    public static function should_gate(array $settings): bool
    {
        if (!self::active() || empty($settings['woo_enabled'])) {
            return true;
        }
        $is_woo_page = (function_exists('is_woocommerce') && is_woocommerce())
            || (function_exists('is_cart') && is_cart())
            || (function_exists('is_checkout') && is_checkout())
            || (function_exists('is_account_page') && is_account_page());
        if (!$is_woo_page) {
            return true;
        }
        switch ($settings['woo_scope']) {
            case 'shop_only':
                return true; // 所有 Woo 页面都判定
            case 'products':
                return self::product_matches($settings);
            case 'all':
            default:
                return true;
        }
    }

    private static function product_matches(array $settings): bool
    {
        // 购物车/结账/支付页：按购物车内商品判断
        if ((function_exists('is_cart') && is_cart()) || (function_exists('is_checkout') && is_checkout())) {
            if (function_exists('WC') && WC()->cart) {
                foreach (WC()->cart->get_cart() as $item) {
                    if (self::id_or_cat_match((int) ($item['product_id'] ?? 0), $settings)) {
                        return true;
                    }
                }
            }
            return false;
        }
        if (function_exists('is_product') && is_product()) {
            return self::id_or_cat_match(get_queried_object_id(), $settings);
        }
        if (function_exists('is_product_category') && is_product_category()) {
            $term = get_queried_object();
            $slugs = self::lines($settings['woo_category_slugs']);
            return $term && in_array($term->slug, $slugs, true);
        }
        return false;
    }

    private static function id_or_cat_match(int $product_id, array $settings): bool
    {
        if ($product_id <= 0) {
            return false;
        }
        $ids = array_map('intval', self::lines($settings['woo_product_ids']));
        if (in_array($product_id, $ids, true)) {
            return true;
        }
        $slugs = self::lines($settings['woo_category_slugs']);
        return !empty($slugs) && has_term($slugs, 'product_cat', $product_id);
    }

    private static function lines(string $value): array
    {
        return array_values(array_filter(array_map('trim', preg_split('/\r\n|\r|\n|,/', $value))));
    }
}
