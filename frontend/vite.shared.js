import { defineConfig, loadEnv } from 'vite';
import vue from '@vitejs/plugin-vue';
import vueJsx from '@vitejs/plugin-vue-jsx';
import { resolve } from 'node:path';
import Compression from 'vite-plugin-compression';
import Components from 'unplugin-vue-components/vite';
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers';
import { EleAdminResolver } from './components/utils/resolvers';
import { ComponentsResolver } from './src/components/resolvers';

const portals = {
  admin: {
    port: 5173,
    outDir: 'dist/admin',
    title: 'H5Cloak 总后台'
  },
  merchant: {
    port: 5174,
    outDir: 'dist/merchant',
    title: 'H5Cloak 商户端'
  }
};

export function createPortalViteConfig(portal) {
  const portalConfig = portals[portal];
  if (!portalConfig) {
    throw new Error(`Unknown frontend portal: ${portal}`);
  }

  return defineConfig(({ command, mode }) => {
    const isBuild = command === 'build';
    const env = loadEnv(mode, process.cwd());
    const alias = [
      { find: '@portal', replacement: resolve(`src/portals/${portal}.js`) },
      { find: '@/', replacement: resolve('src') + '/' },
      { find: 'vue-i18n', replacement: 'vue-i18n/dist/vue-i18n.cjs.js' },
      { find: 'ele-admin-plus/es', replacement: resolve('components') },
      { find: 'ele-admin-plus', replacement: resolve('components') }
    ];
    const plugins = [vue(), vueJsx()];
    const componentsResolvers = [ComponentsResolver()];
    if (isBuild) {
      componentsResolvers.push(
        ElementPlusResolver({
          importStyle: 'sass'
        })
      );
      componentsResolvers.push(
        EleAdminResolver({
          importStyle: 'sass'
        })
      );
      plugins.push(
        Compression({
          disable: !isBuild,
          threshold: 10240,
          algorithm: 'gzip',
          ext: '.gz'
        })
      );
    } else {
      alias.push({ find: './as-needed', replacement: './global-import' });
    }
    plugins.push({
      name: 'portal-title',
      transformIndexHtml(html) {
        return html.replace(
          /<title>.*?<\/title>/,
          `<title>${portalConfig.title}</title>`
        );
      }
    });
    plugins.push(
      Components({
        dts: false,
        globsExclude: ['src/components/*/components/**'],
        resolvers: componentsResolvers
      })
    );
    const baseApi = env.VITE_API_URL;
    const proxyApi = env.VITE_API_PROXY_URL;
    const proxy = {};
    if (proxyApi) {
      const isRemoveBasePath = !proxyApi.endsWith(baseApi);
      proxy[baseApi] = {
        target: isRemoveBasePath
          ? proxyApi
          : proxyApi.substring(0, proxyApi.length - baseApi.length),
        rewrite: isRemoveBasePath
          ? (path) => path.replace(new RegExp(`^${baseApi}`), '')
          : void 0,
        ws: true,
        changeOrigin: true,
        configure: (proxyServer) => {
          proxyServer.on('proxyReq', (proxyReq) => {
            proxyReq.setHeader('Referer', proxyApi);
          });
        }
      };
    }
    return {
      cacheDir: `node_modules/.vite/${portal}`,
      resolve: { alias },
      plugins,
      css: {
        preprocessorOptions: {
          scss: {
            additionalData: `@use "@/styles/variables.scss" as *;`
          }
        }
      },
      optimizeDeps: {
        include: [
          'echarts/core',
          'echarts/charts',
          'echarts/renderers',
          'echarts/components',
          'vue-echarts',
          'echarts-wordcloud',
          'sortablejs',
          'vuedraggable'
        ]
      },
      build: {
        outDir: portalConfig.outDir,
        emptyOutDir: true,
        target: 'chrome63',
        chunkSizeWarningLimit: 4000,
        assetsInlineLimit: 0
      },
      server: {
        host: '127.0.0.1',
        port: portalConfig.port,
        strictPort: true,
        proxy
      },
      preview: {
        host: '127.0.0.1',
        port: portalConfig.port,
        strictPort: true
      }
    };
  });
}
