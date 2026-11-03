import { ref, nextTick, provide, inject, shallowRef, unref } from 'vue';

/**
 * 气泡确认框注入键名
 */
export const POPCONFIRM_KEY = Symbol('popconfirm');

/**
 * 提供气泡确认框操作给子组件
 */
export function usePopconfirmProvider() {
  const popconfirmRef = ref(null);

  const popconfirmVirtualRef = shallowRef();

  const popconfirmProps = shallowRef({});

  const openPopconfirm = (triggerEl, props) => {
    if (triggerEl == null || popconfirmVirtualRef.value === triggerEl) {
      return;
    }
    popconfirmRef.value && popconfirmRef.value.hidePopper();
    nextTick(() => {
      popconfirmProps.value = props || {};
      popconfirmVirtualRef.value = triggerEl;
      nextTick(() => {
        if (popconfirmRef.value) {
          unref(popconfirmRef.value.tooltipRef)?.handleOpen?.();
        }
      });
    });
  };

  provide(POPCONFIRM_KEY, {
    openPopconfirm
  });

  return {
    popconfirmRef,
    popconfirmVirtualRef,
    popconfirmProps,
    openPopconfirm
  };
}

/**
 * 气泡确认框操作
 */
export function usePopconfirm() {
  return inject(POPCONFIRM_KEY, {});
}
