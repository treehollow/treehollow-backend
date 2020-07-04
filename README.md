# thuhole-go-backend

[![Build Status](https://travis-ci.com/thuhole/thuhole-go-backend.svg?branch=master)](https://travis-ci.com/thuhole/thuhole-go-backend)

[T大树洞](https://thuhole.com/) 的Golang后端。

## 安装方式
```bash
git clone https://github.com/thuhole/thuhole-go-backend
cd thuhole-go-backend
go build
```

将`config-sample.json`复制到`config.json`之后修改参数即可运行。

`config.json`需要与可执行文件放在同一文件夹。参数包括：
- `salt`: 邮箱hash的加盐前缀
- `images_path`: 存储图片的文件夹。这一文件夹需要能被网页访问。
- `sql_source`: MySQL数据库配置
- `pin_pids`: 置顶树洞号
- `disallow_report_pids`: 不允许举报的树洞号
- `admins_tokens`: 管理员的tokens
- `bannedEmailHashes`: 被封禁的邮箱哈希值列表
- `is_debug`: 是否启用debug模式
- `smtp_username`: 邮箱服务username
- `smtp_password`: 邮箱服务password
- `smtp_host`: 邮箱服务host

当后端程序运行时编辑`config.json`可以热加载，不需要重启程序。

[./fallback_server/main.go](./fallback_server/main.go)是一个非常简易的显示维护信息的小程序。可以使用Nginx配置为fallback server.

## 部署方式

部署一个后端需要使用以下服务：
- 此程序
- MySQL数据库。使用[./init_db.sql](./init_db.sql)初始化。
- Nginx。使得`config.json`中`images_path`文件夹里的图片文件在网页上可访问到。
- 邮件服务。

## 部署说明
为了降低成本，目前的网站使用了复杂的CDN结构，如图：
```
+---------------------------+
|Frontend website           |          CDN Level                                      Server level
|                           |
|  +---------------------+  |     +-------------------+                       +-----------------------------------+
|  |                     |  |     |                   |                       |                                   |
|  |   css/js resources  +--------> Free jsdelivr CDN +----------------------->    GitHub Repo gh-pages Branch    |
|  |                     |  |     |                   |                       |                                   |
|  +---------------------+  |     +-------------------+                       +----------------+------------------+
|                           |                                                                  |
|                           |                                                                  | Git pull
|                           |                                                                  |
|                           |                                     +-----------------------------------------------+
|  +---------------------+  |     +-------------------+           |Backend server              |                  |
|  |                     |  |     |                   |           |                            |                  |
|  |     Index page      |  |     |                   |           |   +-------+ +--------------v---------------+  |
|  |          &          +-------->                   +--------------->       +->                              |  |
|  |  service worker.js  |  |     |                   |           |   |       | |        Frontend folder       |  |
|  |                     |  |     |                   |           |   |       | |                              |  |
|  +---------------------+  |     |   Fastcache CDN   |           |   |       | +------------------------------+  |
|                           |     |                   |           |   |       |                                   |
|  +---------------------+  |     |                   |           |   |       | +------------------------------+  |
|  |                     |  |     |                   |           |   |       | |                              |  |
|  |     Dynamic API     +-------->                   +---------------> Nginx +->         Go backend           |  |
|  |                     |  |     |                   |           |   |       | |                              |  |
|  +---------------------+  |     +-------------------+           |   |       | +------------------------------+  |
|                           |                                     |   |       |                                   |
|  +---------------------+  |     +-------------------+           |   |       | +------------------------------+  |
|  |                     |  |     |                   |           |   |       | |                              |  |
|  |       Images        +-------->  Cloudflare CDN   +--------------->       +->        Images folder         |  |
|  |                     |  |     |                   |           |   |       | |                              |  |
|  +---------------------+  |     +-------------------+           |   +-------+ +------------------------------+  |
|                           |                                     |                                               |
+---------------------------+                                     +-----------------------------------------------+
```
一共用到了3个CDN：
- jsdelivr: 免费开源CDN，加速GitHub上的静态css/js文件。国内用了网宿CDN，速度较快。
- Fastcache CDN: 付费CDN，加载动态资源和主页。此CDN用于保障三网用户延迟和网速在可接受的范围内。
- Cloudflare CDN: 免费CDN，用于加载图片。未来考虑将图片资源换成专业图床。

## License
[GPL v3](./LICENSE)
