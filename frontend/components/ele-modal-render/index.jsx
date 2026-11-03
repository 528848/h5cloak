/** 弹窗渲染器 */
import { defineComponent } from 'vue';
import ItemRender from './components/item-render.vue';

export default defineComponent({
  name: 'EleModalRender',
  props: {
    modals: Array
  },
  emits: {
    removeItem: (_modalId) => true,
    updateItemVisible: (_modalId, _visible) => true,
    updateItemProps: (_modalId, _opt) => true
  },
  setup(props, { emit }) {
    const handleRemoveItem = (modalId) => {
      emit('removeItem', modalId);
    };

    const handleUpdateItemVisible = (modalId, visible) => {
      emit('updateItemVisible', modalId, visible);
    };

    const handleUpdateItemProps = (modalId, opt) => {
      emit('updateItemProps', modalId, opt);
    };

    const renderModal = (item) => {
      return (
        <ItemRender
          key={item.modalId}
          item={item}
          onRemoveItem={handleRemoveItem}
          onUpdateItemVisible={handleUpdateItemVisible}
          onUpdateItemProps={handleUpdateItemProps}
        >
          {{ ...(item.slots || {}) }}
        </ItemRender>
      );
    };
    return () => (props.modals || []).map((item) => renderModal(item));
  }
});
