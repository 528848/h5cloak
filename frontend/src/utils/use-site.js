import { ref, watch } from 'vue';
import { listSiteOptions } from '@/api/merchant/site';
import { APP_PORTAL } from '@portal';

const KEY = `${APP_PORTAL}-current-site`;

/**
 * 商户端当前站点选择，页面间共享并记忆。
 */
export function useCurrentSite() {
  const sites = ref([]);
  const siteId = ref(Number(localStorage.getItem(KEY)) || null);
  const loaded = ref(false);

  const load = () =>
    listSiteOptions()
      .then((rows) => {
        sites.value = rows || [];
        const exists = sites.value.some((s) => s.siteId === siteId.value);
        if (!exists) {
          siteId.value = sites.value[0]?.siteId ?? null;
        }
      })
      .catch(() => {})
      .finally(() => (loaded.value = true));

  watch(siteId, (v) => {
    if (v) {
      localStorage.setItem(KEY, String(v));
    } else {
      localStorage.removeItem(KEY);
    }
  });

  return { sites, siteId, loaded, load };
}
