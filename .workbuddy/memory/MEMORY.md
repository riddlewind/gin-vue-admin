# 项目长期记忆 · gin-vue-admin

## 新增一个后端业务模块的 5 个注册点（缺一个就报 undefined / 404）

1. `server/service/enter.go` → `ServiceGroup` 加 `XxxServiceGroup`
2. `server/api/v1/enter.go` → `ApiGroup` 加 `XxxApiGroup`
3. `server/router/enter.go` → `RouterGroup` 加 `Xxx`
4. `server/initialize/gorm_biz.go` → `bizModel()` 里 `AutoMigrate(&model.Xxx{})`
5. `server/initialize/router_biz.go` → `initBizRouter()` 里 `xxxRouter.InitXxxRouter(privateGroup)`

另有各包自己的 `enter.go`：`service/<mod>/enter.go`、`api/v1/<mod>/enter.go`、`router/<mod>/enter.go`。

## GORM 更新模型的正确写法（踩过坑，务必沿用）

前端传来的对象通常**不含 CreatedAt**，因此：

- ❌ `db.Save(&obj)` → 全字段覆盖，会把 `created_at` 写成 `0000-00-00`，
  MySQL 严格模式报 `Error 1292 Incorrect datetime value`。
- ❌ `db.Updates(&obj)` → 跳过所有零值字段，`status=0` 这类修改会静默失效。
- ✅ `db.Model(&X{}).Where("id = ?", id).Select("col1","col2",...).Updates(&obj)`
  → 只更新白名单列，且 Select 会强制写入零值。

## 接口通了但提示「权限不足」（code:7，HTTP 200）

GVA 的 casbin 策略在库里，不在代码里。新增接口必须两步：
1. `sys_apis` 插一行（path / description / api_group / method）；
2. `casbin_rule` 插一行（ptype='p', v0=角色ID, v1=path, v2=method）。

`casbin_rule` 表名不是 `sys_casbin_rule`。规则路径**不带** `/api` 前缀（少数带的是历史残留）。
enforcer 用 `sync.Once` 建一次并 `LoadPolicy()`，**改库后必须重启进程**才生效。

## 本机调试技巧：临时实例验证新代码

不动用户正在跑的 8888，另起一个实例验证：

```bash
cp config.yaml config.verify.yaml
sed -i 's/^    addr: 8888/    addr: 8899/; s/^    addr: 8889/    addr: 8890/' config.verify.yaml
sed -i 's/^    open-captcha: 0.*/    open-captcha: 100000/' config.verify.yaml  # 关掉验证码才能 curl 登录
go run . -c config.verify.yaml
# 验证完：taskkill 掉进程 + rm config.verify.yaml
```

注意 `config.yaml` 的键被 viper 按字母排序写过，`system.addr` 在文件靠后位置（mcp 在前）。
