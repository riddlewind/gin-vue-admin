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

### 本机跑 go 命令必须先设这两个环境变量
bash 里 `go` 不在 PATH，且 `%LocalAppData%` 未定义会报
`build cache is required, but could not be located`：

```bash
export GOCACHE="C:/Users/MayoiKaze/AppData/Local/go-build"
export GOPATH="D:/GoPath"
/d/go/bin/go.exe build -o ./xxx.exe .
```

- 本包 `go test` 必须加 `-vet=off`：`auto_code_package_test.go` 有非恒定格式串的 vet 报错（历史遗留）。
- `gofmt -l` 会列出几乎所有文件（CRLF 导致），忽略即可。

### 【硬规则】后端服务由用户手动启动（2026-10-26 明确要求）
**不要再替用户起 8888 的后端服务**（`go run .` / `server.exe` 都不行），启动和重启一律由用户自己做。
需要新代码生效时，只做「编译通过 + 单测通过」的验证并提醒用户重启，不要自己起进程、也不要占 8888 端口。
临时实例验证（8899）仍可用，但用完必须停掉并清理配置。

### MySQL80 启动
服务是 Manual 启动，数据目录归 NETWORK SERVICE：普通权限 `Start-Service MySQL80` 报
Permission denied，直接跑 `mysqld.exe` 报 `Failed to set datadir ... errno 13`。
**只能让用户手动启动**，别浪费时间折腾。

### 路由前缀是空
`config.yaml` 里 `router-prefix: ""`，所以直连后端时路径是 `/autoCode/llmAuto`，
**不带** `/api`（`/api` 只是前端 dev 代理的约定）。public 组的路由不需要 token。

## AI 上游：OpenAI 兼容通道（`service/system/auto_code_llm_openai.go`）

- `autocode.ai-provider` 留空或写 `gva` → 官方 AiPath；其它任意值 → 兼容中转站。
- `ai-model` 给聊天类（analysisChat / workflowPromptChat）用；`ai-codegen-model` 给
  代码生成器（ai）用；`ai-vision-model` 给识图（eye）用，留空各自继承 `ai-model`。
  **eye 必须用支持 image 输入的模型**（deepseek-v4-flash 的 input_modalities 只有 text，
  拿它识图网关会静默丢图，模型只会回「我无法从图片中识别字段」）。
- 代码生成器 `mode=ai` 的返回会被前端直接 `JSON.parse`，服务端因此做了三层兜底：
  `stripJSONFence` → `json.Valid` 校验（不合法直接报错）→ `normalizeCodegenFieldTypes`
  把 `fields[].fieldType` 归一到 13 个合法枚举值（模型爱把 dataType 的 text/varchar 写进来）。
  **前端 fieldType 是固定下拉框，多一个非法值就选不中、模板类型映射也失效**，改白名单要两边同步。
- `mode=eye` 另有一层 `checkEyeDescription`：返回空或含拒绝话术就报错，避免前端把道歉文本当需求描述。
- **`package` 与 `abbreviation` 不能同名**（前端文案：package和结构体简称不可同名）。
  模板渲染 `var {{.Abbreviation}} {{.Package}}.{{.StructName}}`，同名会变成 `var book book.Book`，
  变量遮蔽同名包 → 生成的代码编译失败。服务端 `normalizeCodegenNames` 兜底改名（只改 abbreviation，
  不动 package——package 是页面下拉框从 `sys_auto_code_packages.package_name` 里选的，乱改会选不中）。
  另外 `abbreviation` 也不能等于 `structName`。
- 上游 429 会自动重试（8s→25s→40s，共 2 次）。该 key 额度极小：连续 2~3 次请求就打满，
  恢复一个名额约 20s；上游不给 `Retry-After`，只能盲试。重试用尽由 `rateLimitMessage()` 出人话提示。

## 本机小坑
- `curl 127.0.0.1:xxx` 会走沙箱代理，端口没监听时返回 **502** 而不是连接失败；
  判断服务起没起看 `netstat -ano | findstr ":8888"`，本地请求加 `--noproxy '*'`。
- 前端 dev server 是 `vite --host` 监听 `0.0.0.0:8080`，但 `127.0.0.1:8080` 被 CEF remote debugging 占了，
  用 localhost 访问会看到 CEF 页面，别误判成前端没起。
