<template>
  <el-dialog
    :model-value="modelValue"
    :title="isEdit ? '编辑巡检记录' : '新增巡检记录'"
    width="760px"
    :close-on-click-modal="false"
    destroy-on-close
    @update:model-value="(val) => emit('update:modelValue', val)"
    @closed="resetForm"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="设备名称" prop="deviceName">
            <el-input
              v-model="form.deviceName"
              maxlength="100"
              show-word-limit
              placeholder="请输入设备名称"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="设备编号" prop="deviceCode">
            <el-input
              v-model="form.deviceCode"
              maxlength="50"
              placeholder="请输入设备编号"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="设备位置" prop="deviceLocation">
            <el-input
              v-model="form.deviceLocation"
              maxlength="200"
              placeholder="请输入设备位置"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="巡检人" prop="inspector">
            <el-input
              v-model="form.inspector"
              maxlength="100"
              placeholder="请输入巡检人"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="巡检时间" prop="inspectionTime">
            <el-date-picker
              v-model="form.inspectionTime"
              type="datetime"
              value-format="YYYY-MM-DD HH:mm:ss"
              placeholder="请选择巡检时间"
              style="width: 100%"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="巡检类型" prop="inspectionType">
            <el-select
              v-model="form.inspectionType"
              placeholder="请选择巡检类型"
              style="width: 100%"
            >
              <el-option
                v-for="item in typeOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="巡检结果" prop="inspectionResult">
            <el-select
              v-model="form.inspectionResult"
              placeholder="请选择巡检结果"
              style="width: 100%"
              @change="handleResultChange"
            >
              <el-option
                v-for="item in resultOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="问题描述" prop="problemDesc">
            <el-input
              v-model="form.problemDesc"
              type="textarea"
              :rows="4"
              maxlength="2000"
              show-word-limit
              placeholder="巡检结果不是「正常」时必填"
            />
          </el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="处理建议" prop="handlingSuggestion">
            <el-input
              v-model="form.handlingSuggestion"
              type="textarea"
              :rows="3"
              maxlength="1000"
              show-word-limit
              placeholder="请输入处理建议"
            />
          </el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="备注" prop="remark">
            <el-input
              v-model="form.remark"
              maxlength="500"
              show-word-limit
              placeholder="请输入备注"
            />
          </el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="现场照片">
            <div class="photo-uploader">
              <!--
                用自定义 http-request 而不是 action：
                action 方式没法在发送前做 canvas 压缩，也没法把返回的 url 存进 rawUrl。
              -->
              <el-upload
                v-model:file-list="fileList"
                :http-request="handleUpload"
                :before-upload="beforeUpload"
                :on-preview="handlePreview"
                :on-remove="handleRemove"
                :on-exceed="handleExceed"
                :limit="10"
                accept="image/*"
                list-type="picture-card"
                multiple
              >
                <el-icon><Plus /></el-icon>
              </el-upload>
              <div class="upload-tip">
                支持拍照和相册选取，单张不超过5MB，最多10张
              </div>
            </div>
          </el-form-item>
        </el-col>
      </el-row>
    </el-form>

    <template #footer>
      <el-button @click="emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="submitForm"
        >确定</el-button
      >
    </template>

    <!-- 照片预览：点 picture-card 上的「眼睛」图标触发 -->
    <el-image-viewer
      v-if="viewerVisible"
      :url-list="viewerList"
      :initial-index="viewerIndex"
      teleported
      @close="viewerVisible = false"
    />
  </el-dialog>
</template>

<script setup>
  import { ref, reactive, computed, watch, nextTick } from 'vue'
  import { ElMessage } from 'element-plus'
  import { Plus } from '@element-plus/icons-vue'
  import service from '@/utils/request'
  import { getUrl } from '@/utils/image'
  import { useUserStore } from '@/pinia'
  import {
    createInspectionRecord,
    updateInspectionRecord
  } from '@/api/inspectionRecord'
  import { useInspectionDict } from './useInspectionDict'

  defineOptions({
    name: 'InspectionRecordForm'
  })

  // 下拉项来自后台字典（inspection_type / inspection_result），加载失败会自动回退静态枚举
  const { typeOptions, resultOptions } = useInspectionDict()

  const props = defineProps({
    modelValue: {
      type: Boolean,
      default: false
    },
    // 传 null 表示新增，传行数据表示编辑
    record: {
      type: Object,
      default: null
    }
  })

  const emit = defineEmits(['update:modelValue', 'success'])

  const userStore = useUserStore()

  const MAX_PHOTO_SIZE = 5 * 1024 * 1024 // 5MB
  const MAX_PHOTO_COUNT = 10

  const formRef = ref(null)
  const submitting = ref(false)
  const fileList = ref([])

  const isEdit = computed(() => !!props.record?.ID)

  const form = reactive({
    deviceName: '',
    deviceCode: '',
    deviceLocation: '',
    inspector: '',
    inspectionTime: '',
    inspectionType: '',
    inspectionResult: '',
    problemDesc: '',
    handlingSuggestion: '',
    remark: ''
  })

  // 结果为「正常」时问题描述可以为空，其余都要填
  const validateProblemDesc = (rule, value, callback) => {
    const abnormal = form.inspectionResult && form.inspectionResult !== 'normal'
    if (abnormal && !String(value || '').trim()) {
      callback(new Error('巡检结果不是「正常」时，必须填写问题描述'))
      return
    }
    callback()
  }

  const rules = {
    deviceName: [
      { required: true, message: '请输入设备名称', trigger: 'blur' },
      { max: 100, message: '设备名称最多 100 个字符', trigger: 'blur' }
    ],
    deviceCode: [
      { max: 50, message: '设备编号最多 50 个字符', trigger: 'blur' }
    ],
    deviceLocation: [
      { max: 200, message: '设备位置最多 200 个字符', trigger: 'blur' }
    ],
    inspector: [{ required: true, message: '请输入巡检人', trigger: 'blur' }],
    inspectionTime: [
      { required: true, message: '请选择巡检时间', trigger: 'change' }
    ],
    inspectionType: [
      { required: true, message: '请选择巡检类型', trigger: 'change' }
    ],
    inspectionResult: [
      { required: true, message: '请选择巡检结果', trigger: 'change' }
    ],
    problemDesc: [
      { max: 2000, message: '问题描述最多 2000 个字符', trigger: 'blur' },
      { validator: validateProblemDesc, trigger: 'blur' }
    ],
    handlingSuggestion: [
      { max: 1000, message: '处理建议最多 1000 个字符', trigger: 'blur' }
    ],
    remark: [{ max: 500, message: '备注最多 500 个字符', trigger: 'blur' }]
  }

  // ---------------------------------------------------------------------------
  // 照片上传
  // ---------------------------------------------------------------------------

  const viewerVisible = ref(false)
  const viewerList = ref([])
  const viewerIndex = ref(0)

  /**
   * canvas 压缩：最大宽度 1920，质量 0.7
   *
   * [注意] 这里返回 Promise，http-request 里 await 它。
   * 不用 el-upload 的 before-upload 返回值替换文件，是因为替换后
   * 新 File 的 uid 和 fileList 里那一条对不上，后面就找不到要更新 url 的项了。
   */
  const compressImage = (file, maxWidth = 1920, quality = 0.7) => {
    return new Promise((resolve) => {
      if (!file.type?.startsWith('image/')) {
        resolve(file)
        return
      }
      const reader = new FileReader()
      reader.onerror = () => resolve(file)
      reader.onload = () => {
        const img = new Image()
        img.onerror = () => resolve(file)
        img.onload = () => {
          // 尺寸和体积都不大就不重新编码，避免白白损失画质
          if (img.width <= maxWidth && file.size <= 1024 * 1024) {
            resolve(file)
            return
          }
          const scale = Math.min(1, maxWidth / img.width)
          const canvas = document.createElement('canvas')
          canvas.width = Math.round(img.width * scale)
          canvas.height = Math.round(img.height * scale)
          const ctx = canvas.getContext('2d')
          ctx.drawImage(img, 0, 0, canvas.width, canvas.height)

          // PNG 保 PNG（怕丢透明通道），其余统一转 JPEG 才能吃到 quality
          const outputType =
            file.type === 'image/png' ? 'image/png' : 'image/jpeg'
          canvas.toBlob(
            (blob) => {
              if (!blob) {
                resolve(file)
                return
              }
              const ext = outputType === 'image/png' ? '.png' : '.jpg'
              const name = file.name.replace(/\.[^.]+$/, '') + ext
              resolve(
                new File([blob], name, {
                  type: outputType,
                  lastModified: Date.now()
                })
              )
            },
            outputType,
            quality
          )
        }
        img.src = reader.result
      }
      reader.readAsDataURL(file)
    })
  }

  const beforeUpload = (file) => {
    if (!file.type?.startsWith('image/')) {
      ElMessage.error('只能上传图片文件')
      return false
    }
    if (file.size > MAX_PHOTO_SIZE) {
      ElMessage.error('单张图片不能超过 5MB')
      return false
    }
    if (fileList.value.length >= MAX_PHOTO_COUNT) {
      ElMessage.warning(`最多只能上传 ${MAX_PHOTO_COUNT} 张照片`)
      return false
    }
    return true
  }

  /**
   * 自定义上传
   *
   * token 不用手动加：utils/request 的请求拦截器统一塞了 x-token，
   * 这里直接复用同一个 axios 实例就自动带上了。
   */
  const handleUpload = async (options) => {
    try {
      const file = await compressImage(options.file)
      const formData = new FormData()
      formData.append('file', file)
      // noSave=1：只落盘、不写进「媒体库」表，巡检照片只属于这条记录
      const res = await service.post(
        '/fileUploadAndDownload/upload?noSave=1',
        formData,
        { headers: { 'Content-Type': 'multipart/form-data' } }
      )
      const url = res?.data?.file?.url
      if (res.code !== 0 || !url) {
        throw new Error(res?.msg || '上传失败')
      }
      // rawUrl 存相对路径（最终入库的值），url 存拼好前缀的地址（给缩略图用）
      const item = fileList.value.find((f) => f.uid === options.file.uid)
      if (item) {
        item.rawUrl = url
        item.url = getUrl(url)
        item.status = 'success'
      }
      options.onSuccess(res)
    } catch (err) {
      // 失败的那条从列表里摘掉，否则会留一个红框占位
      fileList.value = fileList.value.filter((f) => f.uid !== options.file.uid)
      ElMessage.error(err?.message || '照片上传失败')
      options.onError(err)
    }
  }

  const handlePreview = (file) => {
    const urls = fileList.value.map((f) => f.url).filter(Boolean)
    viewerList.value = urls
    viewerIndex.value = Math.max(0, urls.indexOf(file.url))
    viewerVisible.value = true
  }

  const handleRemove = () => {
    // 列表由 v-model:file-list 维护，rawUrl 随条目一起被移除，无需额外清理
  }

  const handleExceed = () => {
    ElMessage.warning(`最多只能上传 ${MAX_PHOTO_COUNT} 张照片`)
  }

  // ---------------------------------------------------------------------------
  // 表单初始化 / 提交
  // ---------------------------------------------------------------------------

  const pad = (n) => String(n).padStart(2, '0')

  const nowString = () => {
    const d = new Date()
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(
      d.getDate()
    )} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  }

  const resetForm = () => {
    Object.assign(form, {
      deviceName: '',
      deviceCode: '',
      deviceLocation: '',
      inspector: '',
      inspectionTime: '',
      inspectionType: '',
      inspectionResult: '',
      problemDesc: '',
      handlingSuggestion: '',
      remark: ''
    })
    fileList.value = []
    formRef.value?.clearValidate()
  }

  const initForm = () => {
    resetForm()
    if (props.record?.ID) {
      // 编辑：把行数据摊进表单
      Object.keys(form).forEach((key) => {
        if (props.record[key] !== undefined && props.record[key] !== null) {
          form[key] = props.record[key]
        }
      })
      // photos 在库里是 JSON 字符串，要还原成 el-upload 的 fileList
      fileList.value = parsePhotoFileList(props.record.photos)
    } else {
      // 新增：巡检人默认当前登录用户昵称，时间默认现在
      form.inspector = userStore.userInfo?.nickName || ''
      form.inspectionTime = nowString()
    }
    nextTick(() => formRef.value?.clearValidate())
  }

  const parsePhotoFileList = (photos) => {
    if (!photos) return []
    let list = []
    try {
      const parsed = JSON.parse(photos)
      list = Array.isArray(parsed) ? parsed : []
    } catch (e) {
      // 脏数据兜底，宁可少显示几张照片，也别让弹窗打不开
      list = []
    }
    return list.filter(Boolean).map((url, index) => ({
      name: url.split('/').pop() || `照片${index + 1}`,
      // rawUrl 是原样入库的相对路径，提交时用它，不能用带前缀的 url
      rawUrl: url,
      url: getUrl(url),
      status: 'success',
      uid: Date.now() + index
    }))
  }

  const handleResultChange = () => {
    // 结果一变，问题描述的必填规则也跟着变，立刻重新校验一次
    formRef.value?.validateField('problemDesc')
  }

  const submitForm = async () => {
    if (!formRef.value) return
    const valid = await formRef.value.validate().catch(() => false)
    if (!valid) return

    submitting.value = true
    try {
      // photos 以数组形式存在 fileList 的 rawUrl 里，入库前序列化成 JSON 字符串
      const photos = fileList.value.map((f) => f.rawUrl).filter(Boolean)
      const payload = { ...form, photos: JSON.stringify(photos) }
      const res = isEdit.value
        ? await updateInspectionRecord({ ...payload, ID: props.record.ID })
        : await createInspectionRecord(payload)
      if (res.code !== 0) return
      ElMessage.success(isEdit.value ? '更新成功' : '创建成功')
      emit('update:modelValue', false)
      emit('success')
    } finally {
      submitting.value = false
    }
  }

  watch(
    () => props.modelValue,
    (val) => {
      if (val) initForm()
    }
  )
</script>

<style lang="scss" scoped>
  .photo-uploader {
    width: 100%;
  }

  .upload-tip {
    margin-top: 8px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1.5;
  }
</style>
