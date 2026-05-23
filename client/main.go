package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gorilla/websocket"
	"github.com/pelletier/go-toml/v2" // 替换为 v2 库以支持 comment tag
)

// ClientConfig 对应 settings.toml 的结构
type ClientConfig struct {
	LocalListen string `toml:"local_listen" comment:"本地 MC 客户端连接的 TCP 监听地址"`
	CdnWSURL    string `toml:"cdn_url" comment:"CDN 的 WebSocket 接入点 (如果 CDN 启用了 HTTPS/TLS，请用 wss://)"`
	SkipTLS     bool   `toml:"skip_tls" comment:"是否跳过 CDN 的 TLS 证书验证 (使用自签名证书时开启)"`
}

var globalConfig ClientConfig

const configFile = "settings.toml"

// FindAvailablePort 从指定端口开始向上寻找可用的 TCP 端口
func FindAvailablePort(host string, startPort int) (int, error) {
	for port := startPort; port <= 65535; port++ {
		addr := net.JoinHostPort(host, strconv.Itoa(port))
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			err := listener.Close()
			if err != nil {
				return 0, err
			} // 关闭监听器，释放端口
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found")
}

func main() {
	// 1. 检查并生成默认配置文件
	ensureConfigFile()

	// 2. 加载 TOML 配置文件
	fileBytes, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("[FATAL] 读取配置文件失败: %v", err)
	}
	if err := toml.Unmarshal(fileBytes, &globalConfig); err != nil {
		log.Fatalf("[FATAL] 解析配置文件失败: %v", err)
	}

	// 3. 智能端口检查与自动替换
	host, portStr, err := net.SplitHostPort(globalConfig.LocalListen)
	if err != nil {
		log.Fatalf("[FATAL] 解析 local_listen 地址格式失败 (形如 127.0.0.1:3001): %v", err)
	}
	startPort, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("[FATAL] 解析端口号失败: %v", err)
	}

	// 寻找可用端口
	finalPort, err := FindAvailablePort(host, startPort)
	if err != nil {
		log.Fatalf("[FATAL] 无法找到可用的 TCP 端口: %v", err)
	}

	// 如果端口变了，更新全局变量并提示用户
	if finalPort != startPort {
		log.Printf("[WARN] 提示: 端口 %d 已被占用，自动切换至新端口 %d", startPort, finalPort)
		globalConfig.LocalListen = net.JoinHostPort(host, strconv.Itoa(finalPort))
	}

	// 4. 监听本地 TCP 端口
	listener, err := net.Listen("tcp", globalConfig.LocalListen)
	if err != nil {
		log.Fatalf("[FATAL] 本地 TCP 监听失败: %v", err)
	}
	defer listener.Close()

	log.Printf("[CLIENT] 客户端代理已启动！请在 MC 游戏中连接: %s", globalConfig.LocalListen)
	log.Printf("[CLIENT] 正在将流量转发至 CDN: %s", globalConfig.CdnWSURL)

	// 监听系统退出信号，优雅关闭
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("[CLIENT] 正在关闭客户端代理...")
		listener.Close()
		os.Exit(0)
	}()

	// 5. 死循环接收来自 MC 游戏客户端的连接
	for {
		gameConn, err := listener.Accept()
		if err != nil {
			log.Printf("[WARN] 接收游戏客户端连接失败: %v", err)
			continue
		}

		// 异步处理每一个玩家的本地连接
		go handleGameConnection(gameConn)
	}
}

// ensureConfigFile 检查配置文件是否存在，若不存在则使用 v2 encoder 自动生成（含注释）
func ensureConfigFile() {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Printf("[INFO] 未检测到配置文件，正在生成默认的 %s ...", configFile)

		// 定义客户端默认配置
		defaultConfig := ClientConfig{
			LocalListen: "127.0.0.1:3001",
			CdnWSURL:    "ws://example.com:80/mc",
			SkipTLS:     false,
		}

		// 使用 bytes.Buffer 和 toml.NewEncoder 进行结构化序列化
		var buf bytes.Buffer
		encoder := toml.NewEncoder(&buf)

		// 编码执行后，v2 库会提取 struct 里的 comment tag 自动转化为 # 注释
		if err := encoder.Encode(defaultConfig); err != nil {
			log.Fatalf("[FATAL] 序列化默认配置失败: %v", err)
		}

		// 写入文件
		if err := os.WriteFile(configFile, buf.Bytes(), 0644); err != nil {
			log.Fatalf("[FATAL] 自动创建配置文件失败: %v", err)
		}

		log.Printf("[INFO] 默认配置文件生成成功，程序将使用默认参数继续启动。")
	}
}

func handleGameConnection(gameConn net.Conn) {
	defer gameConn.Close()

	// 1. 配置 WebSocket Dial
	u, err := url.Parse(globalConfig.CdnWSURL)
	if err != nil {
		log.Printf("[ERR] CDN URL 解析错误: %v", err)
		return
	}

	dialer := websocket.DefaultDialer
	if u.Scheme == "wss" && globalConfig.SkipTLS {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	requestHeader := make(http.Header)
	requestHeader.Set("referer", "https://toh-client.example.com")
	// 添加或修改 User-Agent
	requestHeader.Set("User-Agent", "toh-client")

	// 2. 连接到 CDN 节点
	log.Printf("[CONNECT] 收到游戏客户端连接，正在连接 CDN WebSocket...")
	wsConn, _, err := dialer.Dial(u.String(), requestHeader)

	if err != nil {
		log.Printf("[ERR] 无法连接到 CDN 节点 (%s): %v", u.String(), err)
		return
	}
	defer wsConn.Close()

	log.Printf("[SUCCESS] 隧道建立成功！流量已成功连接至 CDN。")

	// 3. 双向管道数据转发
	errChan := make(chan error, 2)

	// 协程 A: 将 MC 游戏客户端的 TCP 字节流 -> 打包发给 CDN WS
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := gameConn.Read(buf)
			if n > 0 {
				err = wsConn.WriteMessage(websocket.BinaryMessage, buf[:n])
				if err != nil {
					errChan <- err
					return
				}
			}
			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	// 协程 B: 将来自 CDN WS 的数据 -> 解包发回给 MC 游戏客户端
	go func() {
		for {
			msgType, payload, err := wsConn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			if msgType == websocket.BinaryMessage || msgType == websocket.TextMessage {
				_, err = gameConn.Write(payload)
				if err != nil {
					errChan <- err
					return
				}
			}
		}
	}()

	<-errChan
	log.Printf("[DISCONNECT] 连接已断开。")
}
