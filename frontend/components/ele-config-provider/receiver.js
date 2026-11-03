import { inject, computed } from 'vue';
import defaultLocale from '../lang/zh_CN';
import { modalItemContextKey as injectContext } from '../utils/hook';
export { injectContext };

/**
 * 全局配置键名
 */
export const CONFIG_KEY = Symbol('config');

/**
 * 获取全局配置
 */
export function useReceiver() {
  return inject(CONFIG_KEY, {});
}

/**
 * 获取全局属性
 */
export function useGlobalProps(name) {
  const globalConfig = useReceiver();
  return computed(() => globalConfig[name] ?? {});
}

/**
 * 获取国际化
 */
export function useLocale(name, props) {
  const globalConfig = useReceiver();
  const lang = computed(() => {
    const temp = globalConfig.locale ?? defaultLocale;
    try {
      if (name) {
        return Object.assign({}, temp[name] ?? {}, props?.locale);
      }
    } catch (e) {
      console.error(injectContext, e);
    }
    return temp;
  });
  return { lang, globalConfig };
}
export const configValues = '';
