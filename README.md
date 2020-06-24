# thuhole-go-backend

[T大树洞](https://thuhole.com/) 的Golang后端。

MySQL使用[./init_db.sql](./init_db.sql)初始化。

## 构建方式
```bash
cd src

# 安装依赖
go get ./...

# 编译
go build
```

将`config-sample.json`复制到`config.json`之后修改参数即可运行。

`config.json`需要与可执行文件放在同一文件夹。参数包括：
- `salt`: 邮箱hash的加盐前缀
- `images_path`: 存储图片的文件夹。这一文件夹需要能被网页访问。
- `sql_source`: MySQL数据库配置
- `pinned_pids`: 置顶树洞号
- `report_whitelist_pids`: 不允许举报的树洞号
- `report_admin_tokens`: 管理员的tokens，管理员举报树洞=直接禁言
- `bannedEmailHashed`: 被封禁的邮箱哈希值列表
- `mailgun_key`: 邮件服务mailgun的API Key
- `mailgun_domain`: 邮件服务mailgun的domain
- `is_debug`: 是否启用debug模式

当后端程序运行时编辑`config.json`可以热加载，不需要重启程序。

[./fallback_server/main.go](./fallback_server/main.go)是一个非常简易的显示维护信息的小程序。可以使用Nginx配置为fallback server.

## License
[GPL v3](./LICENSE)
