# 成员 03：新增账目 API 与服务接线

## 身份和来源

- 姓名、学号、GitHub 账号：待本人填写。
- 实现与验证方式：本次由 Codex 辅助生成和验证；个人编程体验待本人填写。
- 参考仓库：<https://github.com/Ther-ux/Account>。
- 实际起点：`a56c759bdb54168286a7a28a5718369155ea1c83`；远端 `main` 仍为该基线。一号模块另已提交到 `feature/record-model`，本次据其真实代码完成接口验证。
- 本地分支：`feature/create-api`。
- 三号的 Commit、PR、复审链接：尚未创建，按用户要求完成后先不提交。
- 验证日期：2026-11-07（北京时间）。

## 本次交付与边界

| 文件 | 内容 | 状态 |
| --- | --- | --- |
| `internal/httpapi/create.go` | `Create` 和 `createWriteError` | 已实现 |
| `internal/httpapi/create_test.go` | 新增 API 的成功、解析、校验、存储失败测试 | 已实现；已与一号真实模型一起验证通过 |
| `docs/contributions/member03-wiring.patch` | `cmd/ledger/main.go` 的完整接线改动 | 已准备；未应用到实际入口 |
| `docs/contributions/member03.md` | 验证事实、依赖交接及后续验收方法 | 本文 |

一号已在 [feature/record-model 分支](https://github.com/Ther-ux/Account/tree/feature/record-model) 提交真实 `ledger` 模块，对应提交为 [`4ca5512`](https://github.com/Ther-ux/Account/commit/4ca5512)。本次获取远端更新，逐文件比较 Git blob，确认本地 `record.go`、`record_test.go` 和 `member01.md` 均与该提交完全一致；没有切换工作分支、替换一号文件或合并提交。一号尚未合入远端 `main`。

新增 API 已与真实 `ledger.Record` 和 `ledger.Validate` 兼容，源码无需调整。当前整个本地仓库的 `go test ./...`、`go vet ./...` 和入口构建均通过。API 测试仍按任务要求注入保存函数替身，不进行真实文件保存。

目前仍缺少 2 号 `store`、4 号 `Recent` 和 5 号 `Stats`。按用户确认的“先完成三号模块并准备待接线补丁”，实际 `main.go` 暂时保留原样；补丁不创建任何其他成员的实现。现有入口仍返回三个 501 占位，不能当作已经接通的应用演示。三号的改动继续保留为未提交文件。

## 模块设计及关键逻辑

输入是一个 JSON 账目对象，字段和 `ledger.Record` 一致；通过 `func(ledger.Record) error` 注入保存操作，不接触文件。对外签名保持：

```go
func Create(appendRecord func(ledger.Record) error) http.HandlerFunc
```

既定约定是调用 `ledger.Validate`，输入错误返回 400，存储错误返回 500，保存成功返回 201 和账目对象；所有响应使用 JSON。一次成功请求只调用一次保存函数。金额使用 `int64` 分，不进行浮点金额换算。

本次还采用任务书建议：拒绝第二个 JSON 文档和非法尾随内容。未知字段没有额外限制。HTTP 方法限制由 Go 1.22 的方法路由 `POST /api/records` 承担。

```text
解码到 *ledger.Record
  空请求、解码失败 → 400
  指针为空（顶层 null）→ 400
再次解码，必须得到 EOF
  有第二个文档或尾随语法错误 → 400
调用 ledger.Validate
  失败 → 400，原样保留校验原因
调用 appendRecord，且仅调用一次
  失败 → 500，包含存储错误原因
保存完成后写入 JSON 内容类型、201 和账目
```

关键位置：`Create` 中的指针解码用于识别顶层 `null`；第二次解码检查文档结束；`ledger.Validate` 与 `appendRecord` 之后才开始写成功响应。`createWriteError` 统一生成只有一个 `error` 字段的错误对象。测试辅助函数统一带 `create` 前缀，避免与 4、5 号同包冲突。

## 实际验证结果

运行环境为 Windows/amd64、PowerShell、`go version go1.27.3 windows/amd64`，项目仍使用原 `go.mod` 的 `go 1.22`，没有新增依赖。测试工具链为临时下载并校验官方 SHA-256 的便携版，没有修改系统 PATH。

一号提交后的本次验证使用真实模型，没有替换 `ledger` 包：

| 检查 | 实际结果 | 说明 |
| --- | --- | --- |
| 实际仓库 `go test -v -count=1 -coverprofile=… ./...` | 通过 | 三号 39 组用例、一号 64 个子用例全部通过 |
| `go tool cover -func=…` | 通过 | `Create`、`createWriteError`、`Validate` 语句覆盖率均为 100%；含未测试入口的总覆盖率为 71.2% |
| 实际仓库 `go vet ./...` | 通过 | 无诊断 |
| 实际仓库 `go build -o … ./cmd/ledger` | 通过 | 入口仍为未应用接线补丁的骨架，产物放在仓库外 |
| `git apply --check docs/contributions/member03-wiring.patch` | 通过 | 接线补丁仍可应用到当前入口 |
| `gofmt -l` 检查两个三号 Go 文件 | 通过 | 无待格式化文件 |

本次覆盖率原始文件为 `C:/Users/zhang/.codex/tmp/ledger-member03-real-model.cover`，构建产物为 `C:/Users/zhang/.codex/tmp/ledger-member03-real-model.exe`。这些文件没有放入仓库。未对一号代码或其他成员文件作改动，也未执行 commit、push、PR 或合并。

以下为**首次依赖未就绪时的历史验证**，保留用于说明先前的临时契约验证范围：

| 检查 | 实际结果 | 说明 |
| --- | --- | --- |
| 首次实际仓库 `go test ./...` | 当时失败；本次已通过 | 当时缺少 `personal-ledger/internal/ledger`，目前已使用一号真实代码 |
| 实际仓库 `go test ./cmd/ledger` | 通过 | 未应用接线补丁的入口可编译，无测试文件 |
| 隔离环境 `go test -v -coverprofile=create.cover ./internal/httpapi` | 通过 | 39 组用例，`create.go` 语句覆盖率 100% |
| 隔离环境应用接线补丁后 `go test -v ./...` | 通过 | 验证构造一次 store、三个处理函数共享实例、方法路由及静态首页 |
| 隔离环境 `go vet ./...` | 通过 | 含待接线入口 |
| 隔离环境 `go build -o ledger-contract.exe ./cmd/ledger` | 通过 | 仅验证契约签名和入口可构建 |
| 实际仓库 `git apply --check docs/contributions/member03-wiring.patch` | 通过 | 补丁可应用到当前基线 |

历史隔离环境使用临时 `ledger` 契约替身、内存存储替身和查询处理函数替身；这些文件没有放入交付仓库。查询替身只用于验证传入的回调来自同一存储实例，不实现正式的最近记录排序或统计筛选。历史隔离验证不能替代正式模块联调；本次已补齐真实模型验证，存储和两个查询模块仍待就绪。

本机验证脚本及日志保留在：

- `C:/Users/zhang/.codex/tmp/ledger-member03-verify.py`：首次隔离验证脚本；当前正式代码的复验使用下节命令。
- `C:/Users/zhang/.codex/tmp/ledger-member03-verification.json`：首次契约验证的完整命令、退出码和输出。
- `C:/Users/zhang/.codex/tmp/ledger-member03-contract-1xnol8wn`：本次隔离环境，含显式标记为临时替身的源码。
- `C:/Users/zhang/.codex/tmp/ledger-member03-toolchain/go/bin/go.exe`：本次使用的 Go 工具链。

## 可复现的处理函数验收

一号正式模块已在本地到位，可在仓库根目录复现本次正式代码验证：

```powershell
go test -v -cover ./internal/httpapi -run '^TestCreate'
go test ./...
go vet ./...
```

若 Go 尚未加入 PATH，本机可使用已有便携工具链：

```powershell
$ledgerGo = 'C:/Users/zhang/.codex/tmp/ledger-member03-toolchain/go/bin/go.exe'
& $ledgerGo test -v -count=1 -cover ./internal/httpapi -run '^TestCreate'
& $ledgerGo test ./...
& $ledgerGo vet ./...
```

| 输入或情景 | 预期 HTTP 结果 | 追加次数 |
| --- | --- | --- |
| 饮食支出 1250 分，中文备注“午饭” | 201，JSON 可解码为原账目 | 1 |
| 工资收入，省略备注 | 201，返回空备注 | 1 |
| 金额为 1 分，文档前后有空白 | 201 | 1 |
| 金额为 `9223372036854775807` | 201，无 `int64` 精度损失 | 1 |
| 空白、空请求、破损 JSON、顶层 `null`、数组或标量 | 400，只有非空 `error` 字段 | 0 |
| 金额为小数、`1250.0`、科学计数法、字符串、布尔、数组或对象 | 400 | 0 |
| 金额超出 `int64` 上下界 | 400 | 0 |
| 第一个对象后还有第二个文档或无效字符 | 400 | 0 |
| 不真实的日期、格式错误、类型错误、类别错误或类型类别不匹配 | 400，保留 `ledger.Validate` 的错误原因 | 0 |
| 金额缺省、`null`、0 或负数 | 400 | 0 |
| 追加函数返回 `disk full` | 500，包含具体原因，未提前写成功响应 | 1 |

测试会检查 JSON 内容类型、响应只有一个 JSON 文档、成功对象字段保持不变，以及所有非法输入都未触发保存。

## 接线补丁及接口交接

补丁仅修改 `cmd/ledger/main.go`：去掉仅供占位使用的 `io` 导入，在 `newHandler` 创建一次 `records := store.NewJSONStore("data/records.json")`，并注册：

```go
api.HandleFunc("POST /api/records", httpapi.Create(records.Append))
api.HandleFunc("GET /api/records/recent", httpapi.Recent(records.List))
api.HandleFunc("GET /api/stats", httpapi.Stats(records.List))
```

原 `-addr`、默认 `127.0.0.1:8080`、`web` 静态服务、API 路径和 5 秒 `ReadHeaderTimeout` 保持不变。一个正在运行的服务由同一个 handler 使用同一个存储实例。

| 对接成员 | 需要提供或确认的接口 |
| --- | --- |
| 1 号 | 已确认 `ledger.Record` 字段及 JSON tag、`Validate(Record) error` 完全兼容；真实模型与三号测试一起通过 |
| 2 号 | `NewJSONStore(string) *JSONStore`、`Append(ledger.Record) error`、`List() ([]ledger.Record, error)`；保存成功后才能返回 nil |
| 4 号 | `httpapi.Recent(func() ([]ledger.Record, error)) http.HandlerFunc`；不要修改 `main.go` |
| 5 号 | `httpapi.Stats(func() ([]ledger.Record, error)) http.HandlerFunc`；不要修改 `main.go` |
| 6 号 | POST JSON 数值金额；201 返回账目对象，400/500 返回 `error` 字段 |
| 组长 | 合入真实依赖后复验；本人信息、真实 commit/PR、演示截图及体验另补 |

一号已就绪；待 2、4、5 号正式模块也到位后，三号在仓库根目录按顺序执行：

```powershell
git apply --check docs/contributions/member03-wiring.patch
git apply docs/contributions/member03-wiring.patch
gofmt -w cmd/ledger/main.go
go test ./...
go vet ./...
go build -o ledger.exe ./cmd/ledger
```

补丁针对上述基线生成；如果入口已被其他改动更新，应先复核上下文。应用后重新确认只有一个 store 实例，不能把临时查询替身放入正式代码。

## 真实联调清单（尚未执行）

在独立临时工作目录放置正式二进制和 `web` 目录，使用空账本启动，避免影响真实账目。用 JSON 请求按顺序写入 2026-11-07 的工资 300000 分、饮食 1250 分、交通 1750 分，三个 POST 都应返回 201。

| 操作 | 预期 |
| --- | --- |
| GET `/` | 200，可访问原有首页 |
| GET `/api/records/recent` | 200，交通、饮食、工资的录入倒序 |
| GET `/api/stats` | 200，收入 300000、支出 3000、结余 297000 分 |
| GET `/api/stats?category=交通` | 收入 0、支出 1750、结余 -1750 分 |
| GET `/api/stats?from=2026-11-07&to=2026-11-07&minCents=1250&maxCents=1750` | 支出 3000 分，包含边界 |
| 停止服务，在同一临时工作目录重新启动并查询 | 原三条记录仍可查询，金额不变 |

真实文件保存、Windows 文件替换、重启读回、正式查询筛选和前端操作均待依赖就绪后验证。只有这些项目通过后，才能记录“最终三个 API 不再返回 501”。

## 提交阶段建议及博客材料

按实际完成阶段提交，不凑次数。本次没有执行 commit、push、PR、合并或发布。

1. 1 号依赖可用、创建 API 测试通过后，可提交 `create.go`、`create_test.go` 和本材料，示例：`feat(api): add record creation handler`。
2. 其他后端依赖到齐、接线及真实联调通过后，可提交实际 `main.go` 接线改动并更新本材料，示例：`feat(server): wire ledger handlers to shared store`。此时可移除已用完的待接线补丁。

可保留的真实技术事实：使用指针识别顶层 `null`；使用第二次解码防止尾随 JSON 被忽略；坚持保存成功后再发 201；初次依赖未就绪时先做独立契约验证，一号提交后逐文件核对来源并完成真实模型验证；其余依赖未到位，继续暂缓入口接线。初次契约验证日志及本次正式覆盖率文件见上文。

待本人补充：真实姓名和学号、真实 GitHub 身份、代码复核记录、实际 commit/PR 链接、团队讨论的参与人和过程、最终运行截图或视频、个人团队编程体验。本次未生成这些经历或证据。
