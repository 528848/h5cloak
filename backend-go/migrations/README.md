# 数据库迁移

使用 golang-migrate，文件名格式 `YYYYMMDDHHMMSS_name.up.sql` / `.down.sql`。

```powershell
# 生成一对空迁移
go run ./cmd/migrate create create_auth_tables
# 执行
go run ./cmd/migrate up
# 查看版本
go run ./cmd/migrate status
# 回滚一步
go run ./cmd/migrate down 1
```

约定：

- 每个迭代的表结构变更放在独立迁移文件中，不修改已合并的迁移。
- 所有商户数据表带 `merchant_id` 并建索引。
- 字段命名 snake_case，时间列使用 `DATETIME(3)`。
