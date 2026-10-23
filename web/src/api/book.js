import service from '@/utils/request'

/**
 * 图书模块接口封装
 *
 * [约定] 所有请求都必须走 @/utils/request，不要直接用 axios。
 * 因为 request.js 里已经处理了：
 *   - 自动注入 x-token / x-user-id 请求头
 *   - 统一 loading 与 401 自动跳登录
 *   - 把响应体里的 code !== 0 自动弹错误提示
 * 自己引 axios 会把这套机制全部绕过去。
 *
 * [注意] url 不用写 /api 前缀。.env.development 里 VITE_BASE_API=/api
 * 已经由 request.js 作为 baseURL 注入，Vite 再把它代理到 127.0.0.1:8888。
 */

// 新增图书
export const createBook = (data) => {
  return service({
    url: '/demo/book',
    method: 'post',
    data
  })
}

// 更新图书
export const updateBook = (data) => {
  return service({
    url: '/demo/book',
    method: 'put',
    data
  })
}

// 删除图书（DELETE 也带 body，和后端 ShouldBindJSON 对应）
export const deleteBook = (data) => {
  return service({
    url: '/demo/book',
    method: 'delete',
    data
  })
}

// 按 ID 查单条（GET 用 params，会被拼成 query string）
export const getBook = (params) => {
  return service({
    url: '/demo/book',
    method: 'get',
    params
  })
}

// 分页列表
export const getBookList = (params) => {
  return service({
    url: '/demo/bookList',
    method: 'get',
    params
  })
}
