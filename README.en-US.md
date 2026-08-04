

# TOH Proxy

Proxy forwards your Minecraft server from the TCP protocol to the WebSocket protocol, supporting Proxy Protocol to restore real IP addresses.

By connecting via the `WS/WSS` protocol to a CDN (such as Tencent Cloud, Baidu Cloud, etc.) for proxying, it hides the server's real IP, providing strong DDOS protection.

This project consists of:

* Server
* Client

Players need to run the client proxy to connect to the server, or they can connect directly via a MOD.

---

## Demonstration

The server connection IP will display as the IP of local CDN nodes.
The following example shows a Baidu CDN node:

<p align="center">
  <img width="1270" height="638" src="https://github.com/user-attachments/assets/7f256cbb-c8cb-431a-a64c-b269e7902f3c" />
</p>

---

# Server Deployment

## Prerequisites

For mainland China deployments, you need to prepare:

* An ICP-registered domain name
* A CDN service (EdgeOne / Baidu CDN recommended)

Additionally, you must:

1. Change the Minecraft server listening address to `127.0.0.1`
2. Enable Proxy Protocol

### Velocity

Enable in `velocity.toml`:

```toml
proxy-protocol = true
```

### Paper

Enable Proxy Protocol in `paper-global.yml`.

### Spigot / Other Kernels

It is recommended to place a Velocity proxy in front before using this.

---

## Download Server

Download the version for your OS from the Releases page:

👉 [https://github.com/albert4719/toh-proxy/releases](https://github.com/albert4719/toh-proxy/releases)

Running it will automatically generate the `settings.toml` configuration file.

---

## Server Configuration

```toml
# WS proxy listening address (receives connections from the CDN)
listen = ':8080'

# WS route path
path = '/mc'

# Real internal TCP address of the MC server
backend = '127.0.0.1:25566'

# Header used by the CDN to pass the real IP
# For example: X-Forwarded-For / CF-Connecting-IP
header = 'X-Forwarded-For'

# Header used by the CDN to pass the real port
# Leave empty to use the default ephemeral port 50000
port_header = 'X-Forwarded-Port'

# Security verification key
# CDN requests must carry X-Toh-Token with a matching value to allow connection
# Leave empty to disable verification
password = ''
```

---

# CDN Configuration

## Recommended: Tencent Cloud EdgeOne (Free)

Console:

👉 [https://console.cloud.tencent.com/edgeone/](https://console.cloud.tencent.com/edgeone/)

### Configuration Steps

1. Add a site
2. Enter the origin address:

```text
服务器IP:8080
```

3. Enable in "Site Acceleration":

```text
WebSocket
```

Otherwise, it will not work.

---

### Regarding the Free Plan

Promotion page: [https://cloud.tencent.com/act/pro/eo-freeplan
](https://cloud.tencent.com/act/pro/eo-freeplan) 

Note:

* The free version is relatively slower
* Dynamic acceleration cannot be enabled
* It may slightly increase latency

---

## Baidu CDN

Console:

👉 [https://console.bce.baidu.com/cdn/](https://console.bce.baidu.com/cdn/)

Note:

* The acceleration type must be set to:

```text
DRCDN 动态加速
```

---

## HTTPS Notes

Enabling HTTPS is not recommended.

Reasons:

* It increases the number of requests
* It may incur additional CDN costs

---

# Client Usage

Download the `client` version for your OS and run it.

The configuration file will be automatically generated on the first run.

---

## Client Configuration

```toml
# Local listening address for the MC client to connect
local_listen = '127.0.0.1:3001'

# CDN WebSocket endpoint
# Use wss:// if the CDN has HTTPS/TLS enabled
cdn_url = 'ws://example.com:80/mc'

# Whether to skip TLS certificate verification
# Enable when using self-signed certificates
skip_tls = false
```

---

## Connecting to the Server

After configuration:

### Enter in the Minecraft client:

```text
127.0.0.1:3001
```

You can then enter the server.

---

# MOD Support

With the MOD, players can enter the server directly without needing to run an additional client program.

---

## Currently Supported
(Official) [https://github.com/albert4719/TohConnect](https://github.com/albert4719/TohConnect) \
(Third-party) [https://github.com/MooreFoss/ws2tcp](https://github.com/MooreFoss/ws2tcp)

---
