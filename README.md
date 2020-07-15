# thuhole-go-backend

[![Build Status](https://travis-ci.com/thuhole/thuhole-go-backend.svg?branch=master)](https://travis-ci.com/thuhole/thuhole-go-backend)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

[T大树洞](https://thuhole.com/) 的Golang后端。

## 安装方式
```bash
git clone https://github.com/thuhole/thuhole-go-backend
cd thuhole-go-backend
go install ./...
```

将`config-sample.json`复制到`config.json`之后修改参数即可运行。

`config.json`需要与可执行文件放在同一文件夹。参数包括：
- `salt_hashed`: 邮箱hash的加盐前缀的hash，用于校验stdin中的salt。
- `images_path`: 存储图片的文件夹。这一文件夹需要能被网页访问。
- `sql_source`: MySQL数据库配置
- `redis_source`: Redis数据库配置
- `pin_pids`: 置顶树洞号
- `disallow_report_pids`: 不允许举报的树洞号
- `admins_tokens`: 管理员的tokens
- `bannedEmailHashes`: 被封禁的邮箱哈希值列表
- `is_debug`: 是否启用debug模式
- `smtp_username`: 邮箱服务username
- `smtp_password`: 邮箱服务password
- `smtp_host`: 邮箱服务host
- `*_regex`: 自动折叠识别的正则表达式
- `allow_unregistered_access`: 是否允许未登录用户访问
- `subnets_whitelist`: 当`allow_unregistered_access`=`false`时，允许未登录游客访问的IP白名单
- `max_email_per_ip_per_day`: 每个IP每天允许尝试注册的次数上限
- `recaptcha_private_key`: Google reCAPTCHA v3密钥
- `recaptcha_threshold`: reCAPTCHA v3分数阈值
- `mmdb_path`: GeoLite2或GeoIP2的离线数据库位置
- `allowed_register_countries`: 邮箱注册时的IP所在国家白名单

当后端程序运行时编辑`config.json`可以热加载，不需要重启程序。

执行`go install ./...`时会在`$GOROOT/bin`生成三个文件可执行文件：
- `fallback-server`: 是一个非常简易的显示维护信息的小程序。可以使用Nginx的`error_page 502`配置为fallback server.
- `hole-login-api`: `/api_xmcp/login/*`的API服务
- `hole-services-api`: `/services/thuhole/api.php`以及`/api_xmcp/hole/system_msg`的API服务

## 部署方式

除了编译出的3个可执行文件外，部署一个后端需要使用以下服务：
- MySQL数据库。使用[./init_db.sql](./init_db.sql)初始化。
- Redis数据库。
- Nginx。使得`config.json`中`images_path`文件夹里的图片文件在网页上可访问到。
- 邮件服务。
- Google reCAPTCHA v3, 在https://www.google.com/recaptcha/intro/v3.html 注册。
- CDN服务，详见下文。

## CDN说明
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
- jsdelivr: 免费开源CDN，加速GitHub上的静态css/js文件。国内用了网宿CDN，速度和延迟都很理想。抗DDoS能力一流。
- Fastcache CDN: 付费CDN，加载动态资源和主页。10Tbps带宽储备，提供WAF、 DDoS 、CC防护。此CDN用于保障三网和教育网用户延迟和网速在可接受的范围内。
- Cloudflare CDN: 免费CDN，用于加载图片，大陆地区访问速度和延迟不稳定。抗DDoS能力一流。未来考虑将图片资源换成专业图床。

## License
[AGPL v3](./LICENSE)
