# DIY 常见问题

## 忘记了 root 管理员密码怎么办？

如果你删除了数据库表后重新启动服务，root 用户的记录也会丢失，导致无法登录。

### 原因
`createRootAccountIfNeed()` 函数在 `model/main.go:68` 中已定义，但**从未在启动流程中被调用**，所以删除表后重启不会自动创建 root 账户。

### 解决方案

**方案一：在启动流程中补上 root 账户创建逻辑（推荐）**

编辑 `model/main.go`，找到 `CheckSetup()` 函数，在开头添加一行：

```go
func CheckSetup() {
    // 如果没有 root 用户，先创建一个
    createRootAccountIfNeed()

    setup := GetSetup()
    // ... 其余代码不变
}
```

重启后，若数据库中没有任何用户，系统会自动创建：
- **用户名：** `root`
- **密码：** `123456`

**方案二：手动往数据库插入 root 用户**

直接往 `users` 表插入一条记录（需先计算 bcrypt 密码哈希）。

---

## 日志详情清理后数据库文件没有变小

### 问题
设置了日志保留天数（如 1 天），自动清理任务也删除了大量记录，但数据库文件（`.db`）大小没有变化。

### 原因
使用 **SQLite** 作为数据库时，SQLite 执行 `DELETE` 删除记录后，被删除数据占用的空间只是被标记为"空闲页"，后续新数据会复用这些空间，**但 `.db` 文件本身不会缩小**。这是 SQLite 的正常行为。

MySQL / PostgreSQL 不存在此问题。

### 解决方案

需要先安装 `sqlite3` 工具：
```bash
sudo apt-get update && sudo apt-get install -y sqlite3
```

然后执行以下操作：

```bash
# 1. 查看当前记录数（确认删除是否生效）
sqlite3 one-api.db "SELECT COUNT(*) FROM logs; SELECT COUNT(*) FROM log_details;"

# 2. 执行 VACUUM 释放磁盘空间（会重建数据库文件）
sqlite3 one-api.db "VACUUM;"

# 3. 查看文件大小变化
ls -lh one-api.db
```

**注意事项：**
- `VACUUM` 执行期间数据库会被锁定，建议在访问量低的时候操作
- `VACUUM` 会创建临时文件，确保磁盘有足够空间（约等于原数据库文件大小）
- 可以通过前端"清除历史日志"功能删除旧日志后，再执行 `VACUUM`
