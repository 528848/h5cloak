import { ref, nextTick, provide, inject, shallowRef } from 'vue';

/**
 * 下拉菜单注入键名
 */
export const DROPDOWN_KEY = Symbol('dropdown');

/**
 * 提供下拉菜单操作给子组件
 */
export function useDropdownProvider() {
  const dropdownRef = ref(null);

  const dropdownVirtualRef = shallowRef();

  const dropdownItems = shallowRef([]);

  const dropdownProps = shallowRef({});

  const openDropdown = (triggerEl, items, props) => {
    if (triggerEl == null || dropdownVirtualRef.value === triggerEl) {
      return;
    }
    dropdownRef.value && dropdownRef.value.handleClose();
    nextTick(() => {
      dropdownProps.value = props || {};
      dropdownItems.value = items || [];
      dropdownVirtualRef.value = triggerEl;
      if (dropdownItems.value.length) {
        nextTick(() => {
          dropdownRef.value && dropdownRef.value.handleOpen();
        });
      }
    });
  };

  provide(DROPDOWN_KEY, {
    openDropdown
  });

  return {
    dropdownRef,
    dropdownVirtualRef,
    dropdownItems,
    dropdownProps,
    openDropdown
  };
}

/**
 * 下拉菜单操作
 */
export function useDropdown() {
  return inject(DROPDOWN_KEY, {});
}
