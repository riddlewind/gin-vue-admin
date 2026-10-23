// 巡检记录模块 · 枚举字典取数
//
// 枚举已交给后台「字典管理」维护（inspection_type / inspection_result），
// 这个模块负责三件事：
// 1. 模块级缓存：index / form / detail 共用一份，来回切页面不会重复打接口；
// 2. 兜底：接口失败或字典没配（返回空）时退回 constants.js 里的静态枚举，
//    保证下拉框永远不会变成空的、页面不会因为字典服务抽风而不可用；
// 3. 历史数据兜底：labelMap 保留"已停用"的项，老记录仍能显示中文，
//    只是这些值不会再出现在新增/编辑的下拉里。
import { ref } from 'vue'
import { getDictionaryTreeListByType } from '@/api/sysDictionaryDetail'
import {
  INSPECTION_TYPE_OPTIONS,
  INSPECTION_RESULT_OPTIONS,
  INSPECTION_TYPE_MAP,
  INSPECTION_RESULT_MAP,
  INSPECTION_RESULT_TAG_TYPE
} from './constants'

const TYPE_DICT = 'inspection_type'
const RESULT_DICT = 'inspection_result'

// el-tag 合法的 type。后台在 extend 里填了别的值就忽略，避免把组件渲染坏。
const VALID_TAG_TYPES = ['primary', 'success', 'warning', 'danger', 'info']

/**
 * 后端字典树 -> 下拉选项
 *
 * 只保留启用的项：后端把 status=false 映射成 disabled=true。
 *
 * @param {Array} list 后端返回的字典明细数组
 * @returns {{value: string, label: string}[]}
 */
export const buildOptions = (list) =>
  (list || [])
    .filter((item) => item && item.disabled !== true)
    .map((item) => ({ value: String(item.value), label: item.label }))

/**
 * 后端字典树 -> { 码值: 中文 } 映射
 *
 * 不做启用过滤：历史记录里可能存着已停用的枚举，也要能显示中文。
 *
 * @param {Array} list
 * @returns {Record<string, string>}
 */
export const buildLabelMap = (list) =>
  (list || []).reduce((acc, item) => {
    if (item && item.value !== undefined && item.value !== null) {
      acc[String(item.value)] = item.label
    }
    return acc
  }, {})

/**
 * 后端字典树 -> { 码值: el-tag type } 映射
 *
 * 颜色写在字典明细的 extend 列里（success / warning / danger / info），
 * 这样运营在后台改颜色不用动代码。
 *
 * @param {Array} list
 * @returns {Record<string, string>}
 */
export const buildTagMap = (list) =>
  (list || []).reduce((acc, item) => {
    const ext = String(item?.extend || '').trim()
    if (item?.value !== undefined && VALID_TAG_TYPES.includes(ext)) {
      acc[String(item.value)] = ext
    }
    return acc
  }, {})

// 先用静态枚举占位，接口回来再覆盖：页面首屏就有内容，不会闪一下空下拉
const state = {
  typeOptions: ref([...INSPECTION_TYPE_OPTIONS]),
  resultOptions: ref([...INSPECTION_RESULT_OPTIONS]),
  typeLabels: ref({}),
  resultLabels: ref({}),
  resultTags: ref({}),
  // 'dict' = 用的是后台字典，'fallback' = 用的是内置静态枚举
  source: ref('fallback'),
  loading: ref(false)
}

let loaded = false
let pending = null

const fetchOne = async (type) => {
  const res = await getDictionaryTreeListByType({ type })
  if (res.code !== 0) {
    throw new Error(res.msg || `字典 ${type} 加载失败`)
  }
  return res.data?.list || []
}

/**
 * 加载（或强制重载）巡检相关字典
 *
 * 不会 reject：失败时自动回退静态枚举，只在控制台留一条 warn。
 * @param {boolean} force 传 true 可忽略缓存重新拉取（后台改完字典想立刻生效时用）
 * @returns {Promise<typeof state>}
 */
export const loadInspectionDict = (force = false) => {
  if (pending && !force) return pending
  if (loaded && !force) return Promise.resolve(state)

  state.loading.value = true
  pending = Promise.all([fetchOne(TYPE_DICT), fetchOne(RESULT_DICT)])
    .then(([typeList, resultList]) => {
      // 返回空数组视为"字典没配"，继续用兜底枚举，避免下拉框空掉
      if (typeList.length) {
        state.typeOptions.value = buildOptions(typeList)
        state.typeLabels.value = buildLabelMap(typeList)
      }
      if (resultList.length) {
        state.resultOptions.value = buildOptions(resultList)
        state.resultLabels.value = buildLabelMap(resultList)
        state.resultTags.value = buildTagMap(resultList)
      }
      state.source.value =
        typeList.length || resultList.length ? 'dict' : 'fallback'
      return state
    })
    .catch((err) => {
      console.warn(
        '[inspectionRecord] 字典加载失败，已回退到内置枚举：',
        err?.message || err
      )
      state.source.value = 'fallback'
      return state
    })
    .finally(() => {
      loaded = true
      state.loading.value = false
      pending = null
    })

  return pending
}

/**
 * 页面里用这个拿枚举、中文映射与颜色
 *
 * 首次调用会自动触发加载（模块级单例，多页面共用一次请求）。
 */
export const useInspectionDict = () => {
  loadInspectionDict()

  // 字典里有就用字典，没有就退静态映射，再没有就原样显示码值
  const typeLabel = (value) =>
    state.typeLabels.value[value] || INSPECTION_TYPE_MAP[value] || value || '-'

  const resultLabel = (value) =>
    state.resultLabels.value[value] ||
    INSPECTION_RESULT_MAP[value] ||
    value ||
    '-'

  const resultTagType = (value) =>
    state.resultTags.value[value] || INSPECTION_RESULT_TAG_TYPE[value] || 'info'

  return {
    typeOptions: state.typeOptions,
    resultOptions: state.resultOptions,
    typeLabel,
    resultLabel,
    resultTagType,
    dictSource: state.source,
    dictLoading: state.loading,
    reloadDict: () => loadInspectionDict(true)
  }
}
