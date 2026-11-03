<template>
  <div :title="item.name" class="ele-upload-item" @click="handleItemClick">
    <slot name="thumbnail" :item="item" :imageUrl="imageUrl">
      <div
        v-if="listType === 'file' || item.thumbnail === false || !imageUrl"
        class="ele-upload-thumbnail"
      >
        <ElIcon class="ele-upload-thumbnail-icon">
          <FileOutlined />
        </ElIcon>
        <div class="ele-upload-thumbnail-text">{{ item.name || item.url }}</div>
      </div>
      <ElImage
        v-else
        fit="contain"
        class="ele-upload-image"
        v-bind="imageProps || {}"
        :src="imageUrl"
      />
    </slot>
    <slot name="itemExtra" :item="item"></slot>
    <div
      v-if="item.status === 'uploading' || item.status === 'exception'"
      class="ele-upload-progress"
    >
      <slot name="progress" :item="item">
        <div class="ele-upload-text">
          {{ item.status == 'exception' ? locale.exception : locale.uploading }}
        </div>
        <ElProgress
          :showText="false"
          :strokeWidth="listType === 'file' ? 2 : void 0"
          :status="item.status === 'exception' ? 'exception' : void 0"
          class="ele-upload-progress-bar"
          v-bind="progressProps || {}"
          :percentage="uploadProgress"
        />
        <div
          v-if="!disabled && item.status === 'exception'"
          :title="locale.retry"
          class="ele-upload-tool ele-upload-retry"
          @click.stop="handleRetry"
        >
          <ElIcon class="ele-upload-tool-icon">
            <SyncOutlined />
          </ElIcon>
          <div class="ele-upload-tool-text">{{ locale.retry }}</div>
        </div>
      </slot>
    </div>
    <div
      v-else-if="tools && !readonly && !disabled && !item.readonly"
      class="ele-upload-tools"
    >
      <div
        :title="locale.preview"
        class="ele-upload-tool"
        @click.stop="handlePreview"
      >
        <ElIcon class="ele-upload-tool-icon">
          <EyeOutlined />
        </ElIcon>
        <div class="ele-upload-tool-text">{{ locale.preview }}</div>
      </div>
      <div
        :title="locale.edit"
        class="ele-upload-tool"
        @click.stop="handleEdit"
      >
        <ElIcon class="ele-upload-tool-icon">
          <EditOutlined />
        </ElIcon>
        <div class="ele-upload-tool-text">{{ locale.edit }}</div>
      </div>
    </div>
    <div
      v-if="!readonly && !disabled && !item.readonly"
      :title="locale.remove"
      class="ele-upload-tool ele-upload-remove"
      @click.stop="handleRemove"
    >
      <slot name="remove" :item="item">
        <ElIcon class="ele-upload-tool-icon">
          <CloseOutlined />
        </ElIcon>
      </slot>
    </div>
    <div v-if="sortable" class="ele-upload-handle"></div>
  </div>
</template>

<script setup>
  import { computed } from 'vue';
  import { ElImage, ElProgress, ElIcon } from 'element-plus';
  import {
    CloseOutlined,
    EyeOutlined,
    EditOutlined,
    SyncOutlined,
    FileOutlined
  } from '../../icons/index';

  defineOptions({ name: 'ListItem' });

  const props = defineProps({
    /** 数据 */
    item: {
      type: Object,
      required: true
    },
    /** 是否只读 */
    readonly: Boolean,
    /** 是否禁用 */
    disabled: Boolean,
    /** 是否可拖拽排序 */
    sortable: Boolean,
    /** 图片属性 */
    imageProps: Object,
    /** 进度条属性 */
    progressProps: Object,
    /** 预读图片文件 */
    imageObjectUrls: Array,
    /** 是否开启底部预览和修改的操作按钮 */
    tools: Boolean,
    /** 列表样式 */
    listType: String,
    /** 国际化 */
    locale: {
      type: Object,
      required: true
    }
  });

  const emit = defineEmits({
    itemClick: (_item) => true,
    remove: (_item) => true,
    edit: (_item) => true,
    preview: (_item) => true,
    retry: (_item) => true
  });

  /** 图片预览地址 */
  const imageUrl = computed(() => {
    if (props.item.thumbnail && props.item.thumbnail !== true) {
      return props.item.thumbnail;
    }
    if (props.item.url) {
      return props.item.url;
    }
    if (!props.item.file || !props.imageObjectUrls) {
      return;
    }
    return props.imageObjectUrls.find((d) => d.file === props.item.file)?.url;
  });

  /** 上传进度值 */
  const uploadProgress = computed(() => {
    const progress = props.item.progress;
    if (!progress || progress < 0) {
      return 0;
    }
    return progress > 100 ? 100 : progress;
  });

  /** 点击事件 */
  const handleItemClick = () => {
    emit('itemClick', props.item);
  };

  /** 删除事件 */
  const handleRemove = () => {
    emit('remove', props.item);
  };

  /** 修改事件 */
  const handleEdit = () => {
    emit('edit', props.item);
  };

  /** 预览事件 */
  const handlePreview = () => {
    emit('preview', props.item);
  };

  /** 重试事件 */
  const handleRetry = () => {
    emit('retry', props.item);
  };
</script>
