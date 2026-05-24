# TOH Proxy

将您的 Minecraft 服务器从 TCP 协议代理转发到 WebSocket 协议，支持 Proxy Protocol 还原真实 IP。

通过 `WS/WSS` 协议可接入 CDN（如腾讯云、百度云等）进行代理，从而隐藏服务器真实 IP，实现高强度 DDOS 防护。

本项目分为：

* 服务端（Server）
* 客户端（Client）

玩家需运行客户端代理连接服务器，也可通过 MOD 直接连接。

---

## 效果展示

服务器连接 IP 将显示为各地 CDN 节点 IP。
以下示例为百度 CDN 节点：

<p align="center">
  <img width="1270" height="638" src="https://github.com/user-attachments/assets/7f256cbb-c8cb-431a-a64c-b269e7902f3c" />
</p>

---

# 服务端部署

## 前置要求

国内环境需准备：

* 已备案域名
* CDN 服务（推荐 EdgeOne / 百度 CDN）

同时需要：

1. 将 Minecraft 服务监听地址改为 `127.0.0.1`
2. 开启 Proxy Protocol

### Velocity

在 `velocity.toml` 中开启：

```toml
proxy-protocol = true
```

### Paper

在 `paper-global.yml` 中开启 Proxy Protocol。

### Spigot / 其他核心

建议前置一层 Velocity 后再使用。

---

## 下载服务端

从 Releases 页面下载对应系统版本：

👉 [https://github.com/albert4719/toh-proxy/releases](https://github.com/albert4719/toh-proxy/releases)

运行后会自动生成 `settings.toml` 配置文件。

---

## 服务端配置

```toml
# WS 代理监听地址（接收来自 CDN 的连接）
listen = ':8080'

# WS 路由路径
path = '/mc'

# MC 服务器真实内网 TCP 地址
backend = '127.0.0.1:25566'

# CDN 传递真实 IP 的 Header
# 例如：X-Forwarded-For / CF-Connecting-IP
header = 'X-Forwarded-For'

# CDN 传递真实端口的 Header
# 留空则默认使用临时端口 50000
port_header = 'X-Forwarded-Port'

# 安全校验密钥
# CDN 请求必须携带 X-Toh-Token 且值匹配才允许连接
# 留空则不校验
password = ''
```

---

# CDN 配置

## 推荐：腾讯云 EdgeOne（免费）

控制台：

👉 [https://console.cloud.tencent.com/edgeone/](https://console.cloud.tencent.com/edgeone/)

### 配置方法

1. 添加站点
2. 回源地址填写：

```text
服务器IP:8080
```

3. 在“站点加速”中开启：

```text
WebSocket
```

否则无法使用。

---

### 关于免费套餐

可在闲鱼搜索 EdgeOne 免费无限流量套餐。

注意：

* 免费版速度相对较慢
* 无法开启动态加速
* 会略微增加延迟

---

## 百度 CDN

控制台：

👉 [https://console.bce.baidu.com/cdn/](https://console.bce.baidu.com/cdn/)

注意：

* 加速类型必须选择：

```text
DRCDN 动态加速
```

---

## HTTPS 说明

不建议开启 HTTPS。

原因：

* 会增加请求数量
* 可能产生额外 CDN 费用

---

# 客户端使用

下载对应系统的 `client` 版本并运行。

首次运行后会自动生成配置文件。

---

## 客户端配置

```toml
# 本地 MC 客户端连接监听地址
local_listen = '127.0.0.1:3001'

# CDN WebSocket 接入点
# 若 CDN 启用了 HTTPS/TLS，请使用 wss://
cdn_url = 'ws://example.com:80/mc'

# 是否跳过 TLS 证书验证
# 使用自签名证书时开启
skip_tls = false
```

---

## 连接服务器

配置完成后：

### Minecraft 客户端填写：

```text
127.0.0.1:3001
```

即可进入服务器。

---

# MOD 支持

使用 MOD 后，玩家无需额外运行客户端程序即可直接进入服务器。

---

## 当前支持
(官方) [https://github.com/albert4719/TohConnect](https://github.com/albert4719/TohConnect) \
(第三方) [https://github.com/MooreFoss/ws2tcp](https://github.com/MooreFoss/ws2tcp)

---

