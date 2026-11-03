import * as components from './core-components';

export const installer = {
  install(app) {
    Object.keys(components).forEach((key) => {
      app.component(components[key].name, components[key]);
    });
  }
};
