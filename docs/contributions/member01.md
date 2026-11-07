# 成员 01：账目模型与校验

## 来源和实际状态

- 姓名、学号：待本人填写；本次 GitHub 上传账号：`Ther-ux`。
- 本次代码由 Codex 按用户确认的“直接实现任务1”生成并验证；成员本人的复审、修改及编程体验待本人补充。
- 实际目录：`C:/Users/zhang/Desktop/记账本/Account`。
- 本地起点：`a56c759bdb54168286a7a28a5718369155ea1c83`，与任务书基线相同；已阅读 README、go.mod、cmd/ledger/main.go、web/index.html。
- 开工时所在分支为 `feature/create-api`，已有 3 号未提交的 API 和材料；本次保留该分支及已有文件，只新增本文档和下述两个模型文件。
- 任务1上传到独立分支 [feature/record-model](https://github.com/Ther-ux/Account/tree/feature/record-model)，只包含本任务的三个文件；实际提交见[分支提交历史](https://github.com/Ther-ux/Account/commits/feature/record-model)。本地原工作分支不切换，其他成员文件保留原样。
- PR 和复审链接：待补充；未合并或发布版本。
- 验证日期：2026-11-07（北京时间）。

## 实现和边界

| 文件及关键位置 | 本次内容 |
| --- | --- |
| [record.go](../../internal/ledger/record.go) 的 `Record` | 五个字段及固定 JSON tag，金额为 `int64` 分 |
| 同文件的 `Validate(record Record) error` | 日期、正金额、类型及类别匹配检查，合法返回 nil |
| [record_test.go](../../internal/ledger/record_test.go) | 4 个测试函数、64 个表驱动子用例，验证规则、错误信息及 JSON 契约 |

输入是已经解码的 `ledger.Record` 值，输出为 nil 或可读错误。依赖只有 Go 标准库的 `time` 和 `errors`；没有文件读写、HTTP 或页面逻辑，也没有增加 ID、createdAt 或其他字段。

对接方导入 `personal-ledger/internal/ledger`。Record 的字段保持：

| Go 字段 | 类型 | JSON tag |
| --- | --- | --- |
| Date | string | date |
| Type | string | type |
| AmountCents | int64 | amountCents |
| Category | string | category |
| Note | string | note |

实现步骤是先定义结构体，再实现完整校验，最后加入边界测试和 JSON 编解码验证，没有保留永远成功的校验占位。

```text
用 time.Parse(time.DateOnly, record.Date) 解析日期
解析失败，或重新格式化的日期不等于原文本 → date 错误
金额 <= 0 → amountCents 错误
按 type 分支：
  income：category 只能为工资、奖金、其他收入
  expense：category 只能为饮食、交通、购物、其他支出
  其他类型 → type 错误
类别不属于当前类型 → category 错误
所有检查通过 → nil
```

日期的既定要求是真实的 `YYYY-MM-DD`。本实现选用标准库解析后再按 `time.DateOnly` 格式化比较：解析验证闰年和每月天数，比较原文本用于拒绝未补零的日期；同时拒绝首尾空白、时间后缀和其他分隔符。单用解析会接受未补零的日期日部分，因此需要额外比较。

金额严格为大于零的整数分，允许 `int64` 的最大正值。本模型接收整数，不处理元转分；JSON 小数、金额字符串及超出 `int64` 的数字由 3 号解码层拒绝。

校验按日期、金额、类型、类别顺序返回首个错误，不汇总多个错误。错误文本包含相应字段名和规则；目前不提供额外的导出错误类型。备注可以省略或为空，中文、空白及换行均原样保留。类型和类别精确匹配，不自动去空白或改大小写，以免把无效输入静默改成有效记录。

`Record` 以值传入，且只有字符串和整数，Validate 不修改字段，也不返回替代记录。这样调用方校验通过后保存和返回的仍是原始数据。

## 实际验证

环境：Windows/amd64、PowerShell、`go version go1.27.3 windows/amd64`。项目仍声明 `go 1.22`，没有改 go.mod 或添加依赖；未单独使用 Go 1.22 工具链复测。

本机 Go 不在默认 PATH 中，本次复用已有便携工具链：`C:/Users/zhang/.codex/tmp/ledger-member03-toolchain/go/bin/go.exe`。下面是实际执行的核心检查：

| 检查 | 实际结果 |
| --- | --- |
| 对两个模型文件执行 gofmt，随后用 gofmt -l 检查 | 通过，无待格式化文件 |
| `go test -v -count=1 -coverprofile=… ./...` | 通过；模型 64 个子用例，以及现有新增 API 的测试均通过 |
| `go tool cover -func=…` | Validate 语句覆盖率 100%；当前新增 API 100%；整个仓库 71.2%，服务入口没有测试覆盖 |
| `go vet ./...` | 通过，无诊断 |

覆盖率原始文件：`C:/Users/zhang/.codex/tmp/ledger-member01-coverage.out`，没有放入仓库。

在已配置 Go 的环境中，于仓库根目录复现：

```powershell
go test -v -count=1 -cover ./internal/ledger
go test ./...
go vet ./...
```

本机也可先将工具路径赋给任务变量，再执行：

```powershell
$ledgerGo = 'C:/Users/zhang/.codex/tmp/ledger-member03-toolchain/go/bin/go.exe'
& $ledgerGo test -v -count=1 -cover ./internal/ledger
& $ledgerGo test ./...
& $ledgerGo vet ./...
```

| 验收输入或条件 | 预期及实际结果 |
| --- | --- |
| 工资、奖金、其他收入，type=income | nil |
| 饮食、交通、购物、其他支出，type=expense | nil |
| 2024-02-29、2000-02-29、正常月末和年初年末 | nil |
| 2026-02-29、1900-02-29、2100-02-29、2026-11-31 | 非 nil，错误包含 date |
| 空日期、2026-1-7、2026-01-7、非法月日、时间后缀或首尾空白 | 非 nil，错误包含 date |
| 1 分、9223372036854775807 分 | nil，JSON 往返金额不丢失 |
| 0、-1、最小 int64 金额 | 非 nil，错误包含 amountCents |
| 空 type、transfer、EXPENSE、带空白的 type | 非 nil，错误包含 type |
| 空类别、未知类别、带空白的类别、全部收支类别反向错配 | 非 nil，错误包含 category |
| Record 零值 | 非 nil，首先报告 date |
| 省略备注、空备注、中文备注 | nil；JSON 使用五个固定字段，空备注编码为 note 空字符串 |
| 含空白和换行的中文备注 | nil，不清理备注 |
| 校验前后比较有效及无效输入 | 原始记录保持不变 |

本次全量测试使用真实模型和已有 3 号处理函数，没有模型替身。服务入口仍是基线的 501 占位；测试通过不代表存储、最近记录、统计或前端已接通。

## 接口交接

| 成员 | 交接项 |
| --- | --- |
| 2 号存储 | 使用 ledger.Record，AmountCents 为整数分；保存流程不应依赖校验会清理或修改输入 |
| 3 号新增 API | 保持解码后调用 ledger.Validate；非 nil 错误可用于 400 的 error；小数、字符串和数值溢出在解码层拒绝。本次现有 Create 测试已与真实模型一起通过 |
| 4、5 号查询 API | 导入统一 Record；日期规则为 Parse 加格式化比较，筛选参数是否缺省仍由各自模块判断 |
| 6、8 号前端 | 收入仅工资、奖金、其他收入；支出仅饮食、交通、购物、其他支出；金额录入和展示用元，传输用整数分；备注可为空 |
| 组长 | 归并时只纳入本任务的三个文件；将来在各模块合入后重新执行全量检查与真实服务联调 |

本次没有需要其他成员修改接口的交接项。若后续调整类别或错误结构，应先统一约定。

## 提交建议与博客材料

本次以一个真实功能阶段提交完整模型、测试和本文档，提交主题为 `feat(ledger): define record validation`。后续若确有边界修复，可按实际内容使用 `test(ledger): cover date and category boundaries` 等主题，不为凑提交次数拆分。其他成员的未提交文件不属于本次上传内容。

可保留的真实技术材料：Record 的五字段定义、日期解析后的格式化比较、收支类别分支、闰年及未补零日期测试、int64 金额的 JSON 往返测试、覆盖率文件和上述实测结果。

本次实际处理的问题：本机 Go 未加入 PATH，改用已有便携工具链完成验证；已有 3 号 API 依赖缺少的模型，新增真实模型后其测试一起通过。这些是本次执行事实，不替代成员本人的编程经历。

待本人补充：真实身份、本人复核和修改记录、实际 commit/PR 及复审链接、团队讨论参与者和过程、截图或视频、个人团队编程体验。尚未验证：真实文件保存、服务接线、完整页面操作及跨模块演示。
