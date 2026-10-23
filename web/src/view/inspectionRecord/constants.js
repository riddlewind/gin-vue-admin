// 巡检记录模块的公共常量与工具函数。
//
// 为什么单独放一个文件：index / form / detail 三个页面都要用到
// "类型枚举、结果枚举、结果标签颜色、时间格式化、photos 解析"，
// 抄三份的话改一个枚举要改三处，迟早漏。
//
// [注意] 枚举的**权威来源**已经换成后台「字典管理」
// （字典类型 inspection_type / inspection_result），页面统一通过
// useInspectionDict.js 取数。本文件里的枚举只作为兜底：
// 接口失败、或字典还没配的时候顶上，保证下拉框不会空。
// 所以要改选项文案/顺序，请去后台字典改，改这里只在兜底场景生效。
import { getUrl } from '@/utils/image'

// 巡检类型（存英文码，显示中文）
export const INSPECTION_TYPE_OPTIONS = [
  { value: 'daily', label: '日常巡检' },
  { value: 'special', label: '专项检查' },
  { value: 'fault', label: '故障排查' },
  { value: 'maintenance', label: '维护后巡检' }
]

// 巡检结果
export const INSPECTION_RESULT_OPTIONS = [
  { value: 'normal', label: '正常' },
  { value: 'abnormal', label: '异常' },
  { value: 'fault', label: '故障' },
  { value: 'pending', label: '待处理' }
]

// 结果 -> Element Plus 标签色
// [注意] Element Plus 没有 "green" 这个 type，绿色对应 success。
// normal-green / abnormal-warning / fault-danger / pending-info 的映射就落在这里。
export const INSPECTION_RESULT_TAG_TYPE = {
  normal: 'success',
  abnormal: 'warning',
  fault: 'danger',
  pending: 'info'
}

const toLabelMap = (options) =>
  options.reduce((acc, cur) => {
    acc[cur.value] = cur.label
    return acc
  }, {})

export const INSPECTION_TYPE_MAP = toLabelMap(INSPECTION_TYPE_OPTIONS)

export const INSPECTION_RESULT_MAP = toLabelMap(INSPECTION_RESULT_OPTIONS)

// 英文码 -> 中文，取不到就原样返回（兼容后端以后新增的枚举值）
export const typeLabel = (value) => INSPECTION_TYPE_MAP[value] || value || '-'

export const resultLabel = (value) =>
  INSPECTION_RESULT_MAP[value] || value || '-'

// 结果 -> el-tag 的 type，未知值兜底 info
export const resultTagType = (value) =>
  INSPECTION_RESULT_TAG_TYPE[value] || 'info'

/**
 * 巡检时间格式化：YYYY-MM-DD HH:mm
 *
 * 后端 time.Time 序列化成 RFC3339（2026-10-23T10:30:00+08:00），
 * 直接展示会带个 T 和时区，这里截断到分钟。
 *
 * @param {string} value
 * @returns {string}
 */
export const formatInspectionTime = (value) => {
  if (!value) return '-'
  const str = String(value).replace('T', ' ')
  return str.length >= 16 ? str.slice(0, 16) : str
}

/**
 * photos 字段（JSON 字符串）-> 可直接用于 <img src> 的 URL 数组
 *
 * [注意] 库里存的是相对路径（/uploads/file/xxx.jpg），必须拼上 VITE_FILE_API
 * 才能被浏览器访问；这个拼接由 utils/image 的 getUrl 负责。
 *
 * @param {string|string[]} photos
 * @returns {string[]}
 */
export const parsePhotos = (photos) => {
  if (!photos) return []
  let list = []
  if (Array.isArray(photos)) {
    list = photos
  } else {
    try {
      const parsed = JSON.parse(photos)
      list = Array.isArray(parsed) ? parsed : []
    } catch (e) {
      // 容错：历史脏数据可能不是合法 JSON（比如逗号分隔的裸字符串），
      // 解析失败时按逗号切一刀，实在没有就返回空数组，绝不抛异常。
      list = String(photos)
        .split(',')
        .map((item) => item.trim())
        .filter(Boolean)
    }
  }
  return list.filter(Boolean).map((url) => getUrl(url))
}
