# thuhole-go-backend

[![Build Status](https://travis-ci.com/thuhole/thuhole-go-backend.svg?branch=master)](https://travis-ci.com/thuhole/thuhole-go-backend)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![codebeat badge](https://codebeat.co/badges/d465de5a-345f-4fe8-9f23-ad089691d78d)](https://codebeat.co/projects/github-com-thuhole-thuhole-go-backend-master)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fthuhole%2Fthuhole-go-backend.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fthuhole%2Fthuhole-go-backend?ref=badge_shield)

[T大树洞](https://thuhole.com/) 的Golang后端。

## 安装方式
```bash
git clone https://github.com/thuhole/thuhole-go-backend
cd thuhole-go-backend

# (optional)upgrade all dependencies
go get -u ./...

# install packages
go install ./...
```

将`example.config.yml`复制到`config.yml`之后修改参数即可运行。

当后端程序运行时编辑`config.yml`可以热加载，不需要重启程序。

执行`go install ./...`时会在`$GOROOT/bin`生成2个可执行文件：
- `treehole-services-api`: `/contents/*`, `/send/*`, `/edit/*`的API服务
- `treehole-login-api`: `/security/*`的API服务

## 部署方式

除了编译出的2个可执行文件外，部署一个后端需要使用以下服务：
- MySQL数据库。
- Redis数据库。
- Nginx。使得`config.yml`中`images_path`文件夹里的图片文件在网页上可访问到。
- 邮件服务。
- Google reCAPTCHA v3, 在https://www.google.com/recaptcha/about/ 注册。
- Google reCAPTCHA v2, 在https://www.google.com/recaptcha/about/ 注册。

## Nginx配置示例

**⚠注意：出于安全考虑，此配置仅供参考，切勿用于生产环境。**

### img.thuhole.com.conf
```
server {
    listen 80;

    server_name img.thuhole.com;
    index index.html index.htm;

    root /path/to/images/folder/;

    location / {
        try_files $uri $uri/ =404;
    }
}
```
### thuhole.com.conf
```
server {
    listen 80;

    server_name thuhole.com;
    index index.html index.htm;

    root /path/to/webhole/folder/;

    location / {
        try_files $uri $uri/ =404;
    }

    location /contents/ {
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;

        proxy_pass http://127.0.0.1:8081;
        error_page 502 = @fallback;
    }

    location /send/ {
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;

        proxy_pass http://127.0.0.1:8081;
        error_page 502 = @fallback;
    }

    location /edit/ {
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;

        proxy_pass http://127.0.0.1:8081;
        error_page 502 = @fallback;
    }

    location /security/ {
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;

        proxy_pass http://127.0.0.1:8080;
        error_page 502 = @fallback;
    }

    location @fallback {
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;

        proxy_pass http://127.0.0.1:1234;
    }
}
```

## License
[AGPL v3](./LICENSE)


[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fthuhole%2Fthuhole-go-backend.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fthuhole%2Fthuhole-go-backend?ref=badge_large)
