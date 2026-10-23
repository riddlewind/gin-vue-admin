<template>
  <div>
    <!-- 搜索区 -->
    <div class="gva-search-box">
      <el-form :inline="true" :model="searchForm" class="search-form">
        <el-form-item label="设备名称">
          <el-input
            v-model="searchForm.deviceName"
            placeholder="请输入设备名称"
            clearable
            style="width: 180px"
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item label="巡检人">
          <el-input
            v-model="searchForm.inspector"
            placeholder="请输入巡检人"
            clearable
            style="width: 160px"
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item label="巡检类型">
          <el-select
            v-model="searchForm.inspectionType"
            placeholder="全部"
            clearable
            style="width: 150px"
          >
            <el-option
              v-for="item in typeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="巡检结果">
          <el-select
            v-model="searchForm.inspectionResult"
            placeholder="全部"
            clearable
            style="width: 140px"
          >
            <el-option
              v-for="item in resultOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="巡检时间">
          <el-date-picker
            v-model="searchForm.dateRange"
            type="daterange"
            value-format="YYYY-MM-DD"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            unlink-panels
            style="width: 260px"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="handleSearch"
            >查询</el-button
          >
          <el-button icon="refresh" @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 工具栏 -->
    <div class="gva-btn-list">
      <!-- 权限标识：inspectionRecord:create -->
      <el-button
        v-if="canCreate"
        type="primary"
        icon="plus"
        @click="handleCreate"
        >新增巡检记录</el-button
      >
    </div>

    <!-- 表格区 -->
    <div class="gva-table-box">
      <el-table
        v-loading="loading"
        :data="tableData"
        row-key="ID"
        style="width: 100%"
      >
        <el-table-column
          prop="deviceName"
          label="设备名称"
          min-width="140"
          show-overflow-tooltip
        />
        <el-table-column
          prop="deviceCode"
          label="设备编号"
          width="130"
          show-overflow-tooltip
        />
        <el-table-column prop="inspector" label="巡检人" width="110" />
        <el-table-column label="巡检时间" width="160">
          <template #default="{ row }">
            {{ formatInspectionTime(row.inspectionTime) }}
          </template>
        </el-table-column>
        <el-table-column label="巡检类型" width="120">
          <template #default="{ row }">
            {{ typeLabel(row.inspectionType) }}
          </template>
        </el-table-column>
        <el-table-column label="巡检结果" width="100">
          <template #default="{ row }">
            <el-tag :type="resultTagType(row.inspectionResult)" effect="light">
              {{ resultLabel(row.inspectionResult) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="现场照片" width="110">
          <template #default="{ row }">
            <!-- 缩略图只显示第一张，点开可以左右翻全部 -->
            <el-image
              v-if="parsePhotos(row.photos).length"
              :src="parsePhotos(row.photos)[0]"
              :preview-src-list="parsePhotos(row.photos)"
              :initial-index="0"
              preview-teleported
              fit="cover"
              class="thumb"
            />
            <span v-else class="muted">暂无</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="190" fixed="right">
          <template #default="{ row }">
            <!-- 权限标识：inspectionRecord:detail -->
            <el-button
              v-if="canDetail"
              type="primary"
              link
              icon="view"
              @click="handleDetail(row)"
              >详情</el-button
            >
            <!-- 权限标识：inspectionRecord:edit -->
            <el-button
              v-if="canEdit"
              type="primary"
              link
              icon="edit"
              @click="handleEdit(row)"
              >编辑</el-button
            >
            <!-- 权限标识：inspectionRecord:delete -->
            <el-button
              v-if="canDelete"
              type="danger"
              link
              icon="delete"
              @click="handleDelete(row)"
              >删除</el-button
            >
          </template>
        </el-table-column>
      </el-table>

      <div class="gva-pagination">
        <el-pagination
          :current-page="page"
          :page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handleCurrentChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <!-- 新增/编辑弹窗 -->
    <InspectionRecordForm
      v-model="formVisible"
      :record="currentRecord"
      @success="handleFormSuccess"
    />

    <!-- 详情弹窗 -->
    <InspectionRecordDetail
      v-model="detailVisible"
      :record="currentRecord"
      @edit="handleEditFromDetail"
    />
  </div>
</template>

<script setup>
  import { ref, reactive, computed, onMounted } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { useBtnAuth } from '@/utils/btnAuth'
  import {
    getInspectionRecordList,
    deleteInspectionRecord
  } from '@/api/inspectionRecord'
  import { formatInspectionTime, parsePhotos } from './constants'
  import { useInspectionDict } from './useInspectionDict'
  import InspectionRecordForm from './form.vue'
  import InspectionRecordDetail from './detail.vue'

  defineOptions({
    name: 'inspectionRecord'
  })

  // 下拉项与中文映射都来自后台字典（inspection_type / inspection_result），
  // 首次调用会自动加载，失败时内部已回退到 constants 里的静态枚举。
  const { typeOptions, resultOptions, typeLabel, resultLabel, resultTagType } =
    useInspectionDict()

  // 按钮级权限：数据来自 route.meta.btns，而 meta.btns 是后端
  // sys_authority_btns → sys_base_menu_btns.name 的映射（只有被授权的键才存在）。
  // 注意键名必须与菜单里配置的按钮名完全一致（inspectionRecord:create 这种），
  // 写成简写 create 会永远拿不到值。
  // 兼容策略：菜单里一个按钮都没配时（空对象）全部放行，避免老菜单下按钮全消失。
  const btnAuth = useBtnAuth()
  const hasBtn = (name) => {
    const keys = Object.keys(btnAuth)
    if (!keys.length) return true
    return !!btnAuth[name]
  }
  const canCreate = computed(() => hasBtn('inspectionRecord:create'))
  const canEdit = computed(() => hasBtn('inspectionRecord:edit'))
  const canDelete = computed(() => hasBtn('inspectionRecord:delete'))
  const canDetail = computed(() => hasBtn('inspectionRecord:detail'))

  const loading = ref(false)
  const tableData = ref([])
  const total = ref(0)
  const page = ref(1)
  const pageSize = ref(10)

  const searchForm = reactive({
    deviceName: '',
    inspector: '',
    inspectionType: '',
    inspectionResult: '',
    dateRange: []
  })

  const formVisible = ref(false)
  const detailVisible = ref(false)
  const currentRecord = ref(null)

  const getList = async () => {
    loading.value = true
    try {
      // 日期范围控件的值是 ['2026-10-23', '2026-10-24']，拆成后端要的两个字段
      const [start, end] = searchForm.dateRange || []
      const res = await getInspectionRecordList({
        page: page.value,
        pageSize: pageSize.value,
        deviceName: searchForm.deviceName || undefined,
        inspector: searchForm.inspector || undefined,
        inspectionType: searchForm.inspectionType || undefined,
        inspectionResult: searchForm.inspectionResult || undefined,
        inspectionTimeStart: start || undefined,
        inspectionTimeEnd: end || undefined
      })
      // [注意] 请求失败时拦截器已经把 msg 弹出来了，这里只负责不覆盖旧数据
      if (res.code !== 0) return
      tableData.value = res.data.list || []
      total.value = res.data.total || 0
    } finally {
      loading.value = false
    }
  }

  const handleSearch = () => {
    page.value = 1
    getList()
  }

  const handleReset = () => {
    searchForm.deviceName = ''
    searchForm.inspector = ''
    searchForm.inspectionType = ''
    searchForm.inspectionResult = ''
    searchForm.dateRange = []
    page.value = 1
    getList()
  }

  const handleCurrentChange = (val) => {
    page.value = val
    getList()
  }

  const handleSizeChange = (val) => {
    pageSize.value = val
    page.value = 1
    getList()
  }

  const handleCreate = () => {
    currentRecord.value = null
    formVisible.value = true
  }

  const handleEdit = (row) => {
    currentRecord.value = { ...row }
    formVisible.value = true
  }

  const handleDetail = (row) => {
    currentRecord.value = { ...row }
    detailVisible.value = true
  }

  // 详情页里点「编辑」：先关详情再开表单，两个弹窗不叠在一起
  const handleEditFromDetail = () => {
    detailVisible.value = false
    formVisible.value = true
  }

  const handleFormSuccess = () => {
    getList()
  }

  const handleDelete = async (row) => {
    try {
      await ElMessageBox.confirm(
        `确定要删除「${row.deviceName || '该记录'}」的巡检记录吗？`,
        '删除确认',
        {
          type: 'warning',
          confirmButtonText: '确定',
          cancelButtonText: '取消'
        }
      )
    } catch (e) {
      // 用户点了取消，ElMessageBox 会 reject，这里必须吞掉，不能当错误处理
      return
    }

    // [注意] 主键字段名是大写 ID（后端 GVA_MODEL 的 json tag 就是 "ID"），
    // 写成 row.id 会拿到 undefined，后端只能回一句"ID不能为空"
    const res = await deleteInspectionRecord({ id: row.ID })
    if (res.code !== 0) return
    ElMessage.success('删除成功')
    // 删掉当前页最后一条时回退一页，避免停在空白页
    if (tableData.value.length === 1 && page.value > 1) {
      page.value -= 1
    }
    getList()
  }

  onMounted(() => {
    getList()
  })
</script>

<style lang="scss" scoped>
  .thumb {
    width: 56px;
    height: 56px;
    border-radius: 4px;
    cursor: pointer;
  }

  .muted {
    color: var(--el-text-color-placeholder);
  }
</style>
