# Git Proxy Go

Go 实现的 Git 代理程序，支持 HTTPS、HTTP、SSH 三种协议的代码仓库请求代理。

## 功能特性

- **HTTP/HTTPS 代理**: 作为反向代理转发 Git 客户端请求到目标 Git 服务器
- **SSH 代理**: 支持 SSH 协议的 git clone/fetch/push 操作
- **认证支持**: 可配置 HTTP Basic 认证
- **路径控制**: 支持白名单/黑名单路径控制
- **生产级**: 完整的错误处理、日志记录、健康检查

## 快速开始

### 1. 配置文件

复制并编辑 `config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  http_port: 8080
  https_port: 8443
  cert_file: "server.crt"
  key_file: "server.key"

target:
  host: "github.com"
  port: 443
  username: ""
  password: ""
  ssh:
    enabled: true
    port: 22

logging:
  level: "info"
  format: "text"

proxy:
  timeout: 30
  max_connections: 100
  path_rewrite: true
```

### 2. 运行代理

```bash
# 使用默认配置
go run cmd/proxy/main.go

# 指定配置文件
go run cmd/proxy/main.go -config /path/to/config.yaml
```

### 3. 使用代理

```bash
# HTTP/HTTPS 协议
git clone http://localhost:8080/owner/repo.git
git clone https://localhost:8443/owner/repo.git

# SSH 协议 (需要配置)
git clone ssh://git@localhost:22/owner/repo.git
```

## 配置说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `server.host` | 监听地址 | `0.0.0.0` |
| `server.http_port` | HTTP 端口 | `8080` |
| `server.https_port` | HTTPS 端口 | `8443` |
| `server.cert_file` | TLS 证书文件 | - |
| `server.key_file` | TLS 私钥文件 | - |
| `target.host` | 目标 Git 服务器 | `github.com` |
| `target.port` | 目标端口 | `443` |
| `target.username` | 认证用户名 | - |
| `target.password` | 认证密码 | - |
| `target.ssh.enabled` | 启用 SSH 代理 | `true` |
| `target.ssh.port` | SSH 端口 | `22` |
| `logging.level` | 日志级别 | `info` |
| `proxy.timeout` | 请求超时(秒) | `30` |
| `proxy.path_rewrite` | 启用路径重写 | `true` |

## 项目结构

```
git-proxy-go/
├── cmd/proxy/
│   └── main.go          # 主程序入口
├── config.yaml          # 配置文件示例
├── go.mod               # Go 模块定义
└── README.md            # 本文件
```

## 构建

```bash
# 下载依赖
go mod tidy

# 构建
go build -o git-proxy cmd/proxy/main.go

# 运行
./git-proxy -config config.yaml
```

## License

MIT
# gitproxy
