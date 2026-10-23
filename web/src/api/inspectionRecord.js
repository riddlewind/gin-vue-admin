import service from '@/utils/request'

/**
 * 巡检记录（对应后端 model/inspectionrecord.InspectionRecord）
 *
 * [注意] 本项目是纯 JavaScript（无 tsconfig / 无 typescript 依赖），
 * 所以这里用 JSDoc 而不是 TS interface —— 既保留了类型提示，
 * 又不需要把整个前端切到 TypeScript。
 *
 * [注意] 主键字段名是**大写 ID**（后端嵌入的 global.GVA_MODEL 里
 * `json:"ID"`），不是小写的 id。业务字段倒是驼峰小写（deviceName 等）。
 * 同理 CreatedAt / UpdatedAt 也是大写开头。
 *
 * @typedef {Object} InspectionRecordItem
 * @property {number} ID 主键
 * @property {string} deviceName 设备名称
 * @property {string} deviceCode 设备编号
 * @property {string} deviceLocation 设备位置
 * @property {string} inspector 巡检人
 * @property {string} inspectionTime 巡检时间（YYYY-MM-DD HH:mm:ss）
 * @property {string} inspectionType 巡检类型：daily/special/fault/maintenance
 * @property {string} inspectionResult 巡检结果：normal/abnormal/fault/pending
 * @property {string} problemDesc 问题描述
 * @property {string} photos 现场照片，JSON 数组字符串，如 ["/uploads/a.jpg"]
 * @property {string} handlingSuggestion 处理建议
 * @property {string} remark 备注
 * @property {string} createdAt
 * @property {string} updatedAt
 */

/**
 * 列表查询参数
 *
 * @typedef {Object} InspectionRecordSearchParams
 * @property {string} [deviceName]
 * @property {string} [inspector]
 * @property {string} [inspectionType]
 * @property {string} [inspectionResult]
 * @property {string} [inspectionTimeStart] 闭区间左端，YYYY-MM-DD
 * @property {string} [inspectionTimeEnd] 闭区间右端，YYYY-MM-DD
 * @property {number} [page] 从 1 开始
 * @property {number} [pageSize]
 */

/**
 * 创建巡检记录
 * @param {Partial<InspectionRecordItem>} data
 * @returns {Promise<any>}
 */
export const createInspectionRecord = (data) => {
  return service.post('/inspectionRecord/createInspectionRecord', data)
}

/**
 * 删除巡检记录
 *
 * [注意] 参数走请求体 { id }，不是 query。
 * 后端 DeleteInspectionRecord 会先 ShouldBindJSON，绑定失败再回退读 query (?ID / ?id)。
 *
 * @param {{ id: number }} data
 * @returns {Promise<any>}
 */
export const deleteInspectionRecord = (data) => {
  return service.delete('/inspectionRecord/deleteInspectionRecord', { data })
}

/**
 * 获取巡检记录详情
 * @param {number} id
 * @returns {Promise<any>}
 */
export const getInspectionRecord = (id) => {
  return service.get('/inspectionRecord/getInspectionRecord', {
    params: { id }
  })
}

/**
 * 获取巡检记录列表（GET + query 参数）
 * @param {InspectionRecordSearchParams} data
 * @returns {Promise<any>}
 */
export const getInspectionRecordList = (data) => {
  return service.get('/inspectionRecord/getInspectionRecordList', {
    params: data
  })
}

/**
 * 更新巡检记录
 *
 * [注意] 必须带上 ID，后端会用 Select(白名单).Updates() 按主键局部更新；
 * 不带 ID 会被后端挡下（"巡检记录ID不能为空"）。
 *
 * @param {Partial<InspectionRecordItem>} data
 * @returns {Promise<any>}
 */
export const updateInspectionRecord = (data) => {
  return service.put('/inspectionRecord/updateInspectionRecord', data)
}
