<?php

if (!defined('ABSPATH')) {
    exit;
}

$option_key = Cloak_Settings::OPTION_KEY;
?>
<div class="wrap cloak-api-wrap">
    <h1><?php esc_html_e('API 斗篷设置', 'cloak-api'); ?></h1>

    <p class="description">
        <?php esc_html_e('在服务端调用斗篷判定接口，根据返回结果展示审核页或真实页。Token 仅保存在服务器，不会暴露给访客。', 'cloak-api'); ?>
    </p>

    <form method="post" action="options.php" id="cloak-api-settings-form">
        <?php settings_fields('cloak_api_settings_group'); ?>

        <div class="cloak-api-card">
            <h2><?php esc_html_e('基础配置', 'cloak-api'); ?></h2>

            <table class="form-table" role="presentation">
                <tr>
                    <th scope="row"><?php esc_html_e('启用斗篷', 'cloak-api'); ?></th>
                    <td>
                        <label>
                            <input type="checkbox" name="<?php echo esc_attr($option_key); ?>[enabled]" value="1" <?php checked(!empty($settings['enabled'])); ?> />
                            <?php esc_html_e('开启后，前台访问将按规则判定并跳转', 'cloak-api'); ?>
                        </label>
                    </td>
                </tr>
                <tr>
                    <th scope="row"><label for="cloak_api_url"><?php esc_html_e('API 地址', 'cloak-api'); ?></label></th>
                    <td>
                        <input
                            type="url"
                            class="regular-text code"
                            id="cloak_api_url"
                            name="<?php echo esc_attr($option_key); ?>[api_url]"
                            value="<?php echo esc_attr($settings['api_url']); ?>"
                            placeholder="https://your-cloak-domain.com/api/open/decision"
                            <?php disabled($url_from_config); ?>
                        />
                        <?php if ($url_from_config) : ?>
                            <p class="description"><?php esc_html_e('已在 wp-config.php 中通过 CLOAK_API_URL 定义，此处不可修改。', 'cloak-api'); ?></p>
                        <?php endif; ?>
                    </td>
                </tr>
                <tr>
                    <th scope="row"><label for="cloak_api_token"><?php esc_html_e('API Token', 'cloak-api'); ?></label></th>
                    <td>
                        <input
                            type="password"
                            class="regular-text code"
                            id="cloak_api_token"
                            name="<?php echo esc_attr($option_key); ?>[api_token]"
                            value=""
                            placeholder="<?php echo esc_attr($settings['api_token'] ? '********' : 'ck_xxxxxxxx'); ?>"
                            autocomplete="new-password"
                            <?php disabled($token_from_config); ?>
                        />
                        <?php if ($token_from_config) : ?>
                            <p class="description"><?php esc_html_e('已在 wp-config.php 中通过 CLOAK_API_TOKEN 定义，此处不可修改。', 'cloak-api'); ?></p>
                        <?php else : ?>
                            <p class="description"><?php esc_html_e('留空则保持已保存的 Token 不变。', 'cloak-api'); ?></p>
                        <?php endif; ?>
                        <p>
                            <button type="button" class="button" id="cloak-api-test-btn"><?php esc_html_e('测试 API 连接', 'cloak-api'); ?></button>
                            <span id="cloak-api-test-result"></span>
                        </p>
                    </td>
                </tr>
                <tr>
                    <th scope="row"><?php esc_html_e('生效范围', 'cloak-api'); ?></th>
                    <td>
                        <select name="<?php echo esc_attr($option_key); ?>[scope]">
                            <option value="entire_site" <?php selected($settings['scope'], 'entire_site'); ?>><?php esc_html_e('全站', 'cloak-api'); ?></option>
                            <option value="homepage_only" <?php selected($settings['scope'], 'homepage_only'); ?>><?php esc_html_e('仅首页', 'cloak-api'); ?></option>
                        </select>
                    </td>
                </tr>
                <tr>
                    <th scope="row"><label for="cloak_exclude_paths"><?php esc_html_e('排除路径', 'cloak-api'); ?></label></th>
                    <td>
                        <textarea
                            class="large-text code"
                            rows="4"
                            id="cloak_exclude_paths"
                            name="<?php echo esc_attr($option_key); ?>[exclude_paths]"
                            placeholder="/blog/*&#10;/shop/checkout&#10;/privacy-policy"
                        ><?php echo esc_textarea($settings['exclude_paths']); ?></textarea>
                        <p class="description">
                            <?php esc_html_e('每行一条路径规则，匹配时跳过斗篷判定。支持精确路径（如 /shop/checkout）和前缀通配（如 /blog/*）。', 'cloak-api'); ?>
                        </p>
                    </td>
                </tr>
                <tr>
                    <th scope="row"><?php esc_html_e('接口失败时', 'cloak-api'); ?></th>
                    <td>
                        <select name="<?php echo esc_attr($option_key); ?>[fail_mode]">
                            <option value="review" <?php selected($settings['fail_mode'], 'review'); ?>><?php esc_html_e('默认展示审核页（推荐）', 'cloak-api'); ?></option>
                            <option value="real" <?php selected($settings['fail_mode'], 'real'); ?>><?php esc_html_e('默认展示真实页', 'cloak-api'); ?></option>
                        </select>
                    </td>
                </tr>
            </table>
        </div>

        <div class="cloak-api-card">
            <h2><?php esc_html_e('审核页（白页）配置', 'cloak-api'); ?></h2>

            <table class="form-table" role="presentation">
                <tr>
                    <th scope="row"><?php esc_html_e('审核页方式', 'cloak-api'); ?></th>
                    <td>
                        <select name="<?php echo esc_attr($option_key); ?>[review_mode]" id="cloak_review_mode">
                            <?php foreach ($review_modes as $value => $label) : ?>
                                <option value="<?php echo esc_attr($value); ?>" <?php selected($settings['review_mode'], $value); ?>>
                                    <?php echo esc_html($label); ?>
                                </option>
                            <?php endforeach; ?>
                        </select>
                    </td>
                </tr>

                <tr class="cloak-mode-field cloak-mode-wp_page" data-mode="wp_page">
                    <th scope="row"><label for="cloak_review_page_id"><?php esc_html_e('审核页', 'cloak-api'); ?></label></th>
                    <td>
                        <select name="<?php echo esc_attr($option_key); ?>[review_page_id]" id="cloak_review_page_id">
                            <option value="0"><?php esc_html_e('— 请选择页面 —', 'cloak-api'); ?></option>
                            <?php foreach ($pages as $page) : ?>
                                <option value="<?php echo esc_attr($page->ID); ?>" <?php selected((int) $settings['review_page_id'], (int) $page->ID); ?>>
                                    <?php echo esc_html($page->post_title); ?> (ID: <?php echo esc_html($page->ID); ?>)
                                </option>
                            <?php endforeach; ?>
                        </select>
                    </td>
                </tr>

                <tr class="cloak-mode-field cloak-mode-external_url" data-mode="external_url">
                    <th scope="row"><label for="cloak_review_external_url"><?php esc_html_e('外部审核页 URL', 'cloak-api'); ?></label></th>
                    <td>
                        <input
                            type="url"
                            class="large-text code"
                            id="cloak_review_external_url"
                            name="<?php echo esc_attr($option_key); ?>[review_external_url]"
                            value="<?php echo esc_attr($settings['review_external_url']); ?>"
                            placeholder="https://safe.example.com/landing"
                        />
                        <p class="description"><?php esc_html_e('仅支持 https 链接。', 'cloak-api'); ?></p>
                    </td>
                </tr>

                <tr class="cloak-mode-field cloak-mode-external_url" data-mode="external_url">
                    <th scope="row"><label for="cloak_external_allowed_hosts"><?php esc_html_e('允许的外部域名', 'cloak-api'); ?></label></th>
                    <td>
                        <textarea
                            class="large-text code"
                            rows="3"
                            id="cloak_external_allowed_hosts"
                            name="<?php echo esc_attr($option_key); ?>[external_allowed_hosts]"
                            placeholder="safe.example.com&#10;landing.example.com"
                        ><?php echo esc_textarea($settings['external_allowed_hosts']); ?></textarea>
                        <p class="description"><?php esc_html_e('每行一个域名。审核页与真实页的外部链接域名均需在此列出；留空时将自动收录已填写的 URL 域名。', 'cloak-api'); ?></p>
                    </td>
                </tr>

                <tr class="cloak-mode-field cloak-mode-html_redirect" data-mode="html_redirect">
                    <th scope="row"><label for="cloak_review_html_url"><?php esc_html_e('HTML 文件 URL', 'cloak-api'); ?></label></th>
                    <td>
                        <input
                            type="url"
                            class="large-text code"
                            id="cloak_review_html_url"
                            name="<?php echo esc_attr($option_key); ?>[review_html_url]"
                            value="<?php echo esc_attr($settings['review_html_url']); ?>"
                            placeholder="<?php echo esc_attr(home_url('/wp-content/cloak/review.html')); ?>"
                        />
                        <p class="description"><?php esc_html_e('将 review.html 上传至可访问目录，填写完整 URL。', 'cloak-api'); ?></p>
                    </td>
                </tr>

                <tr class="cloak-mode-field cloak-mode-html_output" data-mode="html_output">
                    <th scope="row"><label for="cloak_review_html_path"><?php esc_html_e('HTML 文件路径', 'cloak-api'); ?></label></th>
                    <td>
                        <input
                            type="text"
                            class="large-text code"
                            id="cloak_review_html_path"
                            name="<?php echo esc_attr($option_key); ?>[review_html_path]"
                            value="<?php echo esc_attr($settings['review_html_path']); ?>"
                            placeholder="/var/www/cloak-pages/review.html"
                        />
                        <p class="description"><?php esc_html_e('服务器绝对路径。PHP 直接输出内容，地址栏 URL 不变。可放在 Web 目录外。', 'cloak-api'); ?></p>
                    </td>
                </tr>
            </table>
        </div>

        <div class="cloak-api-card">
            <h2><?php esc_html_e('真实页（黑页）配置', 'cloak-api'); ?></h2>

            <table class="form-table" role="presentation">
                <tr>
                    <th scope="row"><?php esc_html_e('真实页方式', 'cloak-api'); ?></th>
                    <td>
                        <select name="<?php echo esc_attr($option_key); ?>[real_mode]" id="cloak_real_mode">
                            <?php foreach ($review_modes as $value => $label) : ?>
                                <option value="<?php echo esc_attr($value); ?>" <?php selected($settings['real_mode'], $value); ?>>
                                    <?php echo esc_html($label); ?>
                                </option>
                            <?php endforeach; ?>
                        </select>
                    </td>
                </tr>

                <tr class="cloak-real-mode-field cloak-real-mode-wp_page" data-mode="wp_page">
                    <th scope="row"><label for="cloak_real_page_id"><?php esc_html_e('真实页', 'cloak-api'); ?></label></th>
                    <td>
                        <select name="<?php echo esc_attr($option_key); ?>[real_page_id]" id="cloak_real_page_id">
                            <option value="0"><?php esc_html_e('— 请选择页面 —', 'cloak-api'); ?></option>
                            <?php foreach ($pages as $page) : ?>
                                <option value="<?php echo esc_attr($page->ID); ?>" <?php selected((int) $settings['real_page_id'], (int) $page->ID); ?>>
                                    <?php echo esc_html($page->post_title); ?> (ID: <?php echo esc_html($page->ID); ?>)
                                </option>
                            <?php endforeach; ?>
                        </select>
                    </td>
                </tr>

                <tr class="cloak-real-mode-field cloak-real-mode-external_url" data-mode="external_url">
                    <th scope="row"><label for="cloak_real_external_url"><?php esc_html_e('外部真实页 URL', 'cloak-api'); ?></label></th>
                    <td>
                        <input
                            type="url"
                            class="large-text code"
                            id="cloak_real_external_url"
                            name="<?php echo esc_attr($option_key); ?>[real_external_url]"
                            value="<?php echo esc_attr($settings['real_external_url']); ?>"
                            placeholder="https://shop.example.com/offer"
                        />
                        <p class="description"><?php esc_html_e('仅支持 https。域名需在上方审核页配置的「允许的外部域名」白名单中。', 'cloak-api'); ?></p>
                    </td>
                </tr>

                <tr class="cloak-real-mode-field cloak-real-mode-html_redirect" data-mode="html_redirect">
                    <th scope="row"><label for="cloak_real_html_url"><?php esc_html_e('HTML 文件 URL', 'cloak-api'); ?></label></th>
                    <td>
                        <input
                            type="url"
                            class="large-text code"
                            id="cloak_real_html_url"
                            name="<?php echo esc_attr($option_key); ?>[real_html_url]"
                            value="<?php echo esc_attr($settings['real_html_url']); ?>"
                            placeholder="<?php echo esc_attr(home_url('/wp-content/cloak/real.html')); ?>"
                        />
                    </td>
                </tr>

                <tr class="cloak-real-mode-field cloak-real-mode-html_output" data-mode="html_output">
                    <th scope="row"><label for="cloak_real_html_path"><?php esc_html_e('HTML 文件路径', 'cloak-api'); ?></label></th>
                    <td>
                        <input
                            type="text"
                            class="large-text code"
                            id="cloak_real_html_path"
                            name="<?php echo esc_attr($option_key); ?>[real_html_path]"
                            value="<?php echo esc_attr($settings['real_html_path']); ?>"
                            placeholder="/var/www/cloak-pages/real.html"
                        />
                        <p class="description"><?php esc_html_e('服务器绝对路径，PHP 直接输出，地址栏 URL 不变。', 'cloak-api'); ?></p>
                    </td>
                </tr>
            </table>
        </div>

        <div class="cloak-api-card cloak-api-card--notice">
            <h2><?php esc_html_e('使用提示', 'cloak-api'); ?></h2>
            <ul class="cloak-api-tips">
                <li><?php esc_html_e('WP Rocket、LiteSpeed 等全页缓存需排除前台，否则斗篷可能失效。', 'cloak-api'); ?></li>
                <li><?php esc_html_e('可在斗篷后台「API 调用日志」查看每次判定结果。', 'cloak-api'); ?></li>
                <li><?php esc_html_e('可在 wp-config.php 中定义 CLOAK_API_TOKEN 和 CLOAK_API_URL 覆盖敏感配置。', 'cloak-api'); ?></li>
            </ul>
        </div>

        <?php submit_button(__('保存设置', 'cloak-api')); ?>
    </form>
</div>
