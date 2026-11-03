/** 全局安装 */
import * as components from './components';

export const installer = {
  install(app) {
    Object.keys(components).forEach((key) => {
      app.component(components[key].name, components[key]);
    });
  }
};
