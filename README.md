# thuhole-go-backend

[T大树洞](https://thuhole.tech/) 的Golang后端。

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
