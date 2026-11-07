# member03 提交包

生成日期：2026-11-07（北京时间）。
负责模块：新增账目 API 与服务接线。

## 包内文件

- internal/httpapi/create.go：新增账目处理函数。
- internal/httpapi/create_test.go：成功、输入错误、模型校验和保存失败测试。
- docs/contributions/member03.md：真实验证记录与接口交接说明。
- docs/contributions/member03-wiring.patch：待应用的 main.go 完整接线补丁。
- SHA256SUMS.txt：上述文件与本说明的 SHA-256 校验清单。

## 使用方法

1. 解压 ZIP，打开 member03 文件夹；将其中的 internal 和 docs 文件夹按相同路径复制到 Account 仓库。已有同名文件时先比较改动。
2. 仓库需要 Go 1.22 或更新版本及一号正式 ledger 模块。一号依赖来自 feature/record-model 的 4ca5512；本包不重复收录一号源码。
3. 在 Account 仓库根目录运行：

```powershell
go test -v -count=1 -cover ./internal/httpapi -run '^TestCreate'
go test ./...
go vet ./...
```

本次已使用一号真实模型完成验证：三号 39 组用例通过，create.go 语句覆盖率 100%；全项目测试、go vet 和入口构建通过。详情见 docs/contributions/member03.md。

## 待接线改动

二号 store、四号 Recent、五号 Stats 正式模块全部到位后，才在仓库根目录依次执行：

```powershell
git apply --check docs/contributions/member03-wiring.patch
git apply docs/contributions/member03-wiring.patch
gofmt -w cmd/ledger/main.go
go test ./...
go vet ./...
go build -o ledger.exe ./cmd/ledger
```

补丁对应基线 a56c759bdb54168286a7a28a5718369155ea1c83 的 main.go；入口更新后应先检查补丁上下文。当前实际 main.go 未应用补丁，三个 API 的入口仍为 501 占位。真实保存、查询、统计和重启读回联调待上述依赖就绪后完成。

## 提交状态

打包没有执行 git add、commit、push、PR、合并或发布；文件保持本地未提交状态。
姓名、学号、真实账号、个人复核记录与实际提交链接仍需本人补充。
包内保留仓库相对目录，不含 .git、其他成员源码、临时替身、Go 工具链、账本数据或二进制产物。
