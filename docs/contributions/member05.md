# 成员 05 收支统计 API 接入说明

姓名、学号、GitHub 账号、本人复核说明、commit/PR 和个人体验：由实际成员填写。本文件记录工具辅助生成的参考实现与实际测试，不代填个人贡献。

## 上传文件

将压缩包直接解压到已有 Account 仓库根目录，保留 internal/ 和 docs/ 目录层级；不要额外套一层“成员05”文件夹。本包不包含或覆盖其他成员代码。

- internal/httpapi/stats.go
- internal/httpapi/stats_test.go
- docs/contributions/member05.md

## 与前面部分的接口

依赖 1 号提供 personal-ledger/internal/ledger 包，其中 Record 为 Date string、Type string、AmountCents int64、Category string、Note string。JSON 字段与原 README 完全一致。本包不复制 Record，避免与 1 号产生重复声明。

由 3 号在创建同一个 JSONStore 后接线：

```go
// 已有 import "personal-ledger/internal/httpapi"
api.HandleFunc("GET /api/stats", httpapi.Stats(records.List))
```

精确函数签名：func Stats(listRecords func() ([]ledger.Record, error)) http.HandlerFunc。无需修改 store，也不直接读取 JSON 文件。

GET /api/stats 的可选参数为 from、to、category、minCents、maxCents。日期与金额区间包含端点，全部条件 AND；缺省或单个空值表示不筛选。非空未知类别、重复的已知参数、坏日期、负数/小数/溢出金额或倒置区间返回 400。读取失败、无效存量金额或统计 int64 溢出返回 500。错误为 {"error":"说明"}，不会泄漏存储路径。

成功返回 {"incomeCents":300000,"expenseCents":3000,"balanceCents":297000}，三个值均为整数分，无外层 data；无匹配返回三个 0。辅助标识符均使用 stats 前缀，避免同包重名。

## 实际验证

环境：Windows amd64，Go 1.27.3；源码保持 Go 1.22 标准库 API 兼容。
- go test ./...：通过（在含共享 Record 及独立联调入口的验证工作区）。
- go vet ./...：通过。
- go build ./cmd/demo567：通过。
- go test -cover ./internal/httpapi：统计模块语句覆盖率 100.0%。
- 覆盖全量、七类筛选语义、日期/金额端点、单边及 AND 条件、无匹配、空值、非法查询、读取失败、错误类型/金额和 int64 累加溢出。
- 独立真实 JSON 联调：收入 300000 分、两笔支出 1250/1750 分，结果为 300000/3000/297000；交通筛选为 0/1750/-1750。
- go test -race ./... 因当前环境未启用 CGO 而未执行成功，不计为通过。

原负责人接入后运行 go test ./internal/httpapi 和 go test ./...。如果 ledger 包尚未合入，编译会提示缺少该依赖；应先合入 1 号公共模型，不要在 stats.go 再定义一份。

