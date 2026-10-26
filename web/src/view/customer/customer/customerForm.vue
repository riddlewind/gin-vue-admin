
<template>
  <div>
    <div class="gva-form-box">
      <el-form :model="formData" ref="elFormRef" label-position="right" :rules="rule" label-width="80px">
        <el-form-item label="名字:" prop="firstName">
    <el-input v-model="formData.firstName" :clearable="true" placeholder="请输入名字" />
</el-form-item>
        <el-form-item label="姓氏:" prop="lastName">
    <el-input v-model="formData.lastName" :clearable="true" placeholder="请输入姓氏" />
</el-form-item>
        <el-form-item label="地址:" prop="address">
    <el-input v-model="formData.address" :clearable="true" placeholder="请输入地址" />
</el-form-item>
        <el-form-item label="电话:" prop="telephone">
    <el-input v-model="formData.telephone" :clearable="true" placeholder="请输入电话" />
</el-form-item>
        <el-form-item label="性别:" prop="gender">
    <el-select v-model="formData.gender" placeholder="请选择性别" style="width:100%" filterable :clearable="true">
       <el-option v-for="item in genderOptions" :key="item" :label="item" :value="item" />
    </el-select>
</el-form-item>
        <el-form-item label="启用状态:" prop="active">
    <el-select v-model="formData.active" placeholder="请选择启用状态" style="width:100%" filterable :clearable="true">
       <el-option v-for="item in activeOptions" :key="item" :label="item" :value="item" />
    </el-select>
</el-form-item>
        <el-form-item label="邮箱:" prop="email">
    <el-input v-model="formData.email" :clearable="true" placeholder="请输入邮箱" />
</el-form-item>
        <el-form-item>
          <el-button :loading="btnLoading" type="primary" @click="save">保存</el-button>
          <el-button type="primary" @click="back">返回</el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import {
  createCustomer,
  updateCustomer,
  findCustomer
} from '@/api/customer/customer'

defineOptions({
    name: 'CustomerForm'
})

// 自动获取字典
import { getDictFunc } from '@/utils/format'
import { useRoute, useRouter } from "vue-router"
import { ElMessage } from 'element-plus'
import { ref, reactive } from 'vue'


const route = useRoute()
const router = useRouter()

// 提交按钮loading
const btnLoading = ref(false)

// 性别 / 启用状态的候选值，必须与 customer.vue 保持一致
const genderOptions = ['男', '女', '未知']
const activeOptions = ['启用', '停用']

const type = ref('')
const formData = ref({
            firstName: '',
            lastName: '',
            address: '',
            telephone: undefined,
            gender: null,
            active: null,
            email: '',
        })
// 验证规则
const rule = reactive({
               firstName : [{
                   required: true,
                   message: '',
                   trigger: ['input','blur'],
               }],
               lastName : [{
                   required: true,
                   message: '',
                   trigger: ['input','blur'],
               }],
               address : [{
                   required: true,
                   message: '',
                   trigger: ['input','blur'],
               }],
               telephone : [{
                   required: true,
                   message: '',
                   trigger: ['input','blur'],
               }],
               gender : [{
                   required: true,
                   message: '',
                   trigger: ['input','blur'],
               }],
               active : [{
                   required: true,
                   message: '',
                   trigger: ['input','blur'],
               }],
               email : [{
                   required: true,
                   message: '',
                   trigger: ['input','blur'],
               }],
})

const elFormRef = ref()

// 初始化方法
const init = async () => {
 // 建议通过url传参获取目标数据ID 调用 find方法进行查询数据操作 从而决定本页面是create还是update 以下为id作为url参数示例
    if (route.query.id) {
      const res = await findCustomer({ ID: route.query.id })
      if (res.code === 0) {
        formData.value = res.data
        type.value = 'update'
      }
    } else {
      type.value = 'create'
    }
}

init()
// 保存按钮
const save = async() => {
      btnLoading.value = true
      elFormRef.value?.validate( async (valid) => {
         if (!valid) return btnLoading.value = false
            let res
           switch (type.value) {
             case 'create':
               res = await createCustomer(formData.value)
               break
             case 'update':
               res = await updateCustomer(formData.value)
               break
             default:
               res = await createCustomer(formData.value)
               break
           }
           btnLoading.value = false
           if (res.code === 0) {
             ElMessage({
               type: 'success',
               message: '创建/更改成功'
             })
           }
       })
}

// 返回按钮
const back = () => {
    router.go(-1)
}

</script>

<style>
</style>
