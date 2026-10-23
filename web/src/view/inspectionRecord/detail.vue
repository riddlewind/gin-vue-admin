<template>
  <el-dialog
    :model-value="modelValue"
    title="巡检记录详情"
    width="840px"
    :close-on-click-modal="false"
    @update:model-value="(val) => emit('update:modelValue', val)"
  >
    <el-descriptions :column="2" border>
      <el-descriptions-item label="设备名称">
        {{ record?.deviceName || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="设备编号">
        {{ record?.deviceCode || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="设备位置">
        {{ record?.deviceLocation || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="巡检人">
        {{ record?.inspector || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="巡检时间">
        {{ formatInspectionTime(record?.inspectionTime) }}
      </el-descriptions-item>
      <el-descriptions-item label="巡检类型">
        {{ typeLabel(record?.inspectionType) }}
      </el-descriptions-item>
      <el-descriptions-item label="巡检结果">
        <el-tag :type="resultTagType(record?.inspectionResult)" effect="light">
          {{ resultLabel(record?.inspectionResult) }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="备注">
        {{ record?.remark || '-' }}
      </el-descriptions-item>
      <el-descriptions-item label="问题描述" :span="2">
        <div class="pre-wrap">{{ record?.problemDesc || '-' }}</div>
      </el-descriptions-item>
      <el-descriptions-item label="处理建议" :span="2">
        <div class="pre-wrap">{{ record?.handlingSuggestion || '-' }}</div>
      </el-descriptions-item>
    </el-descriptions>

    <div class="photo-section">
      <div class="section-title">现场照片</div>
      <div v-if="photoList.length" class="photo-list">
        <div
          v-for="(url, index) in photoList"
          :key="url + index"
          class="photo-item"
        >
          <el-image
            :src="url"
            :preview-src-list="photoList"
            :initial-index="index"
            preview-teleported
            fit="cover"
            class="photo-img"
          />
          <a
            :href="url"
            :download="fileName(url)"
            target="_blank"
            class="photo-link"
            >下载</a
          >
        </div>
      </div>
      <div v-else class="photo-empty">暂无照片</div>
    </div>

    <template #footer>
      <el-button @click="emit('update:modelValue', false)">返回</el-button>
      <el-button type="primary" @click="emit('edit')">编辑</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
  import { computed } from 'vue'
  import { formatInspectionTime, parsePhotos } from './constants'
  import { useInspectionDict } from './useInspectionDict'

  defineOptions({
    name: 'InspectionRecordDetail'
  })

  // 中文映射与标签颜色来自后台字典，加载失败会自动回退静态枚举
  const { typeLabel, resultLabel, resultTagType } = useInspectionDict()

  const props = defineProps({
    modelValue: {
      type: Boolean,
      default: false
    },
    record: {
      type: Object,
      default: null
    }
  })

  const emit = defineEmits(['update:modelValue', 'edit'])

  // photos 是 JSON 字符串，parsePhotos 内部已做好容错与 URL 前缀拼接
  const photoList = computed(() => parsePhotos(props.record?.photos))

  const fileName = (url) => decodeURIComponent(url.split('/').pop() || 'photo')
</script>

<style lang="scss" scoped>
  .pre-wrap {
    white-space: pre-wrap;
    word-break: break-word;
    line-height: 1.6;
  }

  .photo-section {
    margin-top: 16px;
  }

  .section-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    margin-bottom: 10px;
  }

  .photo-list {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
  }

  .photo-item {
    width: 200px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
  }

  .photo-img {
    width: 200px;
    height: 200px;
    border-radius: 6px;
    border: 1px solid var(--el-border-color-lighter);
    cursor: pointer;
  }

  .photo-link {
    font-size: 12px;
    color: var(--el-color-primary);
    text-decoration: none;
  }

  .photo-empty {
    padding: 24px 0;
    text-align: center;
    color: var(--el-text-color-placeholder);
    background: var(--el-fill-color-lighter);
    border-radius: 6px;
  }
</style>
