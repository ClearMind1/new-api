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
