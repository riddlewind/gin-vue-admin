<template>
  <div>
    <!-- 搜索区：GVA 约定用 gva-search-box 这个类名，样式由全局 css 提供 -->
    <div class="gva-search-box">
      <el-form :inline="true" :model="search">
        <el-form-item label="书名">
          <el-input
            v-model="search.title"
            placeholder="支持模糊搜索"
            clearable
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="handleSearch">
            查询
          </el-button>
          <el-button icon="refresh" @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button type="primary" icon="plus" @click="openDrawer('create')">
          新增图书
        </el-button>
      </div>

      <el-table :data="tableData" row-key="ID" style="width: 100%">
        <el-table-column label="ID" prop="ID" width="80" />
        <el-table-column label="书名" prop="title" min-width="180" />
        <el-table-column label="作者" prop="author" min-width="120" />
        <el-table-column label="价格" width="110">
          <template #default="scope">
            <span>￥{{ Number(scope.row.price || 0).toFixed(2) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="库存" prop="stock" width="90" />
        <el-table-column label="状态" width="100">
          <template #default="scope">
            <el-tag :type="scope.row.status === 1 ? 'success' : 'info'">
              {{ scope.row.status === 1 ? '上架' : '下架' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="scope">
            <span>{{ formatDate(scope.row.CreatedAt) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="scope">
            <el-button link type="primary" icon="edit" @click="openDrawer('update', scope.row)">
              编辑
            </el-button>
            <el-button link type="primary" icon="delete" @click="handleDelete(scope.row)">
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="gva-pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :page-sizes="[10, 30, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="getTableData"
          @size-change="getTableData"
        />
      </div>
    </div>

    <!-- 新增/编辑共用同一个抽屉，靠 type 区分 -->
    <el-drawer v-model="drawerVisible" size="500px" :before-close="closeDrawer">
      <template #header>
        <div class="flex justify-between items-center">
          <span class="text-lg">{{ type === 'create' ? '新增图书' : '编辑图书' }}</span>
        </div>
      </template>

      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item label="书名" prop="title">
          <el-input v-model="form.title" placeholder="请输入书名" />
        </el-form-item>
        <el-form-item label="作者" prop="author">
          <el-input v-model="form.author" placeholder="请输入作者" />
        </el-form-item>
        <el-form-item label="价格" prop="price">
          <el-input-number v-model="form.price" :min="0" :precision="2" :step="1" />
        </el-form-item>
        <el-form-item label="库存" prop="stock">
          <el-input-number v-model="form.stock" :min="0" :step="1" />
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-switch
            v-model="form.status"
            :active-value="1"
            :inactive-value="0"
            active-text="上架"
            inactive-text="下架"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <el-button @click="closeDrawer">取 消</el-button>
          <el-button type="primary" :loading="submitting" @click="handleSubmit">
            确 定
          </el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup>
  import { ref, reactive } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { formatDate } from '@/utils/format'
  import {
    createBook,
    updateBook,
    deleteBook,
    getBookList
  } from '@/api/book'

  // [重要] name 必须唯一，且建议和菜单里配置的路由 name 保持一致。
  // keepAlive（页面缓存）就是靠这个 name 匹配的，名字重复会导致缓存错乱。
  defineOptions({
    name: 'DemoBook'
  })

  // ---------- 列表 ----------
  const search = reactive({ title: '' })
  const page = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const tableData = ref([])

  const getTableData = async () => {
    // [注意] 后端返回结构是 { code, data: { list, total, page, pageSize }, msg }
    // request.js 已经把最外层拆掉了，所以这里拿到的 table 直接就是 body。
    const table = await getBookList({
      page: page.value,
      pageSize: pageSize.value,
      title: search.title
    })
    if (table.code === 0) {
      tableData.value = table.data.list || []
      total.value = table.data.total
    }
  }

  const handleSearch = () => {
    page.value = 1 // 搜索必须回到第一页，否则可能停在一个空页上
    getTableData()
  }

  const handleReset = () => {
    search.title = ''
    handleSearch()
  }

  // ---------- 新增 / 编辑 ----------
  const drawerVisible = ref(false)
  const submitting = ref(false)
  const type = ref('create')
  const formRef = ref(null)
  const form = reactive({
    ID: 0,
    title: '',
    author: '',
    price: 0,
    stock: 0,
    status: 1
  })

  // 前端校验只是"体验层"，后端 utils.Verify 才是真正的防线
  const rules = {
    title: [{ required: true, message: '请输入书名', trigger: 'blur' }],
    author: [{ required: true, message: '请输入作者', trigger: 'blur' }]
  }

  const openDrawer = (t, row) => {
    type.value = t
    if (t === 'update' && row) {
      Object.assign(form, row)
    } else {
      Object.assign(form, { ID: 0, title: '', author: '', price: 0, stock: 0, status: 1 })
    }
    drawerVisible.value = true
  }

  const closeDrawer = () => {
    drawerVisible.value = false
    formRef.value?.resetFields()
  }

  const handleSubmit = async () => {
    await formRef.value.validate()
    submitting.value = true
    try {
      const res =
        type.value === 'create' ? await createBook(form) : await updateBook(form)
      if (res.code === 0) {
        ElMessage.success(type.value === 'create' ? '新增成功' : '更新成功')
        closeDrawer()
        getTableData()
      }
    } finally {
      submitting.value = false
    }
  }

  // ---------- 删除 ----------
  const handleDelete = (row) => {
    ElMessageBox.confirm(`确定删除《${row.title}》吗？`, '提示', {
      type: 'warning',
      confirmButtonText: '确定',
      cancelButtonText: '取消'
    }).then(async () => {
      const res = await deleteBook({ id: row.ID })
      if (res.code === 0) {
        ElMessage.success('删除成功')
        // 删掉当前页最后一条时自动回退一页，避免停在空页
        if (tableData.value.length === 1 && page.value > 1) {
          page.value--
        }
        getTableData()
      }
    })
  }

  // 首次进入直接查一次
  getTableData()
</script>

<style scoped></style>
